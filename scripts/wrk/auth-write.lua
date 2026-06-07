-- auth-write.lua — 写接口 QPS 测试
-- 测试: 发文章 / 发评论 / 点赞 / 收藏 / 关注
-- 用法: WRK_TOKEN=<jwt> wrk -t2 -c50 -d30s -s auth-write.lua http://localhost:8080
-- 注意: 写操作压力大，建议降低并发

require("common")

local article_ids = {"08af1cd0-d647-45df-9cbf-6758e44ae7cc","2e443349-f1f7-42ec-937a-b9510c53a03e","9271c875-1673-4e31-b5f9-6db50df3ba88","c814c9d0-a8d7-48b5-b9e4-55390f1e20ca","d3b1ca83-0ed8-4bec-99fd-8467620033a2","ef543ed8-d82f-4ca7-9972-610899a7907b"}

-- 随机标题/内容池
local titles = {
    "Go 语言并发编程实战", "Vue3 组合式 API 详解", "Docker Compose 最佳实践",
    "MySQL 索引优化指南", "Redis 缓存策略总结", "Elasticsearch 全文检索入门",
    "微服务架构设计模式", "博客系统开发日记", "Kubernetes 部署指南",
    "TypeScript 高级类型技巧",
}
local contents = {
    "这是一篇关于编程的实战文章，分享了一些实用的开发经验和技巧。",
    "深入浅出地讲解了核心概念，适合初学者入门学习。",
    "总结了生产环境中的最佳实践，希望对读者有所启发。",
    "通过实际案例演示了该技术的应用场景和解决方案。",
}

math.randomseed(os.time())

request = function()
    counter = counter + 1
    set_auth()
    set_json()

    local dice = counter % 100  -- 0-99 循环

    if dice < 25 then
        -- 25%: 发评论
        local body = string.format(
            '{"content":"测试评论 %d — %s","parent_id":null}',
            counter, os.time()
        )
        local aid = random_from_table(article_ids)
        return wrk.format("POST", "/api/articles/" .. aid .. "/comments", nil, body)

    elseif dice < 45 then
        -- 20%: 点赞
        local body = string.format(
            '{"target_id":"%s","target_type":"article"}',
            random_from_table(article_ids)
        )
        return wrk.format("POST", "/api/likes", nil, body)

    elseif dice < 65 then
        -- 20%: 收藏
        local body = string.format(
            '{"article_id":"%s"}',
            random_from_table(article_ids)
        )
        return wrk.format("POST", "/api/collections", nil, body)

    elseif dice < 80 then
        -- 15%: 发文章
        local body = string.format(
            '{"title":"%s","content":"%s","category":"golang"}',
            random_from_table(titles),
            random_from_table(contents)
        )
        return wrk.format("POST", "/api/articles", nil, body)

    elseif dice < 90 then
        -- 10%: 发布文章 (发布已有文章)
        local aid = random_from_table(article_ids)
        return wrk.format("POST", "/api/articles/" .. aid .. "/publish")

    else
        -- 10%: 关注用户
        local uid = tostring(random_int(1, 10))
        return wrk.format("POST", "/api/users/" .. uid .. "/follow")
    end
end
