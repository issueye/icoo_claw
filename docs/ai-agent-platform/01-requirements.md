# 需求分析与整理

## 背景

当前仓库已有两个本地 Go 包：

- `server/claw/pkg/agent_sdk/sdk`: 从 agentsdk-go 抽取进项目内的 Agent SDK 源码模块，提供 Claude Code 风格运行时能力。
- `go_pkg/redka`: Redis-compatible 数据服务，可基于 SQLite/PostgreSQL 持久化，也可嵌入为 RESP 服务。

目标是在 `server` 下建设三个服务：

- `server/claw`: AI Agent 执行服务。
- `server/gateway`: AI 网关与控制面。
- `server/session_store`: AI 会话存储服务。

技术约束：

- Go + Gin + GORM + SQLite(no cgo)。
- MVC 分层。
- 手动依赖注入。
- SQLite 驱动必须避免 CGO。
- `server/claw` 将 `agentsdk-go` 集成到 `pkg/agent_sdk` 目录下。
- `server/session_store` 使用 `go_pkg/redka` 生成 Redis-compatible 服务，并封装面向会话的增删改查。

## agentsdk-go 能力分析

源码依据：

- Runtime 入口：`server/claw/pkg/agent_sdk/sdk/api/agent.go`
- Options/Request/Response：`server/claw/pkg/agent_sdk/sdk/api/options.go`
- 流式事件：`server/claw/pkg/agent_sdk/sdk/api/stream.go`
- MCP 桥接：`server/claw/pkg/agent_sdk/sdk/mcp/mcp.go`
- Skills：`server/claw/pkg/agent_sdk/sdk/runtime/skills`
- Subagents：`server/claw/pkg/agent_sdk/sdk/runtime/subagents`

已具备能力：

- `api.New(ctx, api.Options)` 创建 Runtime。
- `Runtime.Run(ctx, api.Request)` 同步执行。
- `Runtime.RunStream(ctx, api.Request)` 返回事件通道，可转 SSE。
- `Request` 支持 `SessionID`、`RequestID`、模型档位、traits/tags/channels、团队成员、目标 subagent、工具白名单、强制 skills 等。
- `Options` 支持模型工厂、MCPServers、Skills、Subagents、Middleware、Hooks、Sandbox、AutoCompact、OTEL、HistoryLoader。
- SDK 内部维护会话历史，支持 `Runtime.SessionHistory(sessionID)` 快照。
- 相同 `SessionID` 并发执行会返回 `ErrConcurrentExecution`，平台层需要做串行化、排队或返回 `409 session_busy`。
- Skills 文件默认从项目 `.agents/skills` 加载，也支持代码注册。
- MCP 支持 `stdio://`、SSE/HTTP transport 形式，能把 MCP tools 注册进 SDK 工具系统。

限制与影响：

- SDK 只有 `HistoryLoader`，没有持久化回调；`claw/pkg/agent_sdk` 必须在运行结束后读取 `SessionHistory` 并写回 `session_store`。
- SDK 的会话历史是内存态；服务重启后依赖 `session_store` 还原。
- SDK 的 runtime 配置较多，网关下发的 Agent 配置需要转换为受控的 `api.Options` / `api.Request`，避免把任意文件系统、网络、工具权限直接暴露给用户。

## redka 能力分析

源码依据：

- DB 入口：`go_pkg/redka/redka.go`
- 纯 Go SQLite 示例：`go_pkg/redka/example/modernc/main.go`
- 嵌入 RESP 服务示例：`go_pkg/redka/example/server/main.go`
- RESP 服务：`go_pkg/redka/redsrv/server.go`
- Module 使用说明：`go_pkg/redka/docs/usage-module.md`

已具备能力：

- `redka.Open(path, &redka.Options{DriverName: "sqlite"})` 可配合 `modernc.org/sqlite` 使用纯 Go SQLite。
- `redsrv.New("tcp", addr, db)` 可启动 Redis-compatible RESP 服务。
- 支持 Strings、Hashes、Lists、Sets、Sorted Sets、Keys、Transactions。
- DB 并发安全，建议单进程内复用一个 DB 实例。

限制与影响：

- Redka 不支持 Redis Streams；会话消息应使用 List，元数据使用 Hash。
- Redka 性能目标不是替代高吞吐 Redis，MVP 应控制消息大小、分页读取与清理策略。
- 若 `session_store` 同时对外提供 RESP 与 HTTP API，需要明确 HTTP API 是平台主接口，RESP 主要用于兼容 Redis 客户端和测试。

## 核心业务需求

### AI Agent 服务 `server/claw`

- 提供 Agent 执行 API：同步对话、流式对话、健康检查。
- 封装 `agentsdk-go` 到 `pkg/agent_sdk`，屏蔽 SDK 细节。
- 支持从 `session_store` 加载历史并在执行后持久化历史。
- 支持从网关传入 agent 配置：模型、系统提示词、工具白名单、MCP servers、skills、sandbox、超时、最大迭代数等。
- 支持 SSE 流式输出，并在流结束后保存会话历史。
- 对同一 session 并发执行给出明确策略：MVP 返回 `409 session_busy`，后续可接入队列。

### AI 网关 `server/gateway`

- 作为统一入口，管理控制面资源：
  - MCP 管理：MCP server 配置、连通性检测、启停状态、工具列表缓存。
  - Skills 管理：skills 元数据、启用/禁用、目录同步、版本记录。
  - Agent 管理：agent profile、模型配置、系统提示词、工具权限、sandbox 策略。
  - Agent 服务实例管理：启动、停止、重启、健康检查多个 `server/claw` 进程，维护实例池和路由状态。
  - 对话管理：创建会话、发送消息、流式转发、会话列表、消息查询、归档/删除。
- 使用 GORM + SQLite(no cgo) 存储控制面元数据。
- 对外提供 REST + SSE API。
- 根据会话、AgentProfile、实例健康状态和负载策略选择一个 `claw` 实例执行 Agent，调用 `session_store` 管理会话状态。
- 保留后续鉴权、限流、审计、租户隔离的扩展点。

#### Agent 服务实例管理

- Gateway 持有 `server/claw` 可执行文件路径、端口范围、工作目录、环境变量模板和最大实例数配置。
- Gateway 可按 AgentProfile 手动启动一个或多个 Claw 实例，也可在对话请求到达时按需拉起实例。
- Gateway 为每个实例分配唯一 `instance_id`、HTTP 地址、端口、PID、绑定的 AgentProfile 或标签。
- Gateway 定期调用实例 `/health`，将实例标记为 `starting`、`ready`、`draining`、`stopped`、`failed`。
- Gateway 路由对话时优先选择 `ready` 实例；同一 `session_id` 默认使用一致性 hash 或 sticky mapping 路由到同一实例，降低 SDK 内存 history 与并发冲突成本。
- Gateway 停止实例时先进入 `draining`，不再接收新请求，等待进行中的流式/同步请求结束后再终止进程。
- MVP 仅要求单机本地多进程管理；P2 再扩展远程节点、容器编排或 Kubernetes。

### 会话存储服务 `server/session_store`

- 使用 Redka + SQLite(no cgo) 存储会话元数据、消息、运行事件、摘要等。
- 启动 Redis-compatible RESP 服务端口。
- 封装一层 HTTP API，提供会话 CRUD、消息追加、消息分页、历史覆盖保存、TTL/归档。
- 为 `agentsdk-go` 的 `HistoryLoader` 提供 message 格式转换能力。
- 支持幂等写入，避免流式结束重试导致重复消息。

## 非功能需求

- 可维护性：MVC + service/repository 分层，手动 DI，避免全局变量。
- 可移植性：无 CGO，便于 Windows、Linux、小容器构建。
- 可观测性：请求 ID、会话 ID、Agent run ID、日志、基础 metrics、SDK OTEL 扩展点。
- 安全性：工具白名单、MCP allowlist、sandbox 路径和网络域名控制、输入大小限制。
- 可靠性：会话恢复、幂等持久化、服务健康检查、超时控制。
- 测试：单元测试覆盖 service/repository；集成测试覆盖 Gateway -> Claw -> Session Store。

## 需求优先级

### P0

- 三服务可启动。
- Gateway 可启动至少一个 Claw agent 服务实例，并通过健康检查确认可用。
- Gateway 可创建会话并发送同步/流式消息。
- Claw 可调用 agentsdk-go Runtime。
- Session Store 可用 Redka 保存/读取会话消息。
- GORM SQLite 全部使用 no-cgo driver。
- 基础健康检查与错误码。

### P1

- Gateway 可启动、停止、重启多个 Claw agent 服务实例，并维护实例状态。
- Gateway 对会话请求支持 sticky routing 到可用 Claw 实例。
- MCP 配置管理与工具列表缓存。
- Skills 元数据管理与启用/禁用。
- Agent profile 管理。
- 会话分页、归档、删除、摘要字段。
- SDK HistoryLoader 与运行后持久化闭环。

### P2

- 多 Agent 编排、队列化同 session 请求。
- 远程/容器化 Claw 实例管理、弹性伸缩和实例调度。
- 审计日志、租户隔离、RBAC。
- OTEL 全链路追踪。
- 前端管理台。
