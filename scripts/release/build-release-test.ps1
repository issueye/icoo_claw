Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$destination = Join-Path $repoRoot "release\test"
$bundleScript = Join-Path $PSScriptRoot "build-test-bundle.ps1"

& $bundleScript -OutputPath $destination -NoZip
