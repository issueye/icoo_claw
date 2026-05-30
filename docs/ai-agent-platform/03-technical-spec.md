# 技术说明文档

## 技术选型

| 领域 | 选型 | 说明 |
|---|---|---|
| HTTP / Stream | Gin + Gorilla WebSocket | Gateway/Claw REST，Gateway 对外流式使用 WebSocket |
| ORM | GORM | Gateway 控制面与会话业务数据 |
| SQLite(no cgo) | `github.com/glebarez/sqlite` | GORM 纯 Go SQLite driver |
| Agent Runtime | `common/core/agent_sdk` | agentsdk-go 源码已抽取为共享模块 |
| Config | TOML | 三服务使用 `--config <file>` |
| DI | 手动 DI | `internal/di/container.go` 显式组装 |

不使用 `mattn/go-sqlite3`，避免 CGO。

## Go 模块组织

```text
go.work
server/gateway/go.mod
server/claw/go.mod
```

`server/gateway` 使用 `github.com/glebarez/sqlite`。

## 配置文件

示例文件：

- `config/gateway.toml.example`
- `config/claw.toml.example`

启动示例：

```powershell
.\bin\gateway.exe --config .\config\gateway.toml
.\bin\claw.exe --config .\config\claw.toml
```

Gateway 启动 Claw 实例时会生成实例 TOML：

```toml
http_addr = "127.0.0.1:8101"
session_api_url = "http://127.0.0.1:8080"
internal_token = "dev-internal-token"
runner_mode = "fake"
```

## Gateway: Agent 实例管理

配置项：

```go
type Config struct {
    HTTPAddr string
    DBPath string
    ClawBinaryPath string
    ClawWorkDir string
    ClawConfigDir string
    ClawRunnerMode string
    ClawPortStart int
    ClawPortEnd int
    MaxAgentInstances int
    HealthInterval time.Duration
    ShutdownTimeout time.Duration
    SessionAPIURL string
    InternalToken string
}
```

实例生命周期：

- `Start`: 分配端口、生成 Claw TOML、启动进程、等待健康检查。
- `ProbeInstances`: 巡检 `starting/ready/draining` 实例。
- `Stop`: 先置为 `draining`，等待 inflight 清零，再停止进程。
- `Restart`: 停止旧进程并启动新实例。

RouterPolicy：

- 选 sticky ready 实例。
- sticky 不可用时选 inflight 最少实例。
- 无 ready 实例时自动拉起。
- 选实例前刷新健康状态。

Gateway 外部流式协议：

```text
GET /v1/ws/chat
```

客户端消息：

```json
{"type":"ping"}
{"type":"chat.start","conversation_id":"conv_1","request_id":"req_1","prompt":"hello","metadata":{}}
{"type":"chat.cancel","conversation_id":"conv_1","request_id":"req_1"}
```

服务端消息：

```json
{"type":"pong"}
{"type":"session.accepted","conversation_id":"conv_1","request_id":"req_1"}
{"type":"message.delta","conversation_id":"conv_1","session_id":"sess_1","request_id":"req_1","output":"hello"}
{"type":"message.completed","conversation_id":"conv_1","session_id":"sess_1","request_id":"req_1","stop_reason":"end_turn"}
{"type":"message.error","conversation_id":"conv_1","request_id":"req_1","code":"session_busy","error":"session is already running"}
{"type":"cancel.accepted","conversation_id":"conv_1","request_id":"req_1"}
```

约束：

- 单个 WebSocket 连接一次只允许一个活动请求。
- `chat.cancel` 只接受取消当前活动请求，不直接产出最终消息，最终完成事件仍由流生命周期结束时发出。
- Gateway 对外使用 WebSocket，但对内仍可通过既有 `RunStream` / `Stream()` 管线与 Claw 交互。

## Claw: Agent SDK 封装

接口：

```go
type Runner interface {
    Run(ctx context.Context, req RunRequest) (*RunResponse, error)
    RunStream(ctx context.Context, req RunRequest) (<-chan StreamEvent, error)
}
```

`RunRequest` 当前仍以 `map[string]any` 接收 AgentProfile。P1 应改为强类型 contract。

运行模式：

- `runner_mode = "sdk"`: 使用 agentsdk-go runtime。
- `runner_mode = "fake"`: 使用 fake runner，便于端到端测试。

历史策略：

- `HistoryAdapter.Load` 从 Gateway 会话 API 读取 messages。
- `HistoryAdapter.SaveSnapshot` 调用 Gateway 会话 API 的 snapshot 接口。
- snapshot 使用 revision 冲突保护，避免覆盖并发更新。

## Gateway 会话 API: GORM SQLite

HTTP API：

```text
GET    /health
POST   /v1/sessions
GET    /v1/sessions
GET    /v1/sessions/:session_id
PATCH  /v1/sessions/:session_id
DELETE /v1/sessions/:session_id
GET    /v1/sessions/:session_id/messages
POST   /v1/sessions/:session_id/messages
PUT    /v1/sessions/:session_id/messages/snapshot
GET    /v1/sessions/:session_id/runs
POST   /v1/sessions/:session_id/runs
GET    /v1/sessions/:session_id/runs/:run_id/events
POST   /v1/sessions/:session_id/runs/:run_id/events
```

核心表：

- `session_records`
- `message_records`
- `run_records`
- `run_event_records`

Repository 方法：

```go
type SessionRepository interface {
    Create(ctx context.Context, session model.Session) error
    Get(ctx context.Context, sessionID string) (*model.Session, error)
    List(ctx context.Context, filter model.SessionListFilter) (*model.SessionList, error)
    Update(ctx context.Context, session model.Session) error
    Delete(ctx context.Context, sessionID string) error
    ListMessages(ctx context.Context, sessionID string, page model.MessagePage) (*model.MessageList, error)
    AppendMessages(ctx context.Context, sessionID string, messages []model.Message) error
    ReplaceMessages(ctx context.Context, sessionID string, messages []model.Message, expectedRevision *int64) error
    ListRuns(ctx context.Context, sessionID string, page model.RunPage) ([]model.Run, error)
    AppendRuns(ctx context.Context, sessionID string, runs []model.Run) error
    ListRunEvents(ctx context.Context, sessionID string, runID string, page model.RunEventPage) ([]model.RunEvent, error)
    AppendRunEvents(ctx context.Context, sessionID string, runID string, events []model.RunEvent) error
}
```

## 错误码约定

| HTTP | code | 说明 |
|---|---|---|
| 400 | `bad_request` | 入参错误 |
| 401 | `unauthorized` | 内部 token 错误 |
| 404 | `not_found` | 资源不存在 |
| 409 | `revision_conflict` / `session_busy` | revision 或 session 并发冲突 |
| 502 | `agent_error` / `store_error` | 执行或持久化失败 |
| 503 | `dependency_unavailable` | 无可用 Claw |

## 测试策略

- Unit: config loader、repository、service、client。
- Router: Gateway/Claw HTTP route 基础行为。
- E2E: Gateway 模块根的进程级测试构建并启动 Gateway 与 Claw，使用 Claw fake runner 完成创建 agent、创建 conversation、发送消息、查询历史。
- WebSocket: Gateway controller 测试覆盖 `chat.start`、`ping`、错误传播、增量消息转发。
