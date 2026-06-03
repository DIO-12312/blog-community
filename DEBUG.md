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
