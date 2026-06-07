# docker-run.ps1 — Windows Docker 版 wrk 多方位压测 (无需本地安装 wrk)
#
# 用法:
#   powershell -File docker-run.ps1
#   或指定参数:
#   powershell -File docker-run.ps1 -Host "http://localhost:8080" -User "testuser" -Pass "test123"

param(
    [string]$Host  = "http://host.docker.internal:8080",
    [string]$User  = "testuser",
    [string]$Pass  = "test123",
    [int]$Threads  = 4,
    [int]$Conns    = 100,
    [string]$Duration = "30s"
)

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ResultDir = "$ScriptDir\results\$(Get-Date -Format 'yyyyMMdd_HHmmss')"
New-Item -ItemType Directory -Force -Path $ResultDir | Out-Null

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  WRK QPS 极限测试 (Docker 模式)" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "目标: $Host"
Write-Host "账号: $User"
Write-Host ""

# 获取 Token
Write-Host "[1/2] 获取 JWT Token ..." -ForegroundColor Yellow
$body = @{username=$User;password=$Pass} | ConvertTo-Json
try {
    $resp = Invoke-RestMethod -Uri "$Host/api/users/login" -Method POST -Body $body -ContentType "application/json"
    $Token = $resp.data.token
    Write-Host "[OK] Token: $($Token.Substring(0, [Math]::Min(30, $Token.Length)))..." -ForegroundColor Green
} catch {
    Write-Host "[ERROR] 登录失败: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "[2/2] 预热 ..." -ForegroundColor Yellow
docker run --rm --add-host=host.docker.internal:host-gateway `
    williamyeh/wrk -t2 -c10 -d10s "$Host/api/articles" 2>&1 | Select-Object -Last 5
Write-Host ""

# wrk Docker 封装函数
function Run-Wrk {
    param([string]$Name, [string]$Script, [string]$ExtraArgs = "")
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
    Write-Host "[$Name]" -ForegroundColor Cyan
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan

    $log = "$ResultDir\$Name.log"
    $cmd = "docker run --rm --add-host=host.docker.internal:host-gateway " +
           "-v ${ScriptDir}:/scripts -e WRK_TOKEN=$Token " +
           "williamyeh/wrk -t$Threads -c$Conns -d$Duration $ExtraArgs " +
           "-s /scripts/$Script $Host"

    Invoke-Expression $cmd 2>&1 | Tee-Object -FilePath $log
    Write-Host ""
}

# ─── 测试 ───
Run-Wrk "01-pub-read"   "pub-read.lua"
Run-Wrk "02-pub-read-hc" "pub-read.lua" "-t8 -c500"
Run-Wrk "03-auth-read"  "auth-read.lua"
Run-Wrk "04-auth-write" "auth-write.lua" "-t2 -c50"
Run-Wrk "05-mixed"      "mixed.lua" "-t4 -c200 -d60s"
Run-Wrk "06-mixed-hc"   "mixed.lua" "-t8 -c500 -d60s"

# ─── 阶梯加压读 ───
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host "[07-ladder-read] 读接口阶梯加压" -ForegroundColor Cyan
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
foreach ($c in @(50, 100, 200, 500, 1000)) {
    Write-Host "--- 并发: $c ---" -ForegroundColor Yellow
    docker run --rm --add-host=host.docker.internal:host-gateway `
        -v "${ScriptDir}:/scripts" williamyeh/wrk `
        -t4 -c$c -d15s -s /scripts/pub-read.lua $Host 2>&1 | Select-String "Requests/sec|Latency"
    Start-Sleep 3
}

# ─── 阶梯加压写 ───
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host "[08-ladder-write] 写接口阶梯加压" -ForegroundColor Cyan
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
foreach ($c in @(10, 30, 50, 100)) {
    Write-Host "--- 并发: $c ---" -ForegroundColor Yellow
    docker run --rm --add-host=host.docker.internal:host-gateway `
        -v "${ScriptDir}:/scripts" -e WRK_TOKEN=$Token `
        williamyeh/wrk -t2 -c$c -d15s -s /scripts/auth-write.lua $Host 2>&1 | Select-String "Requests/sec|Latency"
    Start-Sleep 3
}

Write-Host "========================================" -ForegroundColor Green
Write-Host "  完成! 结果: $ResultDir" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
