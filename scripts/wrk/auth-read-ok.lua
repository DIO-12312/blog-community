-- auth-read-ok.lua — 认证读接口 (wrk.headers 在 init 中设置)
require("common")

local ids = {"08af1cd0-d647-45df-9cbf-6758e44ae7cc","2e443349-f1f7-42ec-937a-b9510c53a03e","9271c875-1673-4e31-b5f9-6db50df3ba88","c814c9d0-a8d7-48b5-b9e4-55390f1e20ca","d3b1ca83-0ed8-4bec-99fd-8467620033a2","ef543ed8-d82f-4ca7-9972-610899a7907b"}

local routes = {
    { path = "/api/notifications",              weight = 25 },
    { path = "/api/notifications/unread-count", weight = 15 },
    { path = "/api/collections",                weight = 25 },
    { path = "/api/collections/status?article_id=ID", weight = 10 },
    { path = "/api/users/ID/followers?page=1&size=10", weight = 10 },
    { path = "/api/users/ID/followings?page=1&size=10", weight = 10 },
    { path = "/api/audit-logs",                 weight = 5 },
}

local weighted = {}
for _, r in ipairs(routes) do
    for i = 1, r.weight do
        table.insert(weighted, r)
    end
end

function init(args)
    thread_id = args[1] or 0
    token_cache = os.getenv("WRK_TOKEN") or ""
    if token_cache ~= "" then
        wrk.headers["Authorization"] = "Bearer " .. token_cache
    end
end

request = function()
    counter = counter + 1
    local route = weighted[math.random(#weighted)]
    local path = route.path
    if string.find(path, "ID") then
        path = string.gsub(path, "ID", random_from_table(ids))
    end
    return wrk.format("GET", path)
end
