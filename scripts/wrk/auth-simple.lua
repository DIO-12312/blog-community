-- 极简认证读测试 (不含 common.lua)
math.randomseed(os.time())
counter = 0
local ids = {"08af1cd0-d647-45df-9cbf-6758e44ae7cc","2e443349-f1f7-42ec-937a-b9510c53a03e","9271c875-1673-4e31-b5f9-6db50df3ba88","c814c9d0-a8d7-48b5-b9e4-55390f1e20ca","d3b1ca83-0ed8-4bec-99fd-8467620033a2","ef543ed8-d82f-4ca7-9972-610899a7907b"}

local routes = {
    "/api/notifications",
    "/api/notifications/unread-count",
    "/api/collections",
    "/api/audit-logs",
}

request = function()
    counter = counter + 1
    local path = routes[math.random(#routes)]
    return wrk.format("GET", path)
end

responses = { status_2xx = 0, status_3xx = 0, status_4xx = 0, status_5xx = 0, errors = 0 }
function response(status, headers, body)
    local s = tonumber(status) or 0
    if s >= 200 and s < 300 then responses.status_2xx = responses.status_2xx + 1
    elseif s >= 300 and s < 400 then responses.status_3xx = responses.status_3xx + 1
    elseif s >= 400 and s < 500 then responses.status_4xx = responses.status_4xx + 1
    elseif s >= 500 then responses.status_5xx = responses.status_5xx + 1
    else responses.errors = responses.errors + 1 end
end

function done(summary, latency, requests)
    io.write("------------------------------\n")
    io.write(string.format("Total:    %d\n", summary.requests))
    io.write(string.format("2xx:      %d\n", responses.status_2xx))
    io.write(string.format("3xx:      %d\n", responses.status_3xx))
    io.write(string.format("4xx:      %d\n", responses.status_4xx))
    io.write(string.format("5xx:      %d\n", responses.status_5xx))
    io.write(string.format("Errors:   %d\n", responses.errors))
    io.write(string.format("Req/sec:  %.2f\n", summary.requests / (summary.duration / 1000000)))
    io.write("------------------------------\n")
end
