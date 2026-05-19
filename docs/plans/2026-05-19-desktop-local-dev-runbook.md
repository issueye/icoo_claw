# Desktop Local Dev Runbook

## Goal

Provide a repeatable local path for validating the desktop chat client against the gateway stack without requiring a real model provider.

## Scope

- Build `session_store`, `claw`, and `gateway`
- Start `session_store` and `gateway`
- Let `gateway` auto-start `claw` instances on demand
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

## Start Flow

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
2. Start the frontend preview at `http://127.0.0.1:4173`
3. Reuse browser fallback settings storage when the app is not running inside Wails

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

1. Gateway connects in the actual frontend
2. Default Agent is selected
3. First prompt creates a conversation
4. Assistant response streams and renders
5. Deleting the current conversation returns to `/chat`

## Stop Flow

Run:

```powershell
.\scripts\dev\stop-fake-stack.ps1
```
