-- pub-read.lua — 公开读接口 QPS 极限测试
-- 测试: 文章列表 / 文章详情 / 搜索 / 用户查询 (均无需认证)
-- 用法: WRK_TOKEN=xxx wrk -t4 -c100 -d30s -s pub-read.lua http://localhost:8080

require("common")

-- 预设 ID 池 (从数据库查到的真实 ID，启动后先跑 populate_ids.sh 填充)
local ids = {"08af1cd0-d647-45df-9cbf-6758e44ae7cc","2e443349-f1f7-42ec-937a-b9510c53a03e","9271c875-1673-4e31-b5f9-6db50df3ba88","c814c9d0-a8d7-48b5-b9e4-55390f1e20ca","d3b1ca83-0ed8-4bec-99fd-8467620033a2","ef543ed8-d82f-4ca7-9972-610899a7907b"}
-- 搜索关键词池
local keywords = {"golang", "vue", "docker", "mysql", "redis", "elasticsearch",
                   "微服务", "博客", "教程", "实战"}

-- 路由表 (权重分配)
local routes = {
    { path = "/api/articles",                  weight = 25 }, -- 文章列表
    { path = "/api/articles/ID",               weight = 25 }, -- 文章详情 (ID 动态替换)
    { path = "/api/search?q=KW&page=1&size=10", weight = 20 }, -- 搜索
    { path = "/api/articles/category/golang",  weight = 15 }, -- 分类
    { path = "/api/users?username=testuser",   weight = 15 }, -- 用户查询
}

-- 构建加权数组
local weighted = {}
for _, r in ipairs(routes) do
    for i = 1, r.weight do
        table.insert(weighted, r)
    end
end

request = function()
    counter = counter + 1
    local route = weighted[math.random(#weighted)]
    local path = route.path

    -- 替换动态参数
    if string.find(path, "ID") then
        path = string.gsub(path, "ID", random_from_table(ids))
    end
    if string.find(path, "KW") then
        path = string.gsub(path, "KW", random_from_table(keywords))
    end

    return wrk.format("GET", path)
end
