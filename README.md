# icoo_claw

AI Agent platform prototype with three Go services:

- `server/gateway`: public API gateway and control plane.
- `server/claw`: internal Agent execution service using `agentsdk-go`.
- `server/session_store`: session storage service using Redka over pure-Go SQLite.

## Local Build

```powershell
New-Item -ItemType Directory -Force .\bin, .\data
go build -o .\bin\claw.exe .\server\claw\cmd\claw
go build -o .\bin\gateway.exe .\server\gateway\cmd\gateway
go build -o .\bin\session_store.exe .\server\session_store\cmd\session_store
```

Copy `.env.example` values into your shell before starting services. Gateway can launch additional Claw instances using `CLAW_BINARY_PATH`.

## Local Run

Start Session Store:

```powershell
$env:SESSION_STORE_DB_PATH="./data/session_store.sqlite"
.\bin\session_store.exe
```

Start Gateway:

```powershell
$env:GATEWAY_DB_PATH="./data/gateway.sqlite"
$env:SESSION_STORE_URL="http://127.0.0.1:8082"
$env:CLAW_BINARY_PATH="./bin/claw.exe"
$env:INTERNAL_TOKEN="dev-internal-token"
.\bin\gateway.exe
```

Gateway can start Claw instances through:

```text
POST /v1/agent-instances
```

For direct Claw development:

```powershell
$env:CLAW_HTTP_ADDR=":8081"
$env:SESSION_STORE_URL="http://127.0.0.1:8082"
$env:INTERNAL_TOKEN="dev-internal-token"
.\bin\claw.exe
```

## Tests

```powershell
go test ./server/gateway/...
go test ./server/claw/...
go test ./server/session_store/...
```
