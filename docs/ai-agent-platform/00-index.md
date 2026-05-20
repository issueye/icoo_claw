# AI Agent Platform 文档索引

本文档集基于已抽取到 `server/claw/pkg/agent_sdk/sdk` 的 agentsdk-go 源码模块，以及当前 `server/claw` 与 `server/gateway` 两个运行时服务的 MVP 与后续演进设计。

## 文档列表

- [01-requirements.md](01-requirements.md): 需求分析与需求整理。
- [02-architecture.md](02-architecture.md): 项目架构、模块边界、目录结构与关键链路。
- [03-technical-spec.md](03-technical-spec.md): Go/Gin/GORM/SQLite(no cgo)、agentsdk-go 集成说明与接口草案。
- [04-development-plan.md](04-development-plan.md): 多 worker 并行开发计划、依赖关系与验收标准。
- [05-mvp.md](05-mvp.md): MVP 范围、交付清单、接口、数据模型与不做事项。
- [gateway-session-gorm-models.md](gateway-session-gorm-models.md): Gateway Session API GORM + SQLite(no cgo) 明细模型设计。

## 关键结论

- `agentsdk-go` 已经具备 Agent Runtime、MCP、Skills、Subagents、Hooks、Middleware、Sandbox、同步与流式运行能力，`server/claw` 不应重写这些能力，而应在 `pkg/agent_sdk` 做平台封装。
- `agentsdk-go` 目前提供 `HistoryLoader` 和 `SessionHistory(sessionID)`，但没有外置 history saver 接口；平台层需要在 `Run/RunStream` 前加载历史，在运行结束后主动拉取快照并持久化到网关内置会话库。
- 网关负责控制面与会话 API：MCP 管理、Skills 管理、Agent 管理、Agent 服务实例生命周期、对话管理、鉴权、限流、审计、路由，以及会话状态与消息持久化；`claw` 负责 Agent 执行面。
- Gateway 可以启动并管理多个 `server/claw` Agent 服务实例，形成本机 agent instance pool，并按会话、agent profile 或负载策略路由请求。

