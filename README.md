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
