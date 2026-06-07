-- mixed-v2.lua — 混合负载 80%读+20%写 (不含 wrk.headers，由 -H 传Token)
require("common")

local ids = {"08af1cd0-d647-45df-9cbf-6758e44ae7cc","2e443349-f1f7-42ec-937a-b9510c53a03e","9271c875-1673-4e31-b5f9-6db50df3ba88","c814c9d0-a8d7-48b5-b9e4-55390f1e20ca","d3b1ca83-0ed8-4bec-99fd-8467620033a2","ef543ed8-d82f-4ca7-9972-610899a7907b"}
local keywords = {"golang", "vue", "docker", "mysql", "redis"}
local title_pool = {"压测文章A", "压测文章B", "性能测试C"}
local content_pool = {"压测内容，用于测试系统吞吐量。"}

local scenarios = {
    -- 80% 读
    { m="GET",  path="/api/articles",                        weight=20 },
    { m="GET",  path="/api/articles/ID",                     weight=20 },
    { m="GET",  path="/api/search?q=KW&page=1",              weight=15 },
    { m="GET",  path="/api/articles/category/golang",        weight=10 },
    { m="GET",  path="/api/notifications",                   weight=8  },
    { m="GET",  path="/api/collections",                     weight=7  },
    -- 20% 写
    { m="POST", path="/api/articles/ID/comments", body='{"content":"test"}', weight=8  },
    { m="POST", path="/api/likes",                  body='{"target_id":"ID","target_type":"article"}', weight=5 },
    { m="POST", path="/api/collections",            body='{"article_id":"ID"}', weight=4 },
    { m="POST", path="/api/articles",               body='{"title":"TITLE","content":"BODY","category":"golang"}', weight=3 },
}

local weighted = {}
for _, s in ipairs(scenarios) do
    for i = 1, s.weight do
        table.insert(weighted, s)
    end
end

request = function()
    counter = counter + 1
    local s = weighted[math.random(#weighted)]
    local path = s.path
    if string.find(path, "ID") then path = string.gsub(path, "ID", random_from_table(ids)) end
    if string.find(path, "KW") then path = string.gsub(path, "KW", random_from_table(keywords)) end

    local body = s.body or ""
    if body ~= "" then
        body = string.gsub(body, "ID", random_from_table(ids))
        body = string.gsub(body, "TITLE", random_from_table(title_pool))
        body = string.gsub(body, "BODY", random_from_table(content_pool))
        wrk.headers["Content-Type"] = "application/json"
    end

    return wrk.format(s.m, path, nil, body)
end
