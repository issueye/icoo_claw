Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path

try {
  & (Join-Path $PSScriptRoot "start-fake-stack.ps1") -StartPreview
  Push-Location (Join-Path $repoRoot "desktop\frontend")
  try {
    $env:GATEWAY_BASE_URL = "http://127.0.0.1:8080"
    $env:E2E_DEFAULT_AGENT_ID = "agent_desktop_default"
    $env:PLAYWRIGHT_BASE_URL = "http://127.0.0.1:4173"
    npm run test:e2e
  } finally {
    Remove-Item Env:\GATEWAY_BASE_URL -ErrorAction SilentlyContinue
    Remove-Item Env:\E2E_DEFAULT_AGENT_ID -ErrorAction SilentlyContinue
    Remove-Item Env:\PLAYWRIGHT_BASE_URL -ErrorAction SilentlyContinue
    Pop-Location
  }
} finally {
  & (Join-Path $PSScriptRoot "stop-fake-stack.ps1")
}
