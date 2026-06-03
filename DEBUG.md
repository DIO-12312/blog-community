# DEBUG.md

## 1. POST /api/articles 返回 401 "缺少用户身份"

**现象**: 浏览器携带 token 请求 POST /api/articles，网关返回 401。调试发现 token 已到达网关且解析成功，但下游 content-service 返回 "缺少用户身份"。

**根因**: `AuthMiddleware` 用 `c.Header("X-User-ID", ...)` 设置的是 HTTP **响应头**，而 `proxyTo` 用 `c.GetHeader("X-User-ID")` 读取的是 HTTP **请求头**。二者方向不同，用户身份从未转发给下游。

**修复** (`api-gateway/middleware/auth.go` + `api-gateway/routes/router.go`):
- AuthMiddleware: `c.Header(...)` → `c.Set("userID", ...)`
- proxyTo: `c.GetHeader(...)` → `c.Get("userID")`

---

## 2. content-service: Table 'blog.articles' doesn't exist

**现象**: POST /api/articles 返回 400 "Table 'blog.articles' doesn't exist"。

**根因**: `content-service/main.go` 中只有注释 `// 2. 执行数据库迁移`，没有实际调用 `db.AutoMigrate()`。其他服务（user、interaction、audit、notification）都有。

**修复** (`services/content-service/main.go`):
```go
db.AutoMigrate(&models.Article{}, &models.Category{})
```

---

## 3. content-service: Incorrect datetime value '0000-00-00'

**现象**: POST /api/articles 返回 400 "Incorrect datetime value: '0000-00-00' for column 'published_at'"。

**根因**: `PublishedAt time.Time` 零值在 MySQL 8.0 严格模式下不合法。草稿文章尚未发布，`published_at` 应为 NULL。

**修复** (`shared/models/articles.go` + `services/content-service/service/article.go`):
- `PublishedAt time.Time` → `PublishedAt *time.Time`
- `article.PublishedAt = now` → `article.PublishedAt = &now`

---

## 4. Docker 容器间网络不通

**现象**: API 网关代理后端服务时返回 502 Bad Gateway。

**根因**: `router.go` 中服务地址使用了 `localhost:8001`，但 Docker 容器间通信需使用服务名。

**修复** (`api-gateway/routes/router.go`):
```go
// 错误: "http://localhost:8001"
// 正确: "http://user-service:8001"
```

---

## 5. 登录成功后仍无法获取用户信息

**现象**: 登录成功但前端获取用户信息失败，显示 "缺少令牌"。

**根因**: `getProfile(id)` 请求 URL 与服务端路由不匹配，且 token 在验证 profile 成功前就写入了 localStorage。

**修复** (`frontend/src/api/index.ts` + `frontend/src/stores/user.ts`):
- `api.get('/users/${id}')` → `api.get('/users', { params: { id } })`
- 先拉取 profile 成功后再保存 token

---

## 6. notification-service 无 HTTP 接口

**现象**: 前端请求 `/api/notifications` 返回 404。

**根因**: notification-service 只有 MQ 消费者，没有 HTTP 服务器。

**修复** (`services/notification-service/main.go`): 添加 Gin HTTP 服务器监听 `:8004`，暴露通知相关路由。

---

## 7. Elasticsearch 未创建索引

**现象**: 启动后 ES 中没有 `articles` 索引，搜索功能不可用。

**根因**:
- ES 官方镜像不带 `analysis-ik` 中文分词插件，索引创建时指定 `ik_max_word` 分析器导致失败，只打了 warning 日志
- ES 启动慢，search-service 用 `service_started` 时 ES 还没就绪

**修复**:
- 创建 `elasticsearch/Dockerfile`，从 URL 安装 ik 插件
- `docker-compose.yml`: ES 改用自定义构建，添加 healthcheck，search-service 依赖改为 `service_healthy`
- search-service 已有 `EnsureIndex()`，现在能正常执行

---

## 8. 前端文章列表/搜索结果不显示

**现象**: `HomeView.vue` 报错 `Cannot read properties of undefined (reading 'length')`，列表为空。

**根因**: API 返回 `{ data: [...], pagination: { total } }`，经 axios 响应拦截器解析后，前端误用 `res.data.list` 取值。实际 `res.data` 就是数组本身，`res.pagination.total` 才是总数。

**修复** (`frontend/src/views/HomeView.vue`):
- `res.data.list` → `res.data || []`
- `res.data.total` → `res.pagination?.total || 0`
- 模板中 `articles.length` → `articles?.length || 0`，防止 null 崩溃

---

## 9. 文章发布后仍是草稿状态

**现象**: 前端点击"发布"按钮后，数据库文章状态仍为 `draft`。

**根因**: `POST /api/articles` 创建文章默认状态为 `draft`，需要再调用 `POST /api/articles/:id/publish` 才能发布。前端只调了 create 没调 publish。

**修复** (`frontend/src/api/index.ts` + `frontend/src/views/EditorView.vue`):
- api 层新增 `articleApi.publish(id)`
- EditorView 创建成功后自动调用 publish

---

## 10. 评论有数据但前端不显示用户名

**现象**: 数据库有评论但前端不显示，用户名也是空的。

**根因**:
- CommentList.vue 同样用 `res.data.list` 解析 API 响应（应为 `res.data`）
- Comment 模型只存 `user_id` 没有 `username`，查询返回的 username 始终为空

**修复**:
- CommentList.vue: `res.data.list` → `res.data`，`res.data.total` → `res.pagination?.total`
- Comment 模型添加 `Username` 字段，handler 创建评论时从 `X-Username` header 读取
- `GetByArticle` 响应填充 username
