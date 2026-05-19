Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$runDir = Join-Path $repoRoot ".local\fake-stack\run"

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
  Remove-Item $pidPath -Force -ErrorAction SilentlyContinue
}

function Stop-FakeClawProcesses {
  $expectedPath = Join-Path $repoRoot ".local\\fake-stack\\bin\\claw.exe"
  Get-Process claw -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -eq $expectedPath } |
    ForEach-Object {
      Stop-Process -Id $_.Id -Force
      Write-Host "Stopped claw ($($_.Id))"
    }
}

Stop-ByPidFile "gateway"
Stop-ByPidFile "session_store"
Stop-ByPidFile "desktop-preview"
Stop-FakeClawProcesses
