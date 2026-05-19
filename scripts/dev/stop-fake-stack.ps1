Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$runtimeRoot = Join-Path $repoRoot ".local\fake-stack"
$runDir = Join-Path $runtimeRoot "run"

function Stop-ProcessTree {
  param([int]$ProcessId)

  $children = Get-CimInstance Win32_Process -Filter "ParentProcessId = $ProcessId" -ErrorAction SilentlyContinue
  foreach ($child in $children) {
    Stop-ProcessTree -ProcessId ([int]$child.ProcessId)
  }

  $process = Get-Process -Id $ProcessId -ErrorAction SilentlyContinue
  if ($process) {
    Stop-Process -Id $ProcessId -Force
  }
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
      Stop-ProcessTree -ProcessId $process.Id
      Write-Host "Stopped $Name ($processIdValue)"
    }
  }
  Remove-Item $pidPath -Force -ErrorAction SilentlyContinue
}

function Stop-FakeClawProcesses {
  $expectedPath = Join-Path $runtimeRoot "bin\claw.exe"
  Get-Process claw -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -eq $expectedPath } |
    ForEach-Object {
      Stop-Process -Id $_.Id -Force
      Write-Host "Stopped claw ($($_.Id))"
    }
}

function Stop-PreviewProcesses {
  $frontendDir = Join-Path $repoRoot "desktop\frontend"
  $escapedFrontendDir = $frontendDir -replace '\\', '\\'
  Get-CimInstance Win32_Process -ErrorAction SilentlyContinue |
    Where-Object {
      $_.Name -eq "node.exe" -and
      $_.CommandLine -like "*vite*preview*" -and
      $_.CommandLine -like "*$escapedFrontendDir*"
    } |
    ForEach-Object {
      Stop-ProcessTree -ProcessId ([int]$_.ProcessId)
      Write-Host "Stopped desktop preview node ($($_.ProcessId))"
    }
}

Stop-ByPidFile "gateway"
Stop-ByPidFile "session_store"
Stop-ByPidFile "desktop-preview"
Stop-PreviewProcesses
Stop-FakeClawProcesses
