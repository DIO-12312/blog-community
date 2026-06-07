#!/bin/bash
# run_all.sh — 多方位 QPS 极限测试
#
# 用法:
#   chmod +x run_all.sh
#   ./run_all.sh                              # 默认 localhost:8080
#   WRK_HOST=http://api:8080 ./run_all.sh     # 自定义地址
#   WRK_USER=admin WRK_PASS=123 ./run_all.sh  # 自定义账号
#
# 环境变量:
#   WRK_HOST   — API 网关地址 (默认 http://localhost:8080)
#   WRK_USER   — 登录用户名 (默认 testuser)
#   WRK_PASS   — 登录密码   (默认 test123)
#   WRK_DURATION — 每轮压测时长 (默认 30s)
#   WRK_THREADS  — 线程数 (默认 4)
#   WRK_CONNS    — 并发连接数 (默认 100)

set -euo pipefail

# ============ 配置 ============
HOST="${WRK_HOST:-http://localhost:8000}"
USER="${WRK_USER:-testuser}"
PASS="${WRK_PASS:-test123}"
DURATION="${WRK_DURATION:-30s}"
THREADS="${WRK_THREADS:-4}"
CONNS="${WRK_CONNS:-100}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}  Blog Community — WRK QPS 极限测试${NC}"
echo -e "${CYAN}========================================${NC}"
echo ""
echo -e "目标:     ${GREEN}${HOST}${NC}"
echo -e "账号:     ${USER}"
echo -e "单轮时长: ${DURATION}"
echo -e "线程数:   ${THREADS}"
echo -e "并发连接: ${CONNS}"
echo ""

# ============ 环境检查 ============
if ! command -v wrk &>/dev/null; then
    echo -e "${RED}[ERROR] wrk 未安装${NC}"
    echo ""
    echo "安装方式:"
    echo "  Ubuntu/Debian: sudo apt install wrk"
    echo "  macOS: brew install wrk"
    echo "  Arch: yay -S wrk"
    echo ""
    echo "或用 Docker 运行:"
    echo "  docker run --rm --network host \\"
    echo "    -v ${SCRIPT_DIR}:/scripts \\"
    echo "    williamyeh/wrk -t4 -c100 -d30s -s /scripts/pub-read.lua http://host.docker.internal:8080"
    exit 1
fi

# ============ 获取 JWT Token ============
echo -e "${YELLOW}[1/2] 获取 JWT Token ...${NC}"
TOKEN_RESP=$(curl -s -X POST "${HOST}/api/users/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${USER}\",\"password\":\"${PASS}\"}")

TOKEN=$(echo "$TOKEN_RESP" | grep -o '"token":"[^"]*"' | cut -d'"' -f4 || echo "")

if [ -z "$TOKEN" ]; then
    echo -e "${RED}[ERROR] 登录失败，请检查账号密码或服务是否启动${NC}"
    echo "响应: $TOKEN_RESP"
    exit 1
fi
echo -e "${GREEN}[OK] Token: ${TOKEN:0:30}...${NC}"
echo ""

# ============ 预热 ============
echo -e "${YELLOW}[2/2] 预热 (10s, 10 并发) ...${NC}"
wrk -t2 -c10 -d10s "${HOST}/api/articles" 2>&1 | tail -3
echo ""

# ============ 开始压测 ============
export WRK_TOKEN="$TOKEN"
LOG_DIR="${SCRIPT_DIR}/results/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$LOG_DIR"

run_test() {
    local name="$1"
    local script="$2"
    local extra_args="${3:-}"

    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}[${name}]${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    local log_file="${LOG_DIR}/${name}.log"

    # 实际命令
    wrk -t"${THREADS}" -c"${CONNS}" -d"${DURATION}" \
        ${extra_args} \
        -s "${SCRIPT_DIR}/${script}" \
        "${HOST}" 2>&1 | tee "$log_file"

    echo ""
}

# ─── 测试 1: 纯公开读 ─────────────────────
run_test "01-pub-read" "pub-read.lua"

# ─── 测试 2: 公开读 (高并发) ───────────────
run_test "02-pub-read-hc" "pub-read.lua" "-t8 -c500"

# ─── 测试 3: 认证读 ───────────────────────
run_test "03-auth-read" "auth-read.lua"

# ─── 测试 4: 认证写 (低并发) ───────────────
run_test "04-auth-write" "auth-write.lua" "-t2 -c50"

# ─── 测试 5: 混合负载 ─────────────────────
run_test "05-mixed" "mixed.lua" "-t4 -c200 -d60s"

# ─── 测试 6: 混合负载 (高并发) ─────────────
run_test "06-mixed-hc" "mixed.lua" "-t8 -c500 -d60s"

# ─── 测试 7: 极限读 QPS — 逐步提高并发找到饱和点 ──
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}[07-ladder-read] 读接口阶梯加压${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

for conns in 50 100 200 500 1000; do
    echo -e "${YELLOW}--- 并发: ${conns} ---${NC}"
    wrk -t4 -c"${conns}" -d15s \
        -s "${SCRIPT_DIR}/pub-read.lua" \
        "${HOST}" 2>&1 | grep -E "Requests/sec|Latency" || true
    sleep 3
done
echo ""

# ─── 测试 8: 极限写 QPS — 逐步提高并发 ────
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}[08-ladder-write] 写接口阶梯加压${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

for conns in 10 30 50 100; do
    echo -e "${YELLOW}--- 并发: ${conns} ---${NC}"
    wrk -t2 -c"${conns}" -d15s \
        -s "${SCRIPT_DIR}/auth-write.lua" \
        "${HOST}" 2>&1 | grep -E "Requests/sec|Latency" || true
    sleep 3
done
echo ""

# ============ 汇总报告 ============
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  测试完成! 结果保存在: ${LOG_DIR}${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "快速查看 QPS 汇总:"
echo ""

for f in "$LOG_DIR"/*.log; do
    name=$(basename "$f" .log)
    rps=$(grep "Requests/sec" "$f" | awk '{print $2}' | tail -1 || echo "N/A")
    avg=$(grep "Latency" "$f" | grep -oP '[\d.]+(?=ms)' | head -1 || echo "N/A")
    printf "  %-25s  %10s req/s  |  avg %5s ms\n" "$name" "$rps" "$avg"
done

echo ""
echo "详细日志: ls ${LOG_DIR}/"
