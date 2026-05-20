# 需求分析与整理

## 背景

当前仓库目标是在 `server` 下建设两个运行时 Go 服务：

- `server/gateway`: AI 网关、控制面与会话存储 API。
- `server/claw`: AI Agent 执行服务。

技术约束：

- Go + Gin + GORM + SQLite(no cgo)。
- MVC 分层。
- 手动依赖注入。
- SQLite 驱动使用 `github.com/glebarez/sqlite`，避免 CGO。
- `server/claw` 将 `agentsdk-go` 集成到 `pkg/agent_sdk` 目录下。
- 服务启动配置使用 TOML 文件，通过 `--config` 指定。

## agentsdk-go 能力分析

`server/claw/pkg/agent_sdk/sdk` 已提供 Runtime、MCP、Skills、Subagents、Hooks、Middleware、Sandbox、同步与流式运行能力。

平台层的重点不是重写 runtime，而是在 `server/claw/pkg/agent_sdk` 做封装：

- Gateway 下发受控的 AgentProfile 和请求参数。
- Claw 在运行前从网关会话 API 加载历史。
- Claw 在运行结束后读取 runtime session snapshot 并保存到网关会话 API。
- 同 session 并发冲突在 MVP 中返回 `409 session_busy`。
- fake runner 可用于本地端到端测试，真实 runner 使用 SDK model provider。

## 会话存储能力边界

会话存储内置在 Gateway 中，使用 GORM + no-cgo SQLite 作为 MVP 唯一持久化方案。

需要保存：

- session 元数据。
- conversation message snapshot。
- agent run 摘要。
- run event，用于流式事件或排障记录。

需要支持：

- Session CRUD。
- session 列表过滤。
- message 分页、tail、before/after 游标。
- message snapshot replace，带 revision 冲突保护。
- run 与 run event append/list。

不再把额外外部协议作为 MVP 依赖。

## 核心业务需求

### Claw

- 提供内部 Agent 执行 API：同步对话、流式对话、健康检查。
- 封装 `agentsdk-go` 到 `pkg/agent_sdk`。
- 支持 TOML 配置：HTTP 地址、Session API URL、内部 token、runner mode。
- 支持从网关会话 API 加载历史并保存 snapshot。
- 支持 fake runner 与 SDK runner。
- 同 session 并发执行返回明确错误。

### Gateway

- 作为统一入口，管理控制面资源：
  - Agent profile。
  - Agent service instance 生命周期。
  - Conversation。
  - 同步与流式消息入口。
- 使用 GORM + no-cgo SQLite 保存控制面元数据。
- 能启动、停止、重启、drain 本机 Claw 实例。
- 能健康巡检 Claw 实例并维护 `starting/ready/draining/stopped/failed` 状态。
- 对话请求按 session sticky routing，并在无可用实例时按需拉起 Claw。
- Gateway 启动 Claw 时生成实例专用 TOML 配置文件，而不是通过环境变量传配置。
- 对外流式聊天入口使用 WebSocket，推荐路径 `GET /v1/ws/chat`。
- Gateway 对外使用 WebSocket 事件协议，对内到 Claw 仍可复用现有流式执行链路。

## 非功能需求

- 可维护性：MVC + service/repository 分层，手动 DI。
- 可移植性：无 CGO，便于 Windows/Linux 本地运行。
- 可观测性：日志字段包含 request/session/agent/instance/run 信息。
- 安全性：Claw 内部接口使用 internal token，工具白名单和 sandbox 由 AgentProfile 控制。
- 可靠性：会话恢复、幂等或冲突保护、实例健康检查、draining 停止。
- 测试：服务内测试 + 进程级端到端集成测试。

## 需求优先级

### P0

- Gateway 与 Claw 可启动。
- Gateway 与 Claw 健康检查通过。
- Gateway 可创建 AgentProfile 和 Conversation。
- Gateway 可启动 Claw 实例。
- Gateway 可发送同步消息。
- Claw fake runner 可完成端到端对话并保存历史。
- Gateway 会话 API 可保存并读取 messages。
- TOML 配置文件替代环境变量。

### P1

- 流式消息端到端测试。
- Gateway 外部 WebSocket 聊天协议稳定化。
- 强类型 AgentProfile contract。
- run/run event 完整写入。
- 同 session 并发保护。
- MCP/Skills/Sandbox 控制面。

### P2

- 多 Agent 编排。
- 远程/容器化 Claw 实例管理。
- 审计日志、租户隔离、RBAC。
- 外部协议适配层。
- 前端管理台。
