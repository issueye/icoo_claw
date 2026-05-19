Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path

try {
  & (Join-Path $PSScriptRoot "start-fake-stack.ps1") -StartPreview
  Push-Location (Join-Path $repoRoot "desktop\frontend")
  try {
    npm run test:e2e
  } finally {
    Pop-Location
  }
} finally {
  & (Join-Path $PSScriptRoot "stop-fake-stack.ps1")
}
