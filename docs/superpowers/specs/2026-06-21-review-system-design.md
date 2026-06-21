# 审稿系统设计文档

**日期**: 2026-06-21
**版本**: v1.0
**状态**: 设计阶段

---

## 1. 项目概述

在现有博客社区平台上新增**编辑审核制**审稿系统。用户发布文章前必须提交管理员审核，管理员批准后方可公开发布；驳回后的文章可修改并重新提交。

### 核心需求

- 只有管理员角色（admin）可以审稿
- 审稿决策为二元制：通过（approved）/ 驳回（rejected）
- 审稿意见为非必填项
- 驳回后文章回到草稿状态，作者修改后可重新提交
- 全链路通知：作者收到提交/结果通知，管理员收到新待审文章通知
- 作者可见完整审稿历史和状态

---

## 2. 方案概述

**选择方案：扩展现有 content-service**，不新建微服务。审稿是文章生命周期的自然延伸，当前项目只有管理员审稿，逻辑不复杂，新增微服务的跨服务协调成本高于收益。

---

## 3. 数据模型

### 3.1 Article 模型修改

在 `backend/shared/models/articles.go` 中扩展：

```go
const (
    StatusDraft         = "draft"
    StatusPendingReview = "pending_review"  // 新增
    StatusPublished     = "published"
    StatusDeleted       = "deleted"
)
```

`Status` 字段的枚举值使用 `VARCHAR(20)`，无需 DDL 变更，直接兼容新旧枚举值。

### 3.2 新增 ReviewRecord 模型

在 `backend/shared/models/` 中新增文件：

```go
const (
    ReviewActionApproved = "approved"
    ReviewActionRejected = "rejected"
)

// ReviewRecord 审稿记录
type ReviewRecord struct {
    BaseModel
    ArticleID  string  `gorm:"index:idx_review_records_article;size:36" json:"article_id"`
    ReviewerID string  `gorm:"index:idx_review_records_reviewer;size:36" json:"reviewer_id"`
    Action     string  `gorm:"size:20" json:"action"`    // ReviewActionApproved / ReviewActionRejected
    Comment    *string `gorm:"type:text" json:"comment"`  // 审稿意见（可选，指针类型允许 NULL）
}
```

建表由 GORM AutoMigrate 自动完成，Go 模型即表结构定义。

---

## 4. 状态机

```
作者创建文章 → draft
    │
    └─→ 提交审稿 → pending_review
                        │
                   管理员审稿
                   ╱        ╲
            approved       rejected
              │               │
          published         draft
           (发布)      (退回，作者可修改重投)
```

### 状态转换规则

| 当前状态 | 允许操作 | 目标状态 |
|---------|---------|---------|
| `draft` | 提交审稿 | `pending_review` |
| `pending_review` | 管理员通过 | `published` |
| `pending_review` | 管理员驳回 | `draft` |
| `published` | 不可进入审稿流 | — |

### 后端校验规则

- `submit-review`：必须 `Status == draft`，且 `author_id == X-User-ID`
- `review`：必须 `Status == pending_review`，且操作者为管理员
- `EditArticle`：当 `Status == pending_review` 时禁止编辑（在现有 handler 中增加状态判断）
- 所有状态转换在 service 层做双重校验，handler 层只做参数解析

---

## 5. API 设计

遵循项目现有路由分组模式，审稿 API 分为作者端和管理员端。

### 5.1 作者端（需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/articles/:id/submit-review` | 提交审稿（draft → pending_review） |
| `GET` | `/api/articles/:id/review-history` | 查看该文章的审稿历史记录 |

### 5.2 管理员端（需管理员权限）

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/admin/reviews/pending` | 获取待审文章列表（分页：page, size） |
| `POST` | `/api/admin/articles/:id/review` | 执行审稿（approved/rejected） |

### 5.3 请求/响应格式

**POST /api/articles/:id/submit-review**
- 无需请求体，article_id 从 URL 取，user_id 从 `X-User-ID` header 取
- 响应：`{ "code": 200, "message": "已提交审稿", "data": { ...article } }`

**POST /api/admin/articles/:id/review**
- 请求体：`{ "action": "approved|rejected", "comment": "..." }`
- 响应：`{ "code": 200, "message": "审稿完成", "data": { ...reviewRecord } }`

**GET /api/articles/:id/review-history**
- 响应：`{ "code": 200, "data": [ { "id": "", "action": "rejected", "comment": "...", "reviewer_id": "", "created_at": "..." }, ... ] }`

**GET /api/admin/reviews/pending?page=1&size=10**
- 响应：`{ "code": 200, "data": [ ...articles ], "pagination": { "total": 50, "page": 1, "page_size": 10 } }`

---

## 6. 通知系统

### 6.1 事件定义

在 `backend/shared/events/types.go` 中新增：

```go
const (
    EventArticleSubmittedForReview = "article.submitted_for_review"
    EventArticleReviewRejected     = "article.review_rejected"
)
```

### 6.2 事件流

| 操作 | 发布事件 | 消费者 | 通知内容 |
|------|---------|--------|---------|
| 提交审稿 | `article.submitted_for_review` | notification-service → 通知所有管理员 | "[文章标题] 已提交审核，请处理" |
| 审核通过 | `article.published`（复用已有事件） | notification-service → 通知作者 | "你的文章 [文章标题] 已通过审核，已发布" |
| 审核驳回 | `article.review_rejected` | notification-service → 通知作者 | "你的文章 [文章标题] 已被退回" |

### 6.3 通知类型扩展

在 `notification-service` 中新增通知类型：
- `review_approved` — 审稿通过
- `review_rejected` — 审稿驳回
- `new_submission` — 新待审文章（仅管理员接收）

---

## 7. 前端设计

### 7.1 EditorView — 审稿状态与历史

在文章编辑页面增加审稿相关 UI：

- 当文章状态为 `draft` 时，在发布区域显示"提交审稿"按钮
- 当文章状态为 `pending_review` 时，显示"审核中"状态标签，禁用编辑
- 审稿历史列表展示每次审稿记录：审稿人、结果（通过/驳回）、意见、时间
- 被驳回的文章自动回到可编辑状态

### 7.2 AdminReviewView — 管理员审稿页（新增）

路由：`/admin/reviews`

- 待审文章列表（表格）：文章标题、作者、提交时间、操作
- 点击"审稿"弹出审稿对话框：
  - 文章全文预览（显示原始 Markdown 文本，后续迭代再引入渲染库）
  - 通过 / 驳回 选择
  - 意见输入框（可选）
  - 提交审稿结果按钮
- 分页支持

### 7.3 路由与导航

- 管理员左侧导航增加"审稿管理"入口
- 路由守卫：`/admin/reviews` 需要 `meta.requiresAdmin`
- 管理员导航栏显示待审文章数量徽标

### 7.4 通知适配

- 通知列表新增 `review_approved`、`review_rejected` 类型的显示
- 管理员通知中新增 `new_submission` 类型的显示

---

## 8. 部署影响汇总

| 组件 | 变更说明 |
|------|---------|
| `shared/models` | 新增 `ReviewRecord` 模型；`Article.Status` 新增 `pending_review` 枚举值 |
| `shared/events` | 新增 `article.submitted_for_review` 和 `article.review_rejected` 事件类型 |
| `content-service` | 新增审稿 handler/service/repository；发布审稿相关事件 |
| `notification-service` | 新增两个事件消费者；新增三种通知类型 |
| `api-gateway` | 新增审稿路由（作者 + 管理员） |
| `frontend` | 新增 `AdminReviewView`；修改 `EditorView` 审稿状态展示；修改导航栏 |
| **MySQL** | 新增 `review_records` 表（article 表无需 DDL 变更） |
| **RabbitMQ** | 主题交换机自动处理新的事件 routing key（无需手动配置 binding） |

---

## 9. 测试覆盖要求

| 层级 | 内容 | 最低覆盖率 |
|------|------|-----------|
| 单元测试 | ReviewRecord CRUD、状态机转换逻辑（非法转换应拒绝）、事件发布/消费 | 80% |
| 集成测试 | 提交审稿 → 管理员审稿 → 通过/驳回 完整链路 | 80% |
| E2E | 作者提交审稿 → 管理员看到待审列表 → 审稿通过 → 作者看到发布状态和通知 | 关键路径 |

---

## 10. 实现顺序

1. **数据层** — `shared/models` 新增 ReviewRecord，Article 状态枚举扩展
2. **event 定义** — `shared/events/types.go` 新增事件类型
3. **content-service 后端** — handler → service → repository 审稿逻辑实现 + 事件发布
4. **notification-service** — 新增消费者 + 新通知类型处理
5. **api-gateway 路由** — 注册审稿路由
6. **前端** — AdminReviewView 新增 + EditorView 修改 + 导航栏修改
7. **数据库迁移** — review_records 建表
8. **测试** — 单元 + 集成 + E2E
