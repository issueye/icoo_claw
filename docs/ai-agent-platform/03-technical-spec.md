# 技术说明文档

## 技术选型

| 领域 | 选型 | 说明 |
|---|---|---|
| HTTP | Gin | 三服务统一 REST/SSE 框架 |
| ORM | GORM | Gateway 控制面元数据 |
| SQLite(no cgo) | `github.com/glebarez/sqlite` | GORM 的纯 Go SQLite driver |
| Session KV/List | Redka | Redis-like API，SQLite backend |
| Redka SQLite(no cgo) | `modernc.org/sqlite` | Redka `DriverName: "sqlite"` |
| Agent Runtime | `server/claw/pkg/agent_sdk/sdk` | agentsdk-go 源码已抽取为项目内模块，并由 `server/claw/pkg/agent_sdk` 封装 |
| DI | 手动 DI | `internal/di/container.go` 显式组装 |

说明：

- GORM 标准 SQLite driver 通常会引入 CGO 依赖；本项目 Gateway 侧建议使用 `github.com/glebarez/sqlite`。
- Redka 侧不走 GORM，使用 `modernc.org/sqlite` 注册 `database/sql` driver，并设置 `redka.Options{DriverName: "sqlite"}`。

参考：

- GORM SQLite 官方仓库说明标准 driver 使用 `github.com/mattn/go-sqlite3`，并提示 pure Go driver 可看 `github.com/glebarez/sqlite`: https://github.com/go-gorm/sqlite
- `github.com/glebarez/sqlite` Go package: https://pkg.go.dev/github.com/glebarez/sqlite
- `modernc.org/sqlite` Go package: https://pkg.go.dev/modernc.org/sqlite

## Go 模块组织

建议每个服务独立 Go module，仓库根增加 `go.work`：

```text
go.work
server/claw/go.mod
server/gateway/go.mod
server/session_store/go.mod
```

本地依赖使用 replace：

```go
replace github.com/nalgeon/redka => ../../go_pkg/redka
```

说明：agentsdk-go 不再作为第三方 module 引入；其运行时代码位于 `server/claw/pkg/agent_sdk/sdk`。

## 手动 DI 示例结构

```go
type Container struct {
    Config     config.Config
    Router     *gin.Engine
    DB         *gorm.DB
    Services   Services
    Controllers Controllers
}

func NewContainer() (*Container, error) {
    cfg := config.Load()
    db, err := openDB(cfg)
    if err != nil {
        return nil, err
    }

    repos := newRepositories(db)
    clients := newClients(cfg)
    services := newServices(repos, clients, cfg)
    controllers := newControllers(services)
    router := router.New(controllers)

    return &Container{Config: cfg, Router: router}, nil
}
```

## Gateway: Agent 服务实例管理

Gateway 需要具备本机启动多个 `server/claw` Agent 服务实例的能力。MVP 使用本地进程管理，不引入容器运行时。

### 配置项

```go
type AgentProcessConfig struct {
    BinaryPath string        // server/claw 可执行文件路径
    WorkDir string           // 进程工作目录
    PortStart int            // 例如 8101
    PortEnd int              // 例如 8199
    MaxInstances int
    StartupTimeout time.Duration
    ShutdownTimeout time.Duration
    HealthInterval time.Duration
    Env map[string]string
}
```

### AgentInstance 数据模型

```go
type AgentInstance struct {
    ID string
    AgentID string
    Name string
    Status string // starting, ready, draining, stopped, failed
    PID int
    Host string
    Port int
    BaseURL string
    StartedAt *time.Time
    LastHeartbeatAt *time.Time
    LastError string
    Inflight int
    Metadata datatypes.JSON
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### ProcessSupervisor 接口

```go
type ProcessSupervisor interface {
    Start(ctx context.Context, spec StartAgentInstanceSpec) (*AgentProcess, error)
    Stop(ctx context.Context, instance AgentInstance) error
    Restart(ctx context.Context, instance AgentInstance) (*AgentProcess, error)
    Probe(ctx context.Context, instance AgentInstance) error
}
```

启动参数建议：

```text
claw --http-addr 127.0.0.1:8101 \
     --session-store-url http://127.0.0.1:8082 \
     --internal-token ${INTERNAL_TOKEN}
```

### RouterPolicy 接口

```go
type RouterPolicy interface {
    SelectInstance(ctx context.Context, req RouteRequest) (*AgentInstance, error)
    MarkInflight(ctx context.Context, instanceID string, delta int) error
    BindSession(ctx context.Context, sessionID string, instanceID string) error
}
```

MVP 策略：

- 先查 `session_id -> instance_id` sticky mapping。
- sticky 实例仍为 `ready` 时继续使用。
- sticky 不存在或实例不可用时，从同 AgentProfile 的 ready 实例中选择 `Inflight` 最小者。
- 无可用实例时，若未超过 `MaxInstances`，按需启动一个实例。
- 启动失败返回 `503 dependency_unavailable`。

### Gateway Agent Instance API

```text
POST   /v1/agent-instances
GET    /v1/agent-instances
GET    /v1/agent-instances/:id
POST   /v1/agent-instances/:id/stop
POST   /v1/agent-instances/:id/restart
POST   /v1/agent-instances/:id/drain
```

## Claw: `pkg/agent_sdk` 封装

### 对外接口

```go
type Runner interface {
    Run(ctx context.Context, req RunRequest) (*RunResponse, error)
    RunStream(ctx context.Context, req RunRequest) (<-chan StreamEvent, error)
}

type RunRequest struct {
    SessionID string
    RequestID string
    Prompt string
    Agent AgentProfile
    ForceSkills []string
    ToolWhitelist []string
    Metadata map[string]any
}
```

### Runtime 构建映射

| 平台配置 | agentsdk-go 映射 |
|---|---|
| `agent.model_provider` | `model.AnthropicProvider` 或 `model.OpenAIProvider` |
| `agent.model_name` | Provider `ModelName` |
| `agent.system_prompt` | `api.Options.SystemPrompt` |
| `agent.project_root` | `api.Options.ProjectRoot` |
| `agent.mcp_servers` | `api.Options.MCPServers` |
| `agent.skills` | `api.Options.Skills` 或 `.agents/skills` |
| `agent.sandbox` | `api.Options.Sandbox` |
| `agent.max_iterations` | `api.Options.MaxIterations` |
| `session_id` | `api.Request.SessionID` |
| `request_id` | `api.Request.RequestID` |
| `tool_whitelist` | `api.Request.ToolWhitelist` |
| `force_skills` | `api.Request.ForceSkills` |

### 历史加载与保存

`agentsdk-go` 初始化时：

```go
api.Options{
    HistoryLoader: func(sessionID string) ([]message.Message, error) {
        return historyAdapter.Load(ctx, sessionID)
    },
}
```

运行结束后：

```go
snapshot, ok := runtime.SessionHistory(sessionID)
if ok {
    _ = historyAdapter.SaveSnapshot(ctx, sessionID, requestID, snapshot)
}
```

注意：

- 保存必须是幂等操作，建议 `SaveSnapshot` 采用“覆盖该 session 的 messages list”或基于 `run_id` 的 compare-and-swap。
- MVP 选择覆盖快照，简单可靠。
- 生产阶段可改为增量 append + message_id 去重。

## Session Store: Redka 封装

### 启动 Redka DB

```go
import (
    "github.com/nalgeon/redka"
    "github.com/nalgeon/redka/redsrv"
    _ "modernc.org/sqlite"
)

db, err := redka.Open("session_store.sqlite", &redka.Options{
    DriverName: "sqlite",
})
srv := redsrv.New("tcp", ":6380", db)
```

### HTTP API 草案

```text
GET    /health
POST   /v1/sessions
GET    /v1/sessions/:session_id
PATCH  /v1/sessions/:session_id
DELETE /v1/sessions/:session_id
GET    /v1/sessions/:session_id/messages?limit=50&before=
POST   /v1/sessions/:session_id/messages
PUT    /v1/sessions/:session_id/messages/snapshot
POST   /v1/sessions/:session_id/runs
GET    /v1/sessions/:session_id/runs
```

### 消息结构

```json
{
  "id": "msg_...",
  "role": "user|assistant|system|tool",
  "content": "...",
  "content_blocks": [],
  "tool_calls": [],
  "metadata": {},
  "created_at": "2026-05-17T22:00:00+08:00"
}
```

### Repository 方法

```go
type SessionRepository interface {
    Create(ctx context.Context, s Session) error
    Get(ctx context.Context, sessionID string) (*Session, error)
    Update(ctx context.Context, s Session) error
    Delete(ctx context.Context, sessionID string) error
    ListMessages(ctx context.Context, sessionID string, page Page) ([]Message, error)
    AppendMessages(ctx context.Context, sessionID string, messages []Message) error
    ReplaceMessages(ctx context.Context, sessionID string, messages []Message) error
}
```

Redka 方法映射：

- `Hash().SetMany` 保存 meta。
- `Hash().Items` 读取 meta。
- `List().RPush` 追加消息。
- `List().Range` 分页读取消息。
- `Key().Delete` 删除 session 相关 key。
- `ZSet().Add` 维护会话索引。

## Gateway: 控制面数据模型

### AgentProfile

```go
type AgentProfile struct {
    ID string
    Name string
    ModelProvider string
    ModelName string
    BaseURL string
    SystemPrompt string
    MaxIterations int
    ToolWhitelist datatypes.JSONSlice[string]
    MCPServerIDs datatypes.JSONSlice[string]
    SkillIDs datatypes.JSONSlice[string]
    SandboxPolicy datatypes.JSON
    Enabled bool
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### AgentInstance

```go
type AgentInstance struct {
    ID string
    AgentID string
    Status string
    PID int
    Host string
    Port int
    BaseURL string
    LastHeartbeatAt *time.Time
    LastError string
    Inflight int
}
```

### MCPServer

```go
type MCPServer struct {
    ID string
    Name string
    Spec string
    Transport string
    Enabled bool
    LastStatus string
    LastCheckedAt *time.Time
    ToolCache datatypes.JSON
}
```

### Skill

```go
type Skill struct {
    ID string
    Name string
    Description string
    SourceType string
    SourcePath string
    Enabled bool
    Metadata datatypes.JSON
}
```

### Conversation

Gateway 只保存控制面索引与展示字段，消息正文以 Session Store 为准：

```go
type Conversation struct {
    ID string
    SessionID string
    AgentID string
    UserID string
    Title string
    Status string
    LastMessageAt *time.Time
}
```

## Gateway API 草案

```text
GET    /health

POST   /v1/agents
GET    /v1/agents
GET    /v1/agents/:id
PATCH  /v1/agents/:id
DELETE /v1/agents/:id

POST   /v1/agent-instances
GET    /v1/agent-instances
POST   /v1/agent-instances/:id/stop
POST   /v1/agent-instances/:id/restart
POST   /v1/agent-instances/:id/drain

POST   /v1/mcp/servers
GET    /v1/mcp/servers
POST   /v1/mcp/servers/:id/check
GET    /v1/mcp/servers/:id/tools

POST   /v1/skills
GET    /v1/skills
PATCH  /v1/skills/:id
DELETE /v1/skills/:id

POST   /v1/conversations
GET    /v1/conversations
GET    /v1/conversations/:id/messages
POST   /v1/conversations/:id/messages
POST   /v1/conversations/:id/stream
DELETE /v1/conversations/:id
```

## 错误码约定

| HTTP | code | 说明 |
|---|---|---|
| 400 | `bad_request` | 入参错误 |
| 401 | `unauthorized` | 未认证 |
| 403 | `forbidden` | 工具、MCP、agent 权限不允许 |
| 404 | `not_found` | 资源不存在 |
| 409 | `session_busy` | 同 session 正在执行 |
| 413 | `payload_too_large` | 请求或消息过大 |
| 502 | `agent_error` | Claw/模型/工具执行失败 |
| 503 | `dependency_unavailable` | Session Store 或 Claw 不可用 |

## 安全与权限

- Gateway 对 MCP spec 做 allowlist 校验，禁止任意命令执行。
- Claw 只接受 Gateway 内网调用，生产环境加内部 token/mTLS。
- Claw 的 `ProjectRoot`、sandbox allowed paths、network allow 必须由 AgentProfile 控制。
- 默认禁用危险内置工具；按 agent profile 显式开放。
- 每个请求设置 body size limit、timeout、request_id。

## 可观测性

- 所有日志字段包含 `request_id`、`session_id`、`agent_id`、`run_id`。
- Claw 透传 SDK `RequestID`，并启用 `api.Options.OTEL` 扩展点。
- Gateway 记录对话入口耗时、Claw 调用耗时、Session Store 调用耗时。
- Session Store 记录 Redka 操作错误、消息数量、快照大小。

## 测试策略

- Unit：
  - DTO validation。
  - service 编排。
  - repository Redka key 编码。
  - agent_sdk history adapter。
- Integration：
  - Session Store 使用临时 SQLite 文件。
  - Claw 使用 fake model 或 SDK demo model。
  - Gateway -> Claw -> Session Store 全链路。
- Contract：
  - Gateway 与 Claw DTO。
  - Claw 与 Session Store DTO。
