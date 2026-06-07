-- debug.lua — 查看实际HTTP状态码
wrk.method = "GET"
wrk.headers["Connection"] = "close"

statuses = {}

function response(status, headers, body)
    local s = tostring(status)
    statuses[s] = (statuses[s] or 0) + 1
end

function done(summary, latency, requests)
    io.write("=== Response Status Breakdown ===\n")
    local keys = {}
    for k,_ in pairs(statuses) do table.insert(keys, k) end
    table.sort(keys)
    for _,k in ipairs(keys) do
        io.write(string.format("  HTTP %s: %d\n", k, statuses[k]))
    end
    io.write("=================================\n")
end
