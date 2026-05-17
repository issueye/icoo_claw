# MVP 定义

## MVP 目标

用最小实现验证三服务架构：

- Gateway 作为唯一入口。
- Gateway 可以启动和管理多个本机 Claw Agent 服务实例。
- Claw 集成 `agentsdk-go` 并执行一次 Agent 对话。
- Session Store 使用 Redka + SQLite(no cgo) 保存并恢复会话。
- 同步与流式对话都能跑通。

## MVP 范围

### 必须完成

- 三服务独立启动。
- 三服务健康检查。
- Gateway:
  - 创建 AgentProfile。
  - 启动、停止、查询 Claw Agent 实例。
  - 至少支持两个 Claw 实例同时运行。
  - 创建 Conversation。
  - 发送同步消息。
  - 发送流式消息。
  - 查询会话消息。
- Claw:
  - `/internal/agent/run`。
  - `/internal/agent/run/stream`。
  - `pkg/agent_sdk` 封装 Runtime。
  - 从 Session Store 加载历史。
  - 执行后保存历史快照。
- Session Store:
  - Redka DB。
  - Redis-compatible RESP server。
  - Session CRUD。
  - Message list/replace snapshot。
- SQLite(no cgo):
  - Gateway GORM 使用 pure-Go driver。
  - Redka 使用 `modernc.org/sqlite`。

### 暂不完成

- 多租户/RBAC。
- 完整 MCP 工具探测与热重载。
- Skills 文件上传和版本管理。
- Claw 内部 Agent Runtime 池化。
- 同 session 队列。
- UI 管理台。
- 远程/容器化 Claw 实例调度与服务发现。
- 完整 OTEL collector 部署。

## MVP API

### Gateway

```text
GET  /health
POST /v1/agents
GET  /v1/agents/:id
POST /v1/agent-instances
GET  /v1/agent-instances
POST /v1/agent-instances/:id/stop
POST /v1/agent-instances/:id/restart
POST /v1/conversations
GET  /v1/conversations/:id/messages
POST /v1/conversations/:id/messages
POST /v1/conversations/:id/stream
```

### Claw

```text
GET  /health
POST /internal/agent/run
POST /internal/agent/run/stream
```

### Session Store

```text
GET  /health
POST /v1/sessions
GET  /v1/sessions/:session_id
DELETE /v1/sessions/:session_id
GET  /v1/sessions/:session_id/messages
POST /v1/sessions/:session_id/messages
PUT  /v1/sessions/:session_id/messages/snapshot
```

## MVP 数据结构

### AgentProfile

```json
{
  "id": "agent_default",
  "name": "Default Agent",
  "model_provider": "openai|anthropic",
  "model_name": "gpt-4.1-mini",
  "base_url": "",
  "system_prompt": "You are a helpful coding agent.",
  "max_iterations": 8,
  "tool_whitelist": ["read", "grep", "glob"],
  "mcp_server_ids": [],
  "skill_ids": [],
  "enabled": true
}
```

### Conversation

```json
{
  "id": "conv_...",
  "session_id": "sess_...",
  "agent_id": "agent_default",
  "title": "New conversation",
  "status": "active"
}
```

### AgentInstance

```json
{
  "id": "inst_...",
  "agent_id": "agent_default",
  "status": "ready",
  "pid": 12345,
  "host": "127.0.0.1",
  "port": 8101,
  "base_url": "http://127.0.0.1:8101",
  "inflight": 0
}
```

### Claw Run Request

```json
{
  "session_id": "sess_...",
  "request_id": "req_...",
  "prompt": "hello",
  "agent": {},
  "tool_whitelist": ["read", "grep", "glob"],
  "force_skills": []
}
```

## 验收用例

### 用例 1：创建会话并同步对话

1. 创建 AgentProfile。
2. Gateway 启动一个 Claw Agent 实例，并确认 `/health` ready。
3. 创建 Conversation。
4. 调用 `POST /v1/conversations/:id/messages`。
5. Gateway 选择 ready Claw 实例。
6. Claw 从 Session Store 加载空历史。
7. Claw 执行 Agent。
8. Claw 保存历史快照。
9. Gateway 返回 assistant output。
10. 查询 messages，能看到 user 与 assistant 消息。

### 用例 2：同一会话二次对话

1. 在同一个 Conversation 发送第二条消息。
2. Claw `HistoryLoader` 读取第一轮历史。
3. 模型请求上下文包含上一轮 user/assistant。
4. 保存后的消息包含两轮对话。

### 用例 3：流式对话

1. 调用 `POST /v1/conversations/:id/stream`。
2. Gateway 返回 SSE。
3. Claw 转发 SDK `StreamEvent`。
4. 流结束后查询 messages，assistant 最终内容已保存。

### 用例 4：同 session 并发

1. 同时向同一个 Conversation 发送两次请求。
2. 其中一个请求执行。
3. 另一个请求返回 `409 session_busy`。

### 用例 5：多个 Agent 服务实例

1. 调用 `POST /v1/agent-instances` 启动两个 Claw 实例。
2. 查询 `GET /v1/agent-instances`，两个实例状态均为 `ready`。
3. 创建两个不同 conversation 并发送消息。
4. Gateway 按 least inflight 或 sticky session 分配到可用实例。
5. 停止其中一个实例后，新请求不再路由到该实例。

## MVP 完成标准

- Windows 本地可以无 CGO 编译运行。
- 三服务 `go test ./...` 通过。
- fake model 集成测试稳定通过。
- 配真实 API key 可完成一次手动同步对话。
- Gateway 可启动、健康检查、停止至少两个 Claw 本地进程。
- Session Store SQLite 文件中能恢复会话。
- 关键错误有统一 JSON 响应。
