# Context.Context 问题汇总

## 1. Server 层无超时配置

`ReadTimeout`、`WriteTimeout`、`IdleTimeout` 全项目未设置。`gin.Engine.Run()` 使用零值，`c.Request.Context()` 无 deadline，仅客户端断开连接时才会取消。

## 2. Handler 层未包装超时

部分 handler 传了 ctx，但都是 `c.Request.Context()` 原样透传，未用 `WithTimeout` 包装。慢查询可能无限期执行。

```go
// handler/article.go:56
ctx := c.Request.Context()
article, err := h.service.GetArticleDetail(ctx, articleID)
```

## 3. 大量 handler 未传 ctx

`CreateArticle`、`ListArticles`、`ListByCategory` 等方法完全不传 ctx，调用链末端用 `context.Background()` 创建空 context，放弃请求级超时和取消传播。

## 4. GORM 调用未传 ctx

部分 Repository 方法调用 `r.db.First(...)` 而非 `r.db.WithContext(ctx).First(...)`，ctx 链路中断。

```go
// repository/article.go:41
r.db.First(&article, "id = ?", id).Error  // 未传 ctx
```

## 5. 无服务端主动超时通知

慢请求只能依赖前端 Axios 10s 超时。用户收到泛化的 "Network Error"，无法区分"服务器处理超时"与"网络断开"。重试不安全（上一请求可能仍在执行）。

## 修复方向

| 优先级 | 事项 |
|--------|------|
| P0 | `main.go` 设置 `ReadTimeout`/`WriteTimeout` |
| P1 | Handler 层对 `c.Request.Context()` 做 `WithTimeout` 包装，按操作类型差异化（读 2s / 写 5s / 搜索 8s） |
| P1 | 所有 Repository 的 GORM 调用使用 `db.WithContext(ctx)` |
| P2 | 所有 Handler → Service 补齐 ctx 参数 |
| P2 | 超时返回结构化 `504 Gateway Timeout`，前端拦截器区分处理 |
