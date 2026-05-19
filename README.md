# icoo_claw

AI Agent platform prototype with three Go services:

- `server/gateway`: public API gateway and control plane.
- `server/claw`: internal Agent execution service using `agentsdk-go`.
- `server/session_store`: session storage service using GORM over no-cgo SQLite.

## Local Build

```powershell
New-Item -ItemType Directory -Force .\bin, .\data
go build -o .\bin\claw.exe .\server\claw\cmd\claw
go build -o .\bin\gateway.exe .\server\gateway\cmd\gateway
go build -o .\bin\session_store.exe .\server\session_store\cmd\session_store
```

Copy the example TOML files before starting services:

```powershell
Copy-Item .\config\gateway.toml.example .\config\gateway.toml
Copy-Item .\config\session_store.toml.example .\config\session_store.toml
Copy-Item .\config\claw.toml.example .\config\claw.toml
```

Gateway can launch additional Claw instances using `claw_binary_path` from `config/gateway.toml`.

## Local Run

Start Session Store:

```powershell
.\bin\session_store.exe --config .\config\session_store.toml
```

Start Gateway:

```powershell
.\bin\gateway.exe --config .\config\gateway.toml
```

Gateway can start Claw instances through:

```text
POST /v1/agent-instances
```

For direct Claw development:

```powershell
.\bin\claw.exe --config .\config\claw.toml
```

## Tests

```powershell
go test ./server/gateway/...
go test ./server/claw/...
go test ./server/session_store/...
```

Final delivery verification also includes the desktop app and UI smoke path:

```powershell
Push-Location .\desktop\frontend
npm run test
npm run build
Pop-Location

Push-Location .\desktop
go test ./...
Pop-Location

Push-Location .\server\gateway
go test ./...
Pop-Location

Push-Location .\server\claw
go test ./...
Pop-Location

Push-Location .\server\session_store
go test ./...
Pop-Location

.\scripts\dev\run-ui-smoke.ps1
```

Delivery boundary: generated runtime data under `.local/`, frontend build output under `desktop/frontend/dist/`, and Playwright transient output under `desktop/frontend/test-results/` or `desktop/frontend/playwright-report/` are ignored and should not be committed. External/vendor checkout content under `go_pkg/redka/` is also outside this desktop delivery phase.

## Desktop Chat E2E

The repeatable local path uses fake model execution, so no external model provider is required.

First install frontend dependencies and Playwright browsers:

```powershell
Push-Location .\desktop\frontend
npm ci
npx playwright install
Pop-Location
```

One-command smoke validation:

```powershell
.\scripts\dev\run-ui-smoke.ps1
```

That script builds the Go services, starts `session_store`, `gateway`, a default fake agent instance, builds the desktop frontend, starts Vite preview at `http://127.0.0.1:4173`, runs Playwright, then stops the local stack.

By default the fake stack refreshes its own SQLite files under `.local/fake-stack/data/` so reruns do not inherit stale agent instances. Add `-PreserveData` to `start-fake-stack.ps1` when you intentionally want to keep local conversations.

For a half-automatic run where you keep the stack open:

```powershell
.\scripts\dev\start-fake-stack.ps1 -StartPreview
Push-Location .\desktop\frontend
$env:GATEWAY_BASE_URL = "http://127.0.0.1:8080"
$env:E2E_DEFAULT_AGENT_ID = "agent_desktop_default"
$env:PLAYWRIGHT_BASE_URL = "http://127.0.0.1:4173"
npm run test:e2e
Pop-Location
.\scripts\dev\stop-fake-stack.ps1
```

Runtime files and logs are written to `.local/fake-stack/`.
