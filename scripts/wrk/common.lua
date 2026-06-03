-- common.lua — 所有压测脚本的公共模块
-- 用法: wrk -s pub-read.lua 时，每个线程 init() 阶段自动登录并缓存 token

local ok, cjson_mod = pcall(require, "cjson")
local cjson = ok and cjson_mod or nil

-- ============ 配置 (通过环境变量或直接修改) ============
local HOST   = os.getenv("WRK_HOST")   or "http://localhost:8080"
local USER   = os.getenv("WRK_USER")   or "testuser"
local PASS   = os.getenv("WRK_PASS")   or "test123"

-- ============ 线程级变量 ============
token_cache = nil         -- token 缓存
article_ids = {}          -- 已知文章 ID 池
counter     = 0           -- 请求计数器
thread_id   = 0           -- 线程编号

-- ============ 初始化 (每个线程执行一次) ============
function init(args)
    thread_id = args[1] or 0
    login()
end

-- ============ 登录获取 JWT ============
function login()
    -- cjson 编码保留用于未来直接登录场景，当前通过 WRK_TOKEN 环境变量传入
    if cjson then
        local _ = cjson.encode({ username = USER, password = PASS })
    end
    -- wrk 本身不支持在 init 阶段发 HTTP，改用 shell 预获取
    -- 这里设置为空，由外部传入 WRK_TOKEN 环境变量覆盖
    token_cache = os.getenv("WRK_TOKEN") or ""
    if token_cache == "" then
        print("[WARN] thread " .. thread_id .. " WRK_TOKEN not set, auth endpoints will fail")
    end
end

-- ============ 请求构造工具 ============
function set_auth()
    if token_cache ~= "" then
        wrk.headers["Authorization"] = "Bearer " .. token_cache
    end
end

function set_json()
    wrk.headers["Content-Type"] = "application/json"
end

function set_form()
    wrk.headers["Content-Type"] = "application/x-www-form-urlencoded"
end

-- ============ 随机工具 ============
math.randomseed(os.time() + (thread_id or 0) * 1000)

function random_from_table(t)
    if #t == 0 then return "" end
    return t[math.random(#t)]
end

function random_int(min, max)
    return math.random(min, max)
end

-- ============ 统一的响应处理 ============
responses = {
    status_2xx = 0,
    status_3xx = 0,
    status_4xx = 0,
    status_5xx = 0,
    errors     = 0,
}

function response(status, headers, body)
    local s = tonumber(status) or 0
    if s >= 200 and s < 300 then
        responses.status_2xx = responses.status_2xx + 1
    elseif s >= 300 and s < 400 then
        responses.status_3xx = responses.status_3xx + 1
    elseif s >= 400 and s < 500 then
        responses.status_4xx = responses.status_4xx + 1
    elseif s >= 500 then
        responses.status_5xx = responses.status_5xx + 1
    else
        responses.errors = responses.errors + 1
    end
end

function done(summary, latency, requests)
    io.write("------------------------------\n")
    io.write(string.format("Total:       %d requests\n", summary.requests))
    io.write(string.format("2xx:         %d\n", responses.status_2xx))
    io.write(string.format("3xx:         %d\n", responses.status_3xx))
    io.write(string.format("4xx:         %d\n", responses.status_4xx))
    io.write(string.format("5xx:         %d\n", responses.status_5xx))
    io.write(string.format("Errors:      %d\n", responses.errors))
    io.write(string.format("Duration:    %.2f µs\n", summary.duration))
    io.write(string.format("Req/sec:     %.2f\n", summary.requests / (summary.duration / 1000000)))
    io.write(string.format("Transfer:    %.2f MB\n", summary.bytes / 1048576))
    io.write("------------------------------\n")
end
