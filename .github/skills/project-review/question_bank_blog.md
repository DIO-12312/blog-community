# 项目复习题库 — Blog Community

> 本题库按模块分章节，每道题标注难度（⭐基础 / ⭐⭐进阶 / ⭐⭐⭐深挖）和预估复习时长。
> 老师（Agent）根据用户当前复习章节顺序出题，不随机跳题，确保知识体系完整建立。

---

## 第 1 章：项目全景与架构设计

| # | 题目 | 难度 | 考察要点 | 参考答案要点 |
|---|------|------|---------|------------|
| 1-01 | 这个项目叫什么名字，整体要解决什么问题？ | ⭐ | 项目定位 | Blog Community，一个全栈博客社区平台；支持用户注册登录、发布文章、评论互动、关注通知、全文搜索等功能 |
| 1-02 | 项目整体采用了什么架构风格？各个微服务分别是什么？ | ⭐ | 架构设计 | 微服务架构；共7个服务：api-gateway(网关)、user-service(用户)、content-service(内容)、interaction-service(互动)、notification-service(通知)、search-service(搜索)、audit-service(审计) |
| 1-03 | API Gateway 在这个项目中扮演什么角色？它如何将请求路由到后端服务？ | ⭐⭐ | 网关设计 | 项目唯一入口，通过 `ServiceRegistry` 映射表将服务名→内部Docker URL；`httputil.ReverseProxy` 单例缓存实现代理转发；自动注入 `X-User-ID` 和 `X-Username` 头 |
| 1-04 | 项目为什么选择微服务架构而不是单体架构？具体有哪些好处和代价？ | ⭐⭐ | 架构决策 | 好处：独立部署、技术栈灵活、故障隔离、团队可并行开发；代价：网络延迟、数据一致性挑战、运维复杂度增加、调试困难 |
| 1-05 | 项目中每个微服务内部采用什么分层模式？各层职责是什么？ | ⭐ | 分层设计 | Handler(HTTP请求/响应) → Service(业务逻辑) → Repository(数据访问+GORM+Redis)；清晰职责边界，Repository封装数据访问细节 |
| 1-06 | 项目使用了哪些基础设施组件？各承担什么角色？ | ⭐⭐ | 基础设施 | MySQL 8.0(主数据库) + Redis 7(缓存/计数) + RabbitMQ 3(消息队列) + Elasticsearch(全文搜索)；Docker Compose统一编排 |
| 1-07 | 用户从前端发起一个"查看文章"请求，整个链路经过哪些关键步骤？ | ⭐⭐ | 端到端流程 | 浏览器 → Vite代理(dev) → API Gateway(8000) → JWT可选认证 → 代理到 content-service(8002) → 查Redis缓存 → 未命中查MySQL → 填充缓存 → 返回JSON |
| 1-08 | 项目中 shared 模块的设计目的是什么？包含了哪些核心组件？ | ⭐⭐⭐ | 模块共享 | 避免代码重复，提供统一基础能力；包含：database(MySQL连接池)、cache(Redis/SingleFlight)、models(GORM模型定义)、events(RabbitMQ发布订阅)、search(ES客户端)、middleware(CORS/Logger) |

---

## 第 2 章：API 网关与认证体系

| # | 题目 | 难度 | 考察要点 | 参考答案要点 |
|---|------|------|---------|------------|
| 2-01 | JWT 认证在这个项目中是如何工作的？从登录到后续请求的完整流程是怎样的？ | ⭐ | 认证流程 | 登录：user-service验证账号密码→生成JWT(HMAC-SHA256,24h)→返回token；后续请求：前端Axios拦截器自动附加Bearer token→Gateway验证→解析claims→注入X-User-ID/X-Username头→转发后端 |
| 2-02 | API Gateway 如何区分"公开路由"和"需要认证的路由"？两种路由的认证行为有什么不同？ | ⭐⭐ | 路由设计 | 公开路由(注册/登录/文章列表/详情/搜索)使用"可选认证"——token存在则解析、不存在也放行；私有路由(发布/编辑/评论/点赞/关注)使用"强制认证"——无有效token返回401 |
| 2-03 | JWT 的 claims 中包含哪些信息？为什么选择在 Gateway 解析 JWT 而不是各个服务各自解析？ | ⭐⭐ | 职责分离 | Claims包含userID、username；Gateway统一解析避免各服务重复实现认证逻辑；后端服务只需信任头部字段，职责单一 |
| 2-04 | 前端如何处理 JWT 过期或无效的情况？具体代码在哪里？ | ⭐⭐ | 前端认证 | `api/index.ts` 中Axios响应拦截器捕获401→调用`useUserStore().logout()`清空token和用户信息→`router/index.ts`导航守卫`beforeEach`检测未登录→重定向到登录页 |
| 2-05 | 速率限制是如何实现的？具体参数是什么？为什么放在 Gateway 层？ | ⭐⭐ | 限流设计 | `middleware/ratelimit.go` 使用Token Bucket算法，每IP 10 QPS/突发20；Gateway作为唯一入口天然适合做全局限流，后端服务无需各自实现 |
| 2-06 | `proxyCache`（反向代理单例缓存）解决了什么问题？如果不做这个优化会怎样？ | ⭐⭐⭐ | 连接池优化 | 每次请求创建新的`httputil.ReverseProxy`会导致连接无法复用；预初始化单例代理使得底层HTTP Transport连接池被充分利用，减少TCP握手和TLS开销 |
| 2-07 | 为什么反向代理时要重写请求的 `Scheme` 和 `Host`？不重写会怎样？ | ⭐⭐ | 代理细节 | 原始请求的Scheme可能是http、Host是localhost:8000；必须改为目标服务的内部Docker URL（如http://content-service:8002），否则后端服务收到错误的Host头，可能导致路由匹配失败 |
| 2-08 | 密码存储使用了什么算法？为什么选择它？ | ⭐ | 安全设计 | bcrypt加密；自带盐值、计算速度慢（抗暴力破解）、成熟稳定；`golang.org/x/crypto/bcrypt`标准库实现 |

---

## 第 3 章：用户服务

| # | 题目 | 难度 | 考察要点 | 参考答案要点 |
|---|------|------|---------|------------|
| 3-01 | User 模型有哪些核心字段？主键使用什么策略？ | ⭐ | 数据模型 | User模型含username/email/password_hash/avatar/bio字段；`BaseModel`嵌入UUID主键（GORM BeforeCreate钩子自动生成）、CreatedAt/UpdatedAt/DeletedAt(软删除) |
| 3-02 | 为什么选择 UUID 而不是自增 ID 作为主键？有什么优缺点？ | ⭐⭐ | ID设计 | 优点：分布式友好（无需中心化ID生成）、安全性（不可枚举）、导入导出不冲突；缺点：存储空间大(36字节)、索引效率略低于自增整数、不可排序 |
| 3-03 | Follow 模型的两个外键之间有什么约束？如何防止重复关注？ | ⭐⭐ | 数据设计 | `FollowerID`和`FollowingID`组成复合主键(composite PK)，数据库层面保证唯一约束；双重唯一键防止同一条关注关系重复出现 |
| 3-04 | 获取关注者列表和粉丝列表用的是什么查询方式？是否需要分页？ | ⭐ | 查询设计 | GORM的`Preload`或JOIN查询，按`Follow`表筛选；支持分页参数(page/limit)；`GetFollowers`和`GetFollowings`是两个独立查询 |
| 3-05 | 用户关注操作触发了什么事件？这个事件被谁消费了？ | ⭐⭐ | 事件驱动 | 发布`user.followed`事件到RabbitMQ；notification-service消费后创建通知（"XXX关注了你"）；audit-service消费后记录审计日志 |
| 3-06 | `GetProfile`接口为什么同时支持按 ID、按用户名、按邮箱查询？是如何判断的？ | ⭐⭐ | 接口设计 | 通过query参数区分（`?id=`/`?username=`/`?email=`），repository层判断哪个参数非空选择对应查询条件；一个接口覆盖多种查询场景 |

---

## 第 4 章：内容服务与缓存策略

| # | 题目 | 难度 | 考察要点 | 参考答案要点 |
|---|------|------|---------|------------|
| 4-01 | Article 模型有哪些核心字段？Tags 字段是如何存储的？ | ⭐ | 数据模型 | Article含title/content/category_id/status/view_count/like_count/comment_count/tags；tags字段使用JSON类型存储字符串数组 |
| 4-02 | 什么是 Cache-Aside 模式？项目在内容服务中是如何应用的？ | ⭐⭐ | 缓存模式 | Cache-Aside：读时先查缓存→未命中查DB→写缓存→返回；写时先写DB→删缓存；content-service的GetArticle严格遵循此模式 |
| 4-03 | 为什么要设置缓存空值（Null Value Cache）？不设会有什么风险？ | ⭐⭐ | 防穿透 | 防止缓存穿透：大量请求查询不存在的文章ID，缓存未命中→全部打到MySQL→数据库过载；空值缓存5分钟过期，拦截无效请求 |
| 4-04 | SingleFlight 是什么？在项目中解决了什么问题？ | ⭐⭐⭐ | 防击穿 | 缓存击穿预防：热门文章过期瞬间，大量并发请求同时查MySQL；SingleFlight确保同一key只有一个goroutine执行DB查询，其余等待结果共享，防止数据库被打爆 |
| 4-05 | 文章浏览计数为什么从 MySQL 行锁改为 Redis INCR？带来了多大的性能提升？ | ⭐⭐⭐ | 性能优化 | MySQL行锁版：每次阅读UPDATE view_count+1 → 行锁竞争 → QPS仅807；Redis INCR版：原子操作无锁 → QPS达21887+ → 定期5分钟批量同步回MySQL → 提升27倍+ |
| 4-06 | `SyncViewCounts` 定时同步机制是如何工作的？如果服务在同步前崩溃会丢失数据吗？ | ⭐⭐⭐ | 数据可靠性 | 每5分钟执行：SCAN Redis中所有`view_count:*`键→`GET`当前值→`UPDATE articles SET view_count=view_count+?`→`DEL` Redis键；极端情况：崩溃时丢失最多5分钟增量，对于浏览计数是可接受的 |
| 4-07 | 文章的发布流程是怎样的？为什么是"先创建草稿再发布"而不是直接发布？ | ⭐⭐ | 业务设计 | CreateArticle→状态=draft→EditArticle(可多次修改)→PublishArticle→状态=published+published_at写入；草稿机制允许用户逐步完善，意外关闭不丢失内容 |
| 4-08 | 为什么只有草稿状态的文章可以编辑？已发布的文章不能直接修改吗？ | ⭐ | 业务设计 | 已发布文章已有读者看到和评论，直接修改内容可能破坏上下文一致性；编辑限制在草稿阶段保证了内容的稳定性 |
| 4-09 | 文章删除是物理删除还是软删除？如何实现的？ | ⭐ | 数据安全 | GORM软删除(`DeletedAt`字段)；GORM查询自动过滤`deleted_at IS NOT NULL`的记录；数据可恢复，防止误删 |

---

## 第 5 章：互动服务

| # | 题目 | 难度 | 考察要点 | 参考答案要点 |
|---|------|------|---------|------------|
| 5-01 | 评论系统支持嵌套回复吗？数据库是如何建模的？ | ⭐⭐ | 评论建模 | 支持两层结构：顶级评论(parent_id=NULL)和子回复(parent_id=父评论ID)；Comment模型含`parent_id`字段自引用；查询时先取顶级评论→再按parent_id取子回复→前端渲染为树形结构 |
| 5-02 | Like 模型为什么被称为"多态点赞"？是如何实现的？ | ⭐⭐ | 多态设计 | 一个Like表同时服务于点赞文章和点赞评论；通过`target_type`(article/comment)和`target_id`字段区分目标类型；避免为每种可点赞对象建独立表 |
| 5-03 | 点赞数的缓存策略和文章浏览计数有什么相同和不同？ | ⭐⭐ | 缓存对比 | 相同：都用Redis缓存计数减少MySQL压力；不同：点赞有"取消点赞"操作(Redis DECR)，浏览次数只增不减；点赞缓存TTL更短(1分钟 vs 2小时) |
| 5-04 | 评论创建后触发了什么事件？通知是如何生成和发送的？ | ⭐⭐ | 事件通知 | 发布`comment.created`事件→notification-service消费→给文章作者创建通知"XXX评论了你的文章"；通知类型区分来源，方便前端展示 |
| 5-05 | 收藏功能为什么只用 MySQL 而没用 Redis 缓存？这个设计合理吗？ | ⭐⭐⭐ | 缓存决策 | 收藏是低频写操作（远低于浏览/点赞），缓存收益小；每个用户的收藏列表差异大，缓存命中率低；直接查MySQL简单可靠 |
| 5-06 | CommentResponse 结构体是如何设计的？为什么返回的不是 GORM 模型？ | ⭐ | 响应设计 | CommentResponse含用户信息(username/avatar)、子评论列表(children)、时间戳；屏蔽数据库模型细节，前端拿到完整可渲染数据；符合API响应封装原则 |

---

## 第 6 章：通知与搜索服务

| # | 题目 | 难度 | 考察要点 | 参考答案要点 |
|---|------|------|---------|------------|
| 6-01 | Notification Service 为什么同时是 RabbitMQ 消费者和 HTTP 服务器？两者如何共存？ | ⭐⭐ | 双模式设计 | 消费者：`service.StartListening()`启动goroutine监听4种事件；HTTP服务器：Gin路由提供通知查询/标记已读API；两者在同一进程中用不同goroutine运行，共享同一个数据库连接 |
| 6-02 | 通知服务消费了哪 4 种事件？每种事件生成的通知对象是什么？ | ⭐⭐ | 事件消费 | `article.published`(当前仅日志/TODO) → `comment.created`(给文章作者发"评论了你的文章") → `user.followed`(给被关注者发"关注了你") → `article.liked`(给文章作者发"点赞了你的文章") |
| 6-03 | 通知列表查询为什么需要添加复合索引？索引字段是什么？ | ⭐⭐⭐ | 索引优化 | DEBUG.md记录：通知查询按`user_id`过滤+按`created_at`排序；缺少`(user_id, created_at)`复合索引导致全表扫描；添加后查询走覆盖索引，大幅提升分页查询性能 |
| 6-04 | Elasticsearch 在项目中用来做什么？为什么选择 IK 分词器？ | ⭐ | 搜索设计 | Elasticsearch提供全文搜索能力；IK分词器专门处理中文分词（"中华人民共和国"→"中华人民共和国/中华/人民/共和国"）；支持multi_match多字段搜索(title权重×3) |
| 6-05 | Search Service 的索引同步是如何实现的？如何保证 ES 中的数据与 MySQL 一致？ | ⭐⭐ | 数据同步 | 事件驱动同步：监听`article.published`→索引到ES；监听`article.deleted`→从ES删除；最终一致性：ES可能短暂滞后于MySQL（毫秒级），对搜索场景可接受 |
| 6-06 | 搜索结果返回了什么？如何处理搜索结果高亮？ | ⭐⭐ | 结果高亮 | ES返回`_source`(原始文档)+`highlight`字段(匹配片段加`<em>`标签)；前端渲染时解析高亮标记展示搜索词上下文；结果按`_score`评分+创建时间排序 |
| 6-07 | Audit Service 订阅了所有事件（`#`通配符），这个设计的目的是什么？ | ⭐ | 审计设计 | 全局审计：所有业务事件记录到`audit_logs`表（谁/什么操作/什么资源/详细信息/IP）；支持按用户/操作类型/资源筛选查询；用于排查问题和安全审计 |

---

## 第 7 章：事件驱动与消息队列

| # | 题目 | 难度 | 考察要点 | 参考答案要点 |
|---|------|------|---------|------------|
| 7-01 | 项目为什么引入 RabbitMQ？事件驱动相比直接 HTTP 调用有什么优势？ | ⭐ | 事件驱动 | 解耦服务：发布者不需要知道消费者是谁；异步：不阻塞主流程；可靠性：消息持久化+手动确认保证不丢失；扩展性：新增消费者无需修改发布者代码 |
| 7-02 | RabitMQ 使用的是哪种 Exchange 类型？Routing Key 的设计规则是什么？ | ⭐⭐ | 消息路由 | Topic Exchange(`blog_events`)；Routing Key格式：`{resource}.{action}`如`article.published`；消费者可以用精确匹配或通配符(`#`匹配所有)绑定队列 |
| 7-03 | 为什么要使用手动确认（Manual Ack）而不是自动确认？Nack 后的重试策略是什么？ | ⭐⭐ | 可靠性设计 | 自动确认：消息投递即删除，处理失败无法重试；手动确认：处理成功才Ack，失败Nack并requeue→重新投递；保证消息至少被成功处理一次(At-Least-Once) |
| 7-04 | 消息发布时为什么设置 `DeliveryMode: persistent`？不设会怎样？ | ⭐ | 消息持久化 | 持久化消息写入磁盘，RabbitMQ重启后不丢失；非持久消息存在内存，重启或崩溃即丢失 |
| 7-05 | 每个微服务各自声明自己的队列和绑定，为什么不统一在一个地方声明？ | ⭐⭐ | 自治设计 | 每个服务对自己的消费契约负责；服务启动时自动声明队列→即使RabbitMQ被重建也能自动恢复拓扑；新增消费者不影响现有服务 |
| 7-06 | 事件发布是同步还是异步（fire-and-forget）？如果发布失败呢？ | ⭐⭐ | 容错设计 | goroutine异步发布，不阻塞主请求响应；当前没有重试机制（可接受：发布失败意味着RabbitMQ不可用，整个系统已降级）；极端重要事件可考虑outbox模式 |
| 7-07 | `getStringFromMap` 在 Audit Service 中的作用是什么？为什么要这样设计？ | ⭐⭐⭐ | 灵活解析 | 不同event_data结构各异（comment事件含content，follow事件不含）；`getStringFromMap`灵活地从不同结构的JSON中提取user_id；避免为每种事件定义专用解析器 |

---

## 第 8 章：前端架构与状态管理

| # | 题目 | 难度 | 考察要点 | 参考答案要点 |
|---|------|------|---------|------------|
| 8-01 | 前端项目用了什么技术栈？各组件分别负责什么？ | ⭐ | 技术栈 | Vue 3(Composition API+`<script setup>`) + TypeScript + Vite 8(构建) + Pinia 3(状态管理) + Vue Router 5(路由) + Axios(HTTP)；关注点分离，各司其职 |
| 8-02 | Axios 拦截器做了什么处理？请求拦截器和响应拦截器分别负责什么？ | ⭐⭐ | 请求封装 | 请求拦截：从localStorage读取JWT token→添加`Authorization: Bearer xxx`头；响应拦截：自动解包`.data`字段→捕获401错误→触发logout→简化业务代码 |
| 8-03 | Pinia store 中用户状态是如何管理的？登录和登出的完整流程是怎样的？ | ⭐⭐ | 状态管理 | `useUserStore`：`token`/`userInfo`/`isLoggedIn`响应式状态；login：调用API→解析JWT payload获取userId→调用getProfile获取用户信息→保存token到localStorage；logout：清空state+移除localStorage |
| 8-04 | Vue Router 的导航守卫做了什么？为什么需要在路由层面做认证检查？ | ⭐ | 路由守卫 | `beforeEach`检查`meta.requiresAuth`→若需要认证且未登录→重定向`/login`；路由层拦截避免未认证用户看到组件内部才报401，体验更好 |
| 8-05 | 前端项目如何与后端开发环境对接？`vite.config.ts` 中做了什么配置？ | ⭐ | 开发配置 | Vite的`server.proxy`配置：`/api` → `http://localhost:8000`；开发环境下前端请求被代理到API Gateway，避免跨域问题 |
| 8-06 | CommentList 组件是如何实现评论树形结构的？递归渲染怎么做？ | ⭐⭐ | 树形渲染 | 数据结构：顶级评论数组+每个评论含`children`子评论数组；模板递归：顶级渲染`v-for`→每项内部再渲染自身的`children`列表→递归用自身组件名；实现树形嵌套展示 |
| 8-07 | 前端项目的路由结构是怎样的？哪些页面是懒加载的？ | ⭐ | 路由设计 | 7条路由：Home(首页) / Login / Register / Article(详情) / Editor(编辑) / Notifications / UserProfile；全部使用`() => import()`动态导入实现路由级懒加载，减少首屏JS体积 |

---

## 第 9 章：测试体系、部署与工程质量

| # | 题目 | 难度 | 考察要点 | 参考答案要点 |
|---|------|------|---------|------------|
| 9-01 | 项目中存在哪些类型的测试？目前测试覆盖情况如何？ | ⭐ | 测试现状 | `content-service/repository/article_test.go`包含单元测试；测试覆盖SingleFlight防击穿等核心逻辑；整体测试体系仍在建设中 |
| 9-02 | SingleFlight 的测试是如何设计的？如何验证"防止缓存击穿"的有效性？ | ⭐⭐ | 测试验证 | 并发测试：同时启动大量goroutine查询同一key→验证DB查询次数=1→其余goroutine共享结果；对比测试：不使用SingleFlight时所有goroutine都查DB |
| 9-03 | Docker Compose 中服务之间的依赖关系是如何处理的？健康检查做了什么？ | ⭐⭐ | 容器编排 | `depends_on`声明依赖关系(如user-service依赖mysql/redis/rabbitmq)；`condition: service_healthy`等待依赖服务健康检查通过才启动；MySQL健康检查用`mysqladmin ping`，ES用`curl` |
| 9-04 | 每个微服务都有自己的 Dockerfile，它们用的构建模式有什么共同特点？ | ⭐⭐ | 构建优化 | 全部使用多阶段构建：第一阶段`go build`编译二进制；第二阶段`alpine`精简镜像(只含二进制)；最终镜像体积小、无编译工具链残留、安全攻击面小 |
| 9-05 | DEBUG.md 记录了 14 个问题和修复，选一个你认为最有代表性的说说你的理解。 | ⭐⭐⭐ | 工程反思 | (开放题)如"浏览量MySQL行锁→Redis INCR优化"——代表性问题：表面看是功能正确但性能不达标→通过分析瓶颈(行锁竞争)→引入新组件(Redis原子操作)→设计补偿机制(定时同步)→最终QPS提升27倍→体现了"先正确后优化"的工程方法论 |
| 9-06 | 如果要在项目中新增一个"点赞通知不重复推送"的功能，你会怎么设计？涉及哪些服务和数据表？ | ⭐⭐⭐ | 综合设计 | notification-service新增去重逻辑：同user_id+source_id+type在短时间内(N分钟)不重复创建；可用Redis SETNX做幂等锁；或在notifications表加唯一索引(user_id,source_id,type,created_at日期截断) |
| 9-07 | 项目目前的 Go 模块组织方式是什么？shared 模块如何被各服务引用？ | ⭐⭐ | 模块管理 | 每个服务独立go.mod模块+shared也是独立模块；各服务通过`replace`指令引用本地shared路径（`replace blog-community/shared => ../../shared`）；Docker构建时需复制整个backend目录 |
| 9-08 | 如果项目要上线生产环境，你认为还有哪些最关键的TODO？ | ⭐⭐⭐ | 生产就绪 | 敏感配置环境变量化(JWT密钥/DB密码)、CI/CD流水线、日志集中收集(ELK)、监控告警(Prometheus+Grafana)、TLS/HTTPS、数据库备份策略、集成测试补充、压力测试基线 |

---

## 题库统计

| 章节 | 题数 | ⭐基础 | ⭐⭐进阶 | ⭐⭐⭐深挖 |
|------|------|--------|----------|----------|
| 第1章：项目全景与架构设计 | 8 | 3 | 4 | 1 |
| 第2章：API网关与认证体系 | 8 | 2 | 5 | 1 |
| 第3章：用户服务 | 6 | 2 | 4 | 0 |
| 第4章：内容服务与缓存策略 | 9 | 3 | 3 | 3 |
| 第5章：互动服务 | 6 | 1 | 4 | 1 |
| 第6章：通知与搜索服务 | 7 | 2 | 4 | 1 |
| 第7章：事件驱动与消息队列 | 7 | 2 | 4 | 1 |
| 第8章：前端架构与状态管理 | 7 | 3 | 4 | 0 |
| 第9章：测试体系与工程质量 | 8 | 1 | 3 | 4 |
| **合计** | **66** | **19** | **35** | **12** |
