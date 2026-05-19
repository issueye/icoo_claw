# MVP 定义

## MVP 目标

用最小实现验证三服务架构：

- Gateway 作为唯一入口。
- Gateway 可以启动和管理多个本机 Claw Agent 服务实例。
- Claw 集成 `agentsdk-go`，并提供 fake runner 用于稳定测试。
- Session Store 使用 GORM + no-cgo SQLite 保存并恢复会话。
- 同步对话能完整跑通。

## 必须完成

Gateway:

- 创建 AgentProfile。
- 启动、停止、查询 Claw Agent 实例。
- 自动拉起 Claw 实例。
- 创建 Conversation。
- 发送同步消息。
- 提供 WebSocket 流式聊天入口。
- 查询会话消息。

Claw:

- `/internal/agent/run`。
- `/internal/agent/run/stream`。
- `pkg/agent_sdk` 封装 Runtime。
- fake runner。
- 从 Session Store 加载历史。
- 执行后保存历史 snapshot。

Session Store:

- GORM + `github.com/glebarez/sqlite`。
- Session CRUD/list。
- Message list/append/replace snapshot。
- Run list/append。
- Run event list/append。

配置:

- 三服务使用 TOML。
- Gateway 启动 Claw 实例时生成 TOML 文件。

## 暂不完成

- 额外兼容协议。
- 非 GORM 的会话持久化实现。
- 多租户/RBAC。
- 完整 MCP 工具探测与热重载。
- Skills 文件上传和版本管理。
- 同 session 队列。
- UI 管理台。
- 远程/容器化 Claw 实例调度。

## MVP API

Gateway:

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
POST   /v1/conversations
GET    /v1/conversations
GET    /v1/conversations/:id/messages
POST   /v1/conversations/:id/messages
GET    /v1/ws/chat
POST   /v1/conversations/:id/stream
DELETE /v1/conversations/:id
```

说明：

- `GET /v1/ws/chat` 是当前推荐的外部流式聊天入口。
- `POST /v1/conversations/:id/stream` 在 MVP 中可暂时保留为兼容接口，但新桌面端不再依赖它。

Claw:

```text
GET  /health
POST /internal/agent/run
POST /internal/agent/run/stream
```

Session Store:

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

## 验收用例

### 用例 1：创建会话并同步对话

1. 启动 Session Store。
2. 启动 Gateway。
3. 创建 AgentProfile。
4. 创建 Conversation。
5. 调用 `POST /v1/conversations/:id/messages`。
6. Gateway 自动拉起 fake Claw。
7. Claw 从 Session Store 加载历史。
8. Claw 保存 user/assistant snapshot。
9. Gateway 返回 assistant output。
10. 查询 messages，能看到 user 与 assistant 消息。

### 用例 2：同一会话二次对话

1. 在同一个 Conversation 发送第二条消息。
2. Claw 读取第一轮历史。
3. 保存后的消息包含两轮对话。

### 用例 3：流式对话

1. 建立 `GET /v1/ws/chat` WebSocket 连接。
2. 客户端发送 `chat.start`。
3. Gateway 返回 `session.accepted`。
4. Gateway 按增量返回 `message.delta`。
5. 流结束后返回 `message.completed`。
6. 查询 messages，能看到最终 assistant 内容。

### 用例 4：多个 Agent 服务实例

1. 启动两个 Claw 实例。
2. 查询实例状态均为 `ready`。
3. 不同 conversation 按 sticky/least inflight 分配实例。
4. 停止其中一个实例后，新请求不再路由到该实例。

## MVP 完成标准

- Windows 本地可以无 CGO 编译运行。
- 三服务 `go test ./...` 通过。
- 三服务进程级端到端测试通过。
- Session Store SQLite 文件中能恢复会话。
- 关键错误有统一 JSON 响应。
