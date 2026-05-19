# Icoo Claw Desktop

Wails3 desktop chat client for the `icoo_claw` gateway.

## Current scope

- Chat-first shell
- Gateway mode only
- Local TOML settings
- Gateway HTTP discovery
- WebSocket streaming chat

## Commands

From this directory:

```powershell
wails3 generate bindings
wails3 dev -port 9346
wails3 build
```

## Local Chat E2E

From the repo root, install dependencies once:

```powershell
Push-Location .\desktop\frontend
npm ci
npx playwright install
Pop-Location
```

Run the full local smoke chain:

```powershell
.\scripts\dev\run-ui-smoke.ps1
```

The script starts `session_store`, `gateway`, the default fake agent `agent_desktop_default`, a ready fake agent instance, and a production-style Vite preview before running Playwright.

To run E2E against an already-started preview:

```powershell
Push-Location .\desktop\frontend
$env:GATEWAY_BASE_URL = "http://127.0.0.1:8080"
$env:E2E_DEFAULT_AGENT_ID = "agent_desktop_default"
$env:PLAYWRIGHT_BASE_URL = "http://127.0.0.1:4173"
npm run test:e2e
Pop-Location
```
