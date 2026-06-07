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

---

## 11. API 网关高并发时大量 502

**现象**: 压测时 c=100 即产生 502 Bad Gateway，c=500 时 502 比例超过 80%。直连后端则无此问题。

**根因**: `router.go` 的 `proxyTo()` 中每次请求都调用 `httputil.NewSingleHostReverseProxy(target)`，创建新的 HTTP Transport 和连接池。每个请求用完即丢弃，连接无法复用，高并发时端口/连接耗尽。

**修复** (`api-gateway/routes/router.go`):
- 新增全局 `proxyCache` map，启动时预创建每个服务的单例反向代理
- `proxyTo` 直接复用缓存代理，仅设置 per-request 的 header 和 URL

```go
// 优化前: 每请求创建新 proxy
proxy := httputil.NewSingleHostReverseProxy(target)

// 优化后: 全局单例复用连接池
var proxyCache = initProxyCache()  // 启动时创建
proxy := proxyCache[serviceName]   // 直接复用
```

**效果**: 网关损耗从 50% 降至接近 0%，502 错误归零。

---

## 12. view_count 行级锁 — 文章详情高并发退化

**现象**: `GET /api/articles/:id` 高并发时 DB 返回 `Error 1040: Too many connections`，单次请求延迟 >2s，502 错误频繁。

**根因**: `IncrementViewCount` 每次执行 `UPDATE articles SET view_count = view_count + 1 WHERE id = ?`。对同一文章的高并发请求竞争同一行的排他锁，UPDATE 串行化，每个耗时 ~2s，连接全部堆积等待。

**修复** (`services/content-service/repository/article.go` + `service/article.go` + `main.go` + `shared/cache/redis.go` + `shared/cache/keys.go`):
- `IncrementViewCount`: DB UPDATE → `Redis INCR view_count:id`（内存原子操作，<1ms）
- `GetArticleDetail`: 合并 `DB基础值 + Redis实时增量` 作为返回的 view_count
- 新增 `SyncViewCounts`: 使用 Redis SCAN 扫描所有 `view_count:*` 键，批量写入 MySQL 后清除
- 新增 `StartViewCountSyncWorker`: 每 5 分钟自动执行同步
- `RedisClient` 新增 `GetInt64`、`ScanKeys` 方法

**效果**: 文章详情 QPS 从 807 提升至 21,887+，502 和 Error 1040 归零。

---

## 13. MySQL max_connections 不足

**现象**: 高并发时多个微服务报 `Error 1040: Too many connections`。

**根因**: MySQL 默认 `max_connections=151`。6 个微服务各建连接池，高并发时总连接数超出上限。

**修复** (`docker-compose.yml`):
```yaml
mysql:
  image: mysql:8.0
  command: --max_connections=500
```

---

## 14. 通知表缺少复合索引 — 高并发通知查询退化

**现象**: 10,000+ 条通知后，`GET /api/notifications` 查询延迟 >6s。

**根因**: `SELECT * FROM notifications WHERE user_id = ? ORDER BY created_at DESC LIMIT 10` 只有单列 `idx_notifications_user_id` 索引，需对所有匹配行做 filesort。

**修复**:
```sql
CREATE INDEX idx_notifications_user_created ON notifications(user_id, created_at DESC);
```
