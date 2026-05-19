# Desktop Local Dev Runbook

## Goal

Provide a repeatable local path for validating the desktop chat client against the gateway stack without requiring a real model provider.

## Scope

- Build `session_store`, `claw`, and `gateway`
- Start `session_store` and `gateway`
- Ensure one ready fake `claw` agent instance is available
- Force `claw_runner_mode = "fake"`
- Seed one default agent for the desktop app

## Scripts

- Start: [scripts/dev/start-fake-stack.ps1](/E:/code/issueye/icoo_claw/scripts/dev/start-fake-stack.ps1)
- Smoke chat flow: [scripts/dev/smoke-chat-flow.ps1](/E:/code/issueye/icoo_claw/scripts/dev/smoke-chat-flow.ps1)
- UI smoke: [scripts/dev/run-ui-smoke.ps1](/E:/code/issueye/icoo_claw/scripts/dev/run-ui-smoke.ps1)
- Stop: [scripts/dev/stop-fake-stack.ps1](/E:/code/issueye/icoo_claw/scripts/dev/stop-fake-stack.ps1)

## Runtime Layout

All generated runtime files live under:

```text
.local/fake-stack/
  bin/
  config/
  data/
  logs/
  run/
```

This keeps local binaries, SQLite files, generated TOML, logs, and pid files out of the repo root.

By default, `start-fake-stack.ps1` refreshes only the generated SQLite/config files under `.local/fake-stack/data/` after it stops processes it owns. Use `-PreserveData` when a manual debugging session needs to keep local fake-stack conversations.

## Start Flow

Install dependencies once before running the UI smoke chain:

```powershell
Push-Location .\desktop\frontend
npm ci
npx playwright install
Pop-Location
```

Run:

```powershell
.\scripts\dev\start-fake-stack.ps1
```

The script will:

1. Stop any older `gateway` or `session_store` processes started by the same script
2. Build fresh binaries
3. Generate local TOML config files
4. Start `session_store`
5. Start `gateway`
6. Wait for `/health`
7. Create or reuse default agent `agent_desktop_default`
8. Create or reuse a ready fake agent instance for that agent

## Desktop Settings

Use these values in the desktop app:

```text
Gateway URL: http://127.0.0.1:8080
Default Agent: agent_desktop_default
```

## Preview Flow

Run:

```powershell
.\scripts\dev\start-fake-stack.ps1 -StartPreview
```

This will:

1. Start the fake stack
2. Build the desktop frontend
3. Start the frontend preview at `http://127.0.0.1:4173`
4. Proxy `/health`, `/v1/*`, and `/v1/ws/chat` from Vite to gateway so browser E2E does not require backend CORS
5. Reuse browser fallback settings storage when the app is not running inside Wails

## Smoke Validation

Run:

```powershell
.\scripts\dev\smoke-chat-flow.ps1
```

This validates:

1. Gateway health
2. Conversation creation
3. WebSocket streaming via `/v1/ws/chat`
4. Final message persistence in Session Store

## UI Smoke

Run:

```powershell
.\scripts\dev\run-ui-smoke.ps1
```

This validates the preview UI against the fake stack:

1. Gateway health and default test agent exist before the browser opens
2. The chat composer becomes editable and sendable
3. First prompt creates a conversation
4. Assistant response streams and renders
5. Deleting the current conversation returns to `/chat`

The Playwright test avoids asserting status copy such as `Gateway Online`. It writes browser fallback settings with the preview origin as the gateway base URL, then uses the Vite proxy to reach the gateway and websocket endpoint.

## Half-Automatic E2E

Keep services and preview running in one terminal:

```powershell
.\scripts\dev\start-fake-stack.ps1 -StartPreview
```

Run Playwright in another terminal:

```powershell
Push-Location .\desktop\frontend
$env:GATEWAY_BASE_URL = "http://127.0.0.1:8080"
$env:E2E_DEFAULT_AGENT_ID = "agent_desktop_default"
$env:PLAYWRIGHT_BASE_URL = "http://127.0.0.1:4173"
npm run test:e2e
Pop-Location
```

## Stop Flow

Run:

```powershell
.\scripts\dev\stop-fake-stack.ps1
```
