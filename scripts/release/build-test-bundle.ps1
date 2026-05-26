param(
  [string]$OutputPath = "",
  [switch]$NoZip
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$releaseRoot = Join-Path $repoRoot "release"
$bundleStamp = Get-Date -Format "yyyyMMdd-HHmmss"
$bundleName = "icoo-claw-test-windows-x64-$bundleStamp"
if ([string]::IsNullOrWhiteSpace($OutputPath)) {
  $bundleRoot = Join-Path $releaseRoot $bundleName
} elseif ([System.IO.Path]::IsPathRooted($OutputPath)) {
  $bundleRoot = $OutputPath
} else {
  $bundleRoot = Join-Path $repoRoot $OutputPath
}
$releaseRoot = [System.IO.Path]::GetFullPath($releaseRoot)
$bundleRoot = [System.IO.Path]::GetFullPath($bundleRoot)
if (-not $bundleRoot.StartsWith($releaseRoot + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) {
  throw "OutputPath must be inside the release directory: $releaseRoot"
}
$zipPath = "$bundleRoot.zip"

function Write-Utf8NoBom {
  param(
    [string]$Path,
    [string]$Content
  )

  $encoding = New-Object System.Text.UTF8Encoding($false)
  [System.IO.File]::WriteAllText($Path, $Content, $encoding)
}

function New-CleanDirectory {
  param([string]$Path)

  if (Test-Path $Path) {
    Get-ChildItem -LiteralPath $Path -File -Recurse -Force | Remove-Item -Force
    Get-ChildItem -LiteralPath $Path -Directory -Recurse -Force |
      Sort-Object FullName -Descending |
      Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
  } else {
    New-Item -ItemType Directory -Force -Path $Path | Out-Null
  }
}

function Backup-RuntimeData {
  param([string]$BundleRoot)

  $source = Join-Path $BundleRoot "runtime\data"
  if (-not (Test-Path $source)) {
    return ""
  }

  $backupRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("icoo-claw-runtime-" + [System.Guid]::NewGuid().ToString("N"))
  $backupData = Join-Path $backupRoot "data"
  New-Item -ItemType Directory -Force -Path $backupRoot | Out-Null
  Copy-Item -LiteralPath $source -Destination $backupData -Recurse -Force
  return $backupRoot
}

function Restore-RuntimeData {
  param(
    [string]$BackupRoot,
    [string]$RuntimeDataDir
  )

  if ([string]::IsNullOrWhiteSpace($BackupRoot)) {
    return
  }
  $backupData = Join-Path $BackupRoot "data"
  if (-not (Test-Path $backupData)) {
    return
  }
  New-Item -ItemType Directory -Force -Path $RuntimeDataDir | Out-Null
  Get-ChildItem -LiteralPath $backupData -Force | ForEach-Object {
    Copy-Item -LiteralPath $_.FullName -Destination $RuntimeDataDir -Recurse -Force
  }
}

function Remove-RuntimeBackup {
  param([string]$BackupRoot)

  if ([string]::IsNullOrWhiteSpace($BackupRoot)) {
    return
  }
  $backupPath = [System.IO.Path]::GetFullPath($BackupRoot)
  $tempPath = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath())
  if ($backupPath.StartsWith($tempPath, [System.StringComparison]::OrdinalIgnoreCase) -and (Test-Path $backupPath)) {
    Remove-Item -LiteralPath $backupPath -Recurse -Force
  }
}

function Assert-Command {
  param([string]$Name)

  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "Required command not found: $Name"
  }
}

function Invoke-Native {
  param(
    [string]$FilePath,
    [string[]]$ArgumentList,
    [string]$WorkingDirectory = $repoRoot
  )

  Push-Location $WorkingDirectory
  try {
    & $FilePath @ArgumentList
    if ($LASTEXITCODE -ne 0) {
      throw "$FilePath failed with exit code $LASTEXITCODE"
    }
  } finally {
    Pop-Location
  }
}

Assert-Command "go"
Assert-Command "wails3"

Add-Type -AssemblyName System.IO.Compression.FileSystem

New-Item -ItemType Directory -Force -Path $releaseRoot | Out-Null
$runtimeBackupRoot = Backup-RuntimeData -BundleRoot $bundleRoot
New-CleanDirectory $bundleRoot

$binDir = Join-Path $bundleRoot "bin"
$configDir = Join-Path $bundleRoot "config"
$binConfigDir = Join-Path $binDir "config"
$runtimeDir = Join-Path $bundleRoot "runtime"
$runtimeConfigDir = Join-Path $runtimeDir "config"
$runtimeDataDir = Join-Path $runtimeDir "data"
$runtimeLogDir = Join-Path $runtimeDir "logs"
$runtimeRunDir = Join-Path $runtimeDir "run"
$scriptDir = Join-Path $bundleRoot "scripts"

foreach ($dir in @(
  $binDir,
  $binConfigDir,
  $configDir,
  $runtimeConfigDir,
  $runtimeDataDir,
  $runtimeLogDir,
  $runtimeRunDir,
  $scriptDir
)) {
  New-Item -ItemType Directory -Force -Path $dir | Out-Null
}
Restore-RuntimeData -BackupRoot $runtimeBackupRoot -RuntimeDataDir $runtimeDataDir
Remove-RuntimeBackup -BackupRoot $runtimeBackupRoot

Write-Host "Building claw.exe..."
Invoke-Native -FilePath "go" -ArgumentList @("build", "-o", (Join-Path $binDir "claw.exe"), "./server/claw/cmd/claw")

Write-Host "Building gateway.exe..."
Invoke-Native -FilePath "go" -ArgumentList @("build", "-o", (Join-Path $binDir "gateway.exe"), "./server/gateway/cmd/gateway")

Write-Host "Building desktop.exe..."
Invoke-Native -FilePath "wails3" -ArgumentList @("build") -WorkingDirectory (Join-Path $repoRoot "desktop")

$desktopExe = Join-Path $repoRoot "desktop\bin\desktop.exe"
if (-not (Test-Path $desktopExe)) {
  throw "desktop.exe not found after Wails build: $desktopExe"
}
Copy-Item -LiteralPath $desktopExe -Destination (Join-Path $binDir "desktop.exe") -Force

$manualRootGatewayConfig = @"
http_addr = "127.0.0.1:8080"
db_path = "../runtime/data/gateway.sqlite"

session_api_url = "http://127.0.0.1:8080"
internal_token = "dev-internal-token"

claw_binary_path = "../bin/claw.exe"
claw_work_dir = ".."
claw_config_dir = "../runtime/data/claw-configs"
claw_runner_mode = "sdk"
claw_port_start = 8101
claw_port_end = 8199
max_agent_instances = 4
health_interval_seconds = 10
shutdown_timeout_seconds = 10
"@
Write-Utf8NoBom -Path (Join-Path $configDir "gateway.toml") -Content $manualRootGatewayConfig

$manualBinGatewayConfig = @"
http_addr = "127.0.0.1:8080"
db_path = "../../runtime/data/gateway.sqlite"

session_api_url = "http://127.0.0.1:8080"
internal_token = "dev-internal-token"

claw_binary_path = "../claw.exe"
claw_work_dir = "../.."
claw_config_dir = "../../runtime/data/claw-configs"
claw_runner_mode = "sdk"
claw_port_start = 8101
claw_port_end = 8199
max_agent_instances = 4
health_interval_seconds = 10
shutdown_timeout_seconds = 10
"@
Write-Utf8NoBom -Path (Join-Path $binConfigDir "gateway.toml") -Content $manualBinGatewayConfig

$startStackScript = @'
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$packageRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$binDir = Join-Path $packageRoot "bin"
$runtimeRoot = Join-Path $packageRoot "runtime"
$configDir = Join-Path $runtimeRoot "config"
$dataDir = Join-Path $runtimeRoot "data"
$logDir = Join-Path $runtimeRoot "logs"
$runDir = Join-Path $runtimeRoot "run"

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
  Remove-Item -LiteralPath $pidPath -Force -ErrorAction SilentlyContinue
}

function Stop-PackageClawProcesses {
  $expectedPath = Join-Path $binDir "claw.exe"
  Get-Process claw -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -eq $expectedPath } |
    ForEach-Object {
      Stop-Process -Id $_.Id -Force
      Start-Sleep -Milliseconds 300
    }
}

function Stop-GatewayProcessesOnPort {
  Get-NetTCPConnection -LocalPort $gatewayPort -State Listen -ErrorAction SilentlyContinue |
    Select-Object -ExpandProperty OwningProcess -Unique |
    ForEach-Object {
      $process = Get-Process -Id $_ -ErrorAction SilentlyContinue
      if ($process -and $process.ProcessName -eq "gateway") {
        Stop-Process -Id $process.Id -Force
        Start-Sleep -Milliseconds 300
      }
    }
}

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

Stop-ByPidFile "gateway"
Stop-GatewayProcessesOnPort
Stop-PackageClawProcesses

$gatewayConfigPath = Join-Path $configDir "gateway.toml"

$gatewayConfig = @"
http_addr = "127.0.0.1:$gatewayPort"
db_path = "$(Join-Path $dataDir "gateway.sqlite" | ForEach-Object { $_ -replace '\\','/' })"
session_api_url = "http://127.0.0.1:$gatewayPort"
internal_token = "$token"
claw_binary_path = "$(Join-Path $binDir "claw.exe" | ForEach-Object { $_ -replace '\\','/' })"
claw_work_dir = "$( $packageRoot -replace '\\','/' )"
claw_config_dir = "$(Join-Path $dataDir "claw-configs" | ForEach-Object { $_ -replace '\\','/' })"
claw_runner_mode = "sdk"
claw_port_start = $clawPortStart
claw_port_end = $clawPortEnd
max_agent_instances = 2
health_interval_seconds = 1
shutdown_timeout_seconds = 2
"@
Write-Utf8NoBom -Path $gatewayConfigPath -Content $gatewayConfig

Write-Host "Starting gateway..."
$gatewayProc = Start-Process `
  -FilePath (Join-Path $binDir "gateway.exe") `
  -ArgumentList @("--config", $gatewayConfigPath) `
  -WorkingDirectory $packageRoot `
  -WindowStyle Hidden `
  -RedirectStandardOutput (Join-Path $logDir "gateway.out.log") `
  -RedirectStandardError (Join-Path $logDir "gateway.err.log") `
  -PassThru
Set-Content -Path (Join-Path $runDir "gateway.pid") -Value $gatewayProc.Id -Encoding UTF8

Wait-Health "gateway" "http://127.0.0.1:$gatewayPort/health"

Write-Host "Ensuring default agent..."
$agentBody = @{
  id = $agentId
  name = "Desktop Default Agent"
  model_provider = "openai"
  model_name = ""
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
Write-Host "Test stack is ready."
Write-Host "Gateway:       http://127.0.0.1:$gatewayPort"
Write-Host "Default Agent: $agentId"
Write-Host "Desktop exe:   $(Join-Path $binDir "desktop.exe")"
'@
Write-Utf8NoBom -Path (Join-Path $scriptDir "start-stack.ps1") -Content $startStackScript

$stopStackScript = @'
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$packageRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$binDir = Join-Path $packageRoot "bin"
$runDir = Join-Path $packageRoot "runtime\run"

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
      Write-Host "Stopped $Name ($processIdValue)"
    }
  }
  Remove-Item -LiteralPath $pidPath -Force -ErrorAction SilentlyContinue
}

function Stop-PackageClawProcesses {
  $expectedPath = Join-Path $binDir "claw.exe"
  Get-Process claw -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -eq $expectedPath } |
    ForEach-Object {
      Stop-Process -Id $_.Id -Force
      Write-Host "Stopped claw ($($_.Id))"
    }
}

function Stop-DesktopProcess {
  Stop-ByPidFile "desktop"

  $expectedPath = Join-Path $binDir "desktop.exe"
  Get-Process desktop -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -eq $expectedPath } |
    ForEach-Object {
      Stop-Process -Id $_.Id -Force
      Write-Host "Stopped desktop ($($_.Id))"
    }
}

Stop-DesktopProcess
Stop-ByPidFile "gateway"
Stop-PackageClawProcesses
'@
Write-Utf8NoBom -Path (Join-Path $scriptDir "stop-stack.ps1") -Content $stopStackScript

$launchDesktopScript = @'
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$packageRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$desktopExe = Join-Path $packageRoot "bin\desktop.exe"
$runDir = Join-Path $packageRoot "runtime\run"

if (-not (Test-Path $desktopExe)) {
  throw "desktop.exe not found: $desktopExe"
}

$configRoot = Join-Path $env:APPDATA "icoo-claw"
New-Item -ItemType Directory -Force -Path $configRoot | Out-Null
New-Item -ItemType Directory -Force -Path $runDir | Out-Null

$settingsPath = Join-Path $configRoot "settings.toml"
$workspaceRoot = $packageRoot -replace '\\','/'
$content = @"
[gateway]
base_url = "http://127.0.0.1:8080"
default_agent_id = "agent_desktop_default"

[workspace]
root_dir = "$workspaceRoot"

[ui]
show_timestamps = true
"@

$encoding = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllText($settingsPath, $content, $encoding)

Write-Host "Desktop settings written to $settingsPath"
$desktopProc = Start-Process -FilePath $desktopExe -WorkingDirectory $packageRoot -PassThru
Set-Content -Path (Join-Path $runDir "desktop.pid") -Value $desktopProc.Id -Encoding UTF8
'@
Write-Utf8NoBom -Path (Join-Path $scriptDir "launch-desktop.ps1") -Content $launchDesktopScript

$runAllScript = @'
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

& (Join-Path $PSScriptRoot "start-stack.ps1")
& (Join-Path $PSScriptRoot "launch-desktop.ps1")
'@
Write-Utf8NoBom -Path (Join-Path $scriptDir "run-all.ps1") -Content $runAllScript

$startCmd = @'
@echo off
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\run-all.ps1"
if errorlevel 1 pause
'@
Write-Utf8NoBom -Path (Join-Path $bundleRoot "start-test-app.cmd") -Content $startCmd

$stopCmd = @'
@echo off
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\stop-stack.ps1"
if errorlevel 1 pause
'@
Write-Utf8NoBom -Path (Join-Path $bundleRoot "stop-test-app.cmd") -Content $stopCmd

$launchCmd = @'
@echo off
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\launch-desktop.ps1"
if errorlevel 1 pause
'@
Write-Utf8NoBom -Path (Join-Path $bundleRoot "launch-desktop.cmd") -Content $launchCmd

$smokeScript = @'
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$gatewayUrl = "http://127.0.0.1:8080"
$agentId = "agent_desktop_default"
$prompt = "hello from packaged smoke test"
$requestId = "req_smoke_" + [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()

try {
  $null = Invoke-RestMethod -Uri "$gatewayUrl/health" -Method Get -TimeoutSec 3
} catch {
  throw "gateway is not reachable at $gatewayUrl"
}

$conversationBody = @{
  agent_id = $agentId
  title = "Packaged Smoke Test"
} | ConvertTo-Json

$conversation = Invoke-RestMethod `
  -Uri "$gatewayUrl/v1/conversations" `
  -Method Post `
  -ContentType "application/json" `
  -Body $conversationBody `
  -TimeoutSec 5

$ws = [System.Net.WebSockets.ClientWebSocket]::new()
$cts = [System.Threading.CancellationTokenSource]::new()
$cts.CancelAfter(15000)
$uri = [Uri]"ws://127.0.0.1:8080/v1/ws/chat"
$ws.ConnectAsync($uri, $cts.Token).GetAwaiter().GetResult()

$payload = @{
  type = "chat.start"
  conversation_id = $conversation.id
  request_id = $requestId
  prompt = $prompt
} | ConvertTo-Json -Compress

$sendBytes = [System.Text.Encoding]::UTF8.GetBytes($payload)
$segment = [ArraySegment[byte]]::new($sendBytes)
$ws.SendAsync($segment, [System.Net.WebSockets.WebSocketMessageType]::Text, $true, $cts.Token).GetAwaiter().GetResult() | Out-Null

$buffer = New-Object byte[] 4096
$messages = @()

while ($ws.State -eq [System.Net.WebSockets.WebSocketState]::Open) {
  $stream = New-Object System.IO.MemoryStream
  do {
    $recvSegment = [ArraySegment[byte]]::new($buffer)
    $result = $ws.ReceiveAsync($recvSegment, $cts.Token).GetAwaiter().GetResult()
    if ($result.MessageType -eq [System.Net.WebSockets.WebSocketMessageType]::Close) {
      $ws.CloseAsync([System.Net.WebSockets.WebSocketCloseStatus]::NormalClosure, "done", $cts.Token).GetAwaiter().GetResult() | Out-Null
      break
    }
    $stream.Write($buffer, 0, $result.Count)
  } while (-not $result.EndOfMessage)

  if ($stream.Length -eq 0) {
    continue
  }

  $text = [System.Text.Encoding]::UTF8.GetString($stream.ToArray())
  $message = $text | ConvertFrom-Json
  $messages += $message

  if ($message.type -eq "session/completed" -or $message.type -eq "session/error") {
    break
  }
}

$persisted = Invoke-RestMethod -Uri "$gatewayUrl/v1/conversations/$($conversation.id)/messages" -Method Get -TimeoutSec 5

[PSCustomObject]@{
  conversationId = $conversation.id
  websocketEvents = $messages
  persistedMessages = $persisted.messages
} | ConvertTo-Json -Depth 8
'@
Write-Utf8NoBom -Path (Join-Path $scriptDir "smoke-chat-flow.ps1") -Content $smokeScript

$desktopSettingsTemplate = @'
[gateway]
base_url = "http://127.0.0.1:8080"
default_agent_id = "agent_desktop_default"

[workspace]
root_dir = ""

[ui]
show_timestamps = true
'@
Write-Utf8NoBom -Path (Join-Path $configDir "desktop-settings.toml.example") -Content $desktopSettingsTemplate

$readme = @'
# Icoo Claw Test Bundle

Windows test package for the chat-first desktop client and local gateway stack.

## Included

- `bin/desktop.exe`
- `bin/gateway.exe`
- `bin/claw.exe`
- `start-test-app.cmd`
- `stop-test-app.cmd`
- `launch-desktop.cmd`
- `scripts/start-stack.ps1`
- `scripts/launch-desktop.ps1`
- `scripts/run-all.ps1`
- `scripts/stop-stack.ps1`
- `scripts/smoke-chat-flow.ps1`

## Quick Start

1. Double-click `start-test-app.cmd`.
2. Wait for the stack to report ready.
3. The desktop app will open with:
   - Gateway URL: `http://127.0.0.1:8080`
   - Default Agent: `agent_desktop_default`

## Stop

Double-click `stop-test-app.cmd`.

## Smoke Check

After the stack is up, run `.\scripts\smoke-chat-flow.ps1`.

## Manual Gateway Start

From this package root:

```powershell
.\bin\gateway.exe --config .\config\gateway.toml
```

From `bin\`:

```powershell
.\gateway.exe --config .\config\gateway.toml
```

## Notes

- The desktop app writes settings to `%APPDATA%\icoo-claw\settings.toml`.
- Runtime databases, generated configs, logs, and pid files live under `runtime/`.
- Configure a provider API Key, Base URL, and model in the desktop app before starting Agent instances.
- Windows WebView2 runtime is required for the desktop window.
- If the client shows that the gateway cannot be reached, start the package through `start-test-app.cmd` instead of opening `desktop.exe` by itself.
'@
Write-Utf8NoBom -Path (Join-Path $bundleRoot "README.md") -Content $readme

Copy-Item -LiteralPath (Join-Path $repoRoot "config\gateway.toml.example") -Destination (Join-Path $configDir "gateway.toml.example") -Force
Copy-Item -LiteralPath (Join-Path $repoRoot "config\claw.toml.example") -Destination (Join-Path $configDir "claw.toml.example") -Force
if (-not $NoZip) {
  if (Test-Path $zipPath) {
    Remove-Item -LiteralPath $zipPath -Force
  }
  Write-Host "Creating zip archive..."
  [System.IO.Compression.ZipFile]::CreateFromDirectory($bundleRoot, $zipPath)
}

Write-Host ""
Write-Host "Bundle ready:"
Write-Host "  Folder: $bundleRoot"
if (-not $NoZip) {
  Write-Host "  Zip:    $zipPath"
}
