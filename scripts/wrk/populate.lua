-- populate.lua — 辅助: 从 API 获取真实 ID 填充到脚本中
-- 使用 wrk 的 setup 阶段拉取数据
-- 独立用法 (预热 + 采集 ID):
--   wrk -t1 -c1 -d1s -s populate.lua http://localhost:8080

require("common")

-- 这里不发起真正的压测请求，数据采集用 shell 脚本完成
-- 参见 populate_ids.sh
