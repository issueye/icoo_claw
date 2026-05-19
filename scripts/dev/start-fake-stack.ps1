param(
  [switch]$StartPreview
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$runtimeRoot = Join-Path $repoRoot ".local\fake-stack"
$binDir = Join-Path $runtimeRoot "bin"
$configDir = Join-Path $runtimeRoot "config"
$dataDir = Join-Path $runtimeRoot "data"
$logDir = Join-Path $runtimeRoot "logs"
$runDir = Join-Path $runtimeRoot "run"

$sessionPort = 8082
$gatewayPort = 8080
$clawPortStart = 8101
$clawPortEnd = 8108
$token = "dev-internal-token"
$agentId = "agent_desktop_default"

foreach ($dir in @($binDir, $configDir, $dataDir, $logDir, $runDir)) {
  New-Item -ItemType Directory -Force -Path $dir | Out-Null
}

function Write-Utf8NoBom {
  param(
    [string]$Path,
    [string]$Content
  )

  $encoding = New-Object System.Text.UTF8Encoding($false)
  [System.IO.File]::WriteAllText($Path, $Content, $encoding)
}

function Stop-ByPidFile {
  param([string]$Name)

  $pidPath = Join-Path $runDir "$Name.pid"
  if (-not (Test-Path $pidPath)) {
    return
  }

  $processIdValue = (Get-Content $pidPath -Raw).Trim()
  if ($processIdValue) {
    $process = Get-Process -Id ([int]$processIdValue) -ErrorAction SilentlyContinue
    if ($process) {
      Stop-Process -Id $process.Id -Force
      Start-Sleep -Milliseconds 300
    }
  }
  Remove-Item $pidPath -Force -ErrorAction SilentlyContinue
}

function Stop-FakeClawProcesses {
  $expectedPath = (Join-Path $binDir "claw.exe")
  Get-Process claw -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -eq $expectedPath } |
    ForEach-Object {
      Stop-Process -Id $_.Id -Force
      Start-Sleep -Milliseconds 300
    }
}

Stop-ByPidFile "gateway"
Stop-ByPidFile "session_store"
Stop-ByPidFile "desktop-preview"
Stop-FakeClawProcesses

Write-Host "Building local binaries..."
& go build -o (Join-Path $binDir "session_store.exe") "./server/session_store/cmd/session_store"
& go build -o (Join-Path $binDir "claw.exe") "./server/claw/cmd/claw"
& go build -o (Join-Path $binDir "gateway.exe") "./server/gateway/cmd/gateway"

$sessionConfig = @"
http_addr = "127.0.0.1:$sessionPort"
db_path = "$(Join-Path $dataDir "session_store.sqlite" | ForEach-Object { $_ -replace '\\','/' })"
"@
Write-Utf8NoBom -Path (Join-Path $configDir "session_store.toml") -Content $sessionConfig

$gatewayConfig = @"
http_addr = "127.0.0.1:$gatewayPort"
db_path = "$(Join-Path $dataDir "gateway.sqlite" | ForEach-Object { $_ -replace '\\','/' })"
session_store_url = "http://127.0.0.1:$sessionPort"
internal_token = "$token"
claw_binary_path = "$(Join-Path $binDir "claw.exe" | ForEach-Object { $_ -replace '\\','/' })"
claw_work_dir = "$( $repoRoot -replace '\\','/' )"
claw_config_dir = "$(Join-Path $dataDir "claw-configs" | ForEach-Object { $_ -replace '\\','/' })"
claw_runner_mode = "fake"
claw_port_start = $clawPortStart
claw_port_end = $clawPortEnd
max_agent_instances = 2
health_interval_seconds = 1
shutdown_timeout_seconds = 2
"@
Write-Utf8NoBom -Path (Join-Path $configDir "gateway.toml") -Content $gatewayConfig

Write-Host "Starting session_store..."
$sessionProc = Start-Process `
  -FilePath (Join-Path $binDir "session_store.exe") `
  -ArgumentList @("--config", (Join-Path $configDir "session_store.toml")) `
  -WorkingDirectory $repoRoot `
  -WindowStyle Hidden `
  -RedirectStandardOutput (Join-Path $logDir "session_store.out.log") `
  -RedirectStandardError (Join-Path $logDir "session_store.err.log") `
  -PassThru
Set-Content -Path (Join-Path $runDir "session_store.pid") -Value $sessionProc.Id -Encoding UTF8

Write-Host "Starting gateway..."
$gatewayProc = Start-Process `
  -FilePath (Join-Path $binDir "gateway.exe") `
  -ArgumentList @("--config", (Join-Path $configDir "gateway.toml")) `
  -WorkingDirectory $repoRoot `
  -WindowStyle Hidden `
  -RedirectStandardOutput (Join-Path $logDir "gateway.out.log") `
  -RedirectStandardError (Join-Path $logDir "gateway.err.log") `
  -PassThru
Set-Content -Path (Join-Path $runDir "gateway.pid") -Value $gatewayProc.Id -Encoding UTF8

function Wait-Health {
  param(
    [string]$Name,
    [string]$Url
  )

  $deadline = (Get-Date).AddSeconds(20)
  while ((Get-Date) -lt $deadline) {
    try {
      $resp = Invoke-RestMethod -Uri $Url -Method Get -TimeoutSec 2
      if ($resp.status -eq "ok") {
        return
      }
    } catch {
      Start-Sleep -Milliseconds 300
    }
  }
  throw "$Name did not become healthy: $Url"
}

Wait-Health "session_store" "http://127.0.0.1:$sessionPort/health"
Wait-Health "gateway" "http://127.0.0.1:$gatewayPort/health"

Write-Host "Ensuring default agent..."
$agentBody = @{
  id = $agentId
  name = "Desktop Default Agent"
  model_provider = "openai"
  model_name = "fake"
  max_iterations = 1
  tool_whitelist = @()
  enabled = $true
} | ConvertTo-Json -Depth 5

try {
  Invoke-RestMethod `
    -Uri "http://127.0.0.1:$gatewayPort/v1/agents" `
    -Method Post `
    -ContentType "application/json" `
    -Body $agentBody `
    -TimeoutSec 5 | Out-Null
} catch {
  if ($_.Exception.Response.StatusCode.value__ -ne 502 -and $_.Exception.Response.StatusCode.value__ -ne 409) {
    try {
      $existing = Invoke-RestMethod -Uri "http://127.0.0.1:$gatewayPort/v1/agents/$agentId" -Method Get -TimeoutSec 5
      if (-not $existing.id) {
        throw
      }
    } catch {
      throw
    }
  }
}

Write-Host ""
Write-Host "Fake stack is ready."
Write-Host "Gateway:       http://127.0.0.1:$gatewayPort"
Write-Host "Default Agent: $agentId"
Write-Host "Desktop exe:   $(Join-Path $repoRoot "desktop\bin\desktop.exe")"
Write-Host ""
Write-Host "Desktop settings:"
Write-Host "  baseUrl        = http://127.0.0.1:$gatewayPort"
Write-Host "  defaultAgentId = $agentId"

if ($StartPreview) {
  $frontendDir = Join-Path $repoRoot "desktop\frontend"
  $previewPort = 4173
  $previewPidPath = Join-Path $runDir "desktop-preview.pid"

  if (Test-Path $previewPidPath) {
    $existingId = (Get-Content $previewPidPath -Raw).Trim()
    if ($existingId) {
      $existing = Get-Process -Id ([int]$existingId) -ErrorAction SilentlyContinue
      if ($existing) {
        Stop-Process -Id $existing.Id -Force
        Start-Sleep -Milliseconds 300
      }
    }
    Remove-Item $previewPidPath -Force -ErrorAction SilentlyContinue
  }

  Write-Host ""
  Write-Host "Starting desktop frontend preview..."
  $previewProc = Start-Process `
    -FilePath powershell `
    -ArgumentList @("-NoProfile", "-Command", "npm run dev -- --host 127.0.0.1 --port $previewPort") `
    -WorkingDirectory $frontendDir `
    -WindowStyle Hidden `
    -RedirectStandardOutput (Join-Path $logDir "desktop-preview.out.log") `
    -RedirectStandardError (Join-Path $logDir "desktop-preview.err.log") `
    -PassThru
  Set-Content -Path $previewPidPath -Value $previewProc.Id -Encoding UTF8

  $previewDeadline = (Get-Date).AddSeconds(20)
  while ((Get-Date) -lt $previewDeadline) {
    try {
      $response = Invoke-WebRequest -Uri "http://127.0.0.1:$previewPort" -UseBasicParsing -TimeoutSec 2
      if ($response.StatusCode -eq 200) {
        Write-Host "Preview:       http://127.0.0.1:$previewPort"
        break
      }
    } catch {
      Start-Sleep -Milliseconds 500
    }
  }
}
