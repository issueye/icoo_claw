# 桌面端唤醒网关流程分析

## 1. 文档目的

本文基于当前 `icoo_claw` 项目代码，梳理桌面端在什么时机、通过什么调用链唤醒网关，以及网关在收到聊天请求后如何进一步拉起对应的 Agent 实例。

本文重点回答两个问题：

1. 桌面端应用如何唤醒 Gateway 进程
2. Gateway 如何在聊天时唤醒对应的执行实例

---

## 2. 结论概览

当前项目中的“桌面端唤醒网关”实际分为两层：

### 2.1 第一层：桌面端唤醒 Gateway 进程

由桌面端 Go 层负责。

触发时机不是用户发送消息瞬间，而是桌面端启动后前端执行网关数据刷新时。如果前端发现 `Gateway` 不可达，会通过 Wails 暴露的系统服务调用本地 Go 逻辑尝试拉起网关进程。

### 2.2 第二层：Gateway 唤醒对应 Agent 实例

由 Gateway 服务负责。

当用户发送消息后，如果当前会话对应的 Agent 没有可用的 `ready` 实例，Gateway 会在路由选择阶段自动启动一个新的 Agent Instance，然后再将请求转发给该实例。

因此完整链路是：

`桌面端启动 -> 健康检查失败 -> 拉起 Gateway -> 用户发送消息 -> Gateway 拉起 Agent Instance -> 开始流式对话`

---

## 3. 相关核心文件

### 3.1 桌面端相关

- `desktop/frontend/src/stores/app.js`
- `desktop/frontend/src/stores/chat.js`
- `desktop/frontend/src/services/wails/config.js`
- `desktop/systemservice.go`
- `desktop/runtime_bootstrap.go`
- `desktop/frontend/src/views/SettingsView.vue`
- `desktop/internal/config/config.go`

### 3.2 Gateway 相关

- `server/gateway/internal/service/chat_service.go`
- `server/gateway/internal/service/router_policy.go`
- `server/gateway/internal/service/agent_instance_service.go`
- `server/gateway/internal/controller/chat_ws_controller.go`
- `server/gateway/internal/controller/agent_controller.go`

---

## 4. 桌面端唤醒 Gateway 的入口流程

## 4.1 应用启动后执行 bootstrap

桌面前端在应用初始化时会调用 `bootstrap()`，入口在：

- `desktop/frontend/src/stores/app.js`

`bootstrap()` 的核心流程是：

1. 加载桌面端本地设置
2. 获取应用信息
3. 调用 `refreshGatewayData()` 刷新网关状态和基础数据

也就是说，桌面端并不是在聊天页首次点击发送时才检测网关，而是在应用启动时就开始检查网关可达性。

---

## 4.2 refreshGatewayData 先探活，再决定是否唤醒

在 `desktop/frontend/src/stores/app.js` 中，`refreshGatewayData()` 的主要逻辑如下：

1. 读取设置里的 `gateway.baseUrl`
2. 如果未配置，则将状态设为 `unconfigured`
3. 如果已配置，则调用 `loadGatewayData(baseUrl)`
4. `loadGatewayData()` 首先请求 `/health`
5. 如果请求成功，则继续拉取 agents 和 conversations
6. 如果请求失败，且错误类型为 `gateway_unreachable`
   - 调用 `ensureBundledGateway(baseUrl)` 尝试唤醒网关
7. 如果唤醒成功，则再次重新加载 Gateway 数据
8. 如果仍失败，则状态设置为 `offline`

因此桌面端“唤醒网关”的直接触发条件是：

- 已配置 `baseUrl`
- 请求 `/health` 失败
- 失败原因被识别为网关不可达

---

## 4.3 前端通过 Wails 桥接调用 Go 层

前端桥接封装位于：

- `desktop/frontend/src/services/wails/config.js`

其中 `ensureBundledGateway(baseUrl)` 会调用：

- `SystemService.EnsureBundledGateway(baseUrl)`

这个桥接方法还有一个特殊处理：

- 如果当前运行环境是浏览器预览模式，而不是 Wails 原生桌面环境
- 那么调用失败时直接返回 `false`

这说明真正的本地进程拉起能力只存在于桌面原生运行环境中。

---

## 5. Go 层如何执行网关唤醒

## 5.1 SystemService 作为唤醒入口

桌面端 Go 层入口位于：

- `desktop/systemservice.go`

`EnsureBundledGateway(baseURL string)` 的逻辑是：

1. 如果 `manager` 不存在则创建 `BundledGatewayManager`
2. 从本地 settings 中加载：
   - `settings.Gateway.ProgramPath`
   - `settings.Gateway.ConfigPath`
3. 调用：
   - `BundledGatewayManager.EnsureBundledGateway(baseURL, programPath, configPath)`

也就是说，前端只传入 `baseUrl`，真正用于决定如何启动网关的 `programPath/configPath` 来自桌面端本地持久化配置。

---

## 5.2 EnsureBundledGateway 的总控逻辑

核心代码位于：

- `desktop/runtime_bootstrap.go`

`BundledGatewayManager.EnsureBundledGateway(baseURL, programPath, configPath)` 是整个桌面端唤醒 Gateway 的总调度入口。

其逻辑可以拆成以下几个阶段。

---

## 6. 唤醒前的判断规则

## 6.1 仅支持本地 Gateway 自动唤醒

方法一开始会先调用 `normalizeGatewayBaseURL(baseURL)`，然后检查目标地址是否属于本地地址：

- `127.0.0.1`
- `localhost`

如果不是本地地址，则直接跳过自动唤醒并返回。

这意味着：

### 关键结论

当前桌面端只会自动唤醒本机 Gateway，不会尝试启动远程 Gateway。

---

## 6.2 优先解析用户配置的程序路径和配置路径

系统会读取：

- `programPath`
- `configPath`

这两个字段来自桌面设置页：

- Gateway Program Path
- Gateway Config Path

它们的作用是：

- `programPath`：指定要启动的网关程序或 bundle 目录
- `configPath`：指定启动时传入的网关配置文件

接着代码会尝试判断：

1. 这两个路径附近是否能识别出一个“bundled package root”
2. 如果不能，应用自身附近是否能自动探测到 bundle 根目录
3. 如果仍不能，但存在 `programPath`，则按“自定义 gateway 程序”启动
4. 如果都没有，则无法自动唤醒

---

## 6.3 健康时不重复启动

在真正拉起进程之前，系统会先检查 `/health`。

如果当前 Gateway 已经健康，则直接跳过启动逻辑。

这是为了避免重复启动本地网关进程。

---

## 7. 桌面端支持的两种网关唤醒模式

## 7.1 模式一：启动 bundled package 中的 Gateway

如果能够识别 bundled package root，则进入 `ensureBundledPackageGateway(cfg)`。

该流程包括：

1. 准备运行目录
2. 停止桌面应用曾管理过的 custom gateway
3. 停止旧的受管进程
4. 生成运行时配置文件
5. 启动 bundled gateway 可执行文件
6. 等待健康检查成功
7. 自动确保默认 Agent 存在
8. 返回成功

### 7.1.1 准备目录

会准备这些目录：

- `bin`
- `config`
- `data`
- `logs`
- `run`

这些目录用于承载运行时二进制、配置、数据库、日志和 pid 文件。

### 7.1.2 自动写入 Gateway 配置

运行时会动态生成网关配置文件，其中包括：

- HTTP 地址
- sqlite 数据库路径
- `session_api_url`
- `internal_token`
- `claw_binary_path`
- `claw_work_dir`
- `claw_config_dir`
- `claw_runner_mode = "fake"`
- claw 端口范围
- 健康检查和关闭超时

这说明 bundled 模式本质上是桌面应用自行组织一套本地可运行的网关运行环境。

### 7.1.3 等待 Gateway 就绪

启动网关后，会轮询请求 `/health`，最长等待 45 秒。

只有健康检查通过后，才认为唤醒完成。

---

## 7.2 模式二：启动用户指定的自定义 Gateway 程序

如果没有 bundled package，但用户提供了 `programPath`，则进入 `ensureCustomGateway(baseURL, programPath, configPath)`。

该流程包括：

1. 校验 `programPath` 是否存在，且必须是可执行文件而不是目录
2. 如果设置了 `configPath`，校验该文件存在，且不是目录
3. 准备 runtime 目录
4. 停止旧的 app 管理进程
5. 组装启动参数
   - 若设置了 `configPath`，则使用 `--config <configPath>`
6. 启动用户指定的 gateway 程序
7. 等待 `/health` 通过
8. 自动确保默认 Agent 存在

因此 custom 模式和 bundled 模式的核心差异主要在于：

- bundled 模式由桌面端自动组织运行时配置和二进制位置
- custom 模式由用户显式提供 gateway 程序和可选配置文件

---

## 8. 唤醒完成后的默认 Agent 保障

无论是 bundled 模式还是 custom 模式，Gateway 拉起成功后都会执行：

- `ensureDefaultAgent(baseURL)`

这段逻辑位于：

- `desktop/runtime_bootstrap.go`

它会向 Gateway 发送 `POST /v1/agents` 请求，尝试创建一个固定默认 Agent：

- `id = agent_desktop_default`
- `name = Desktop Default Agent`
- `model_provider = openai`
- `model_name = fake`
- `max_iterations = 1`
- `tool_whitelist = []`
- `enabled = true`

如果返回：

- `201 Created`，说明创建成功
- `409 Conflict`，说明已存在，也视为成功

如果创建失败，还会再发起一次 `GET /v1/agents/agent_desktop_default` 检查是否已经存在。

### 这一层的意义

桌面端希望在唤醒本地 Gateway 后，系统内至少存在一个能用于桌面聊天的默认 Agent，避免用户刚连接成功却没有可用 Agent。

---

## 9. 用户发送消息时的后续唤醒链路

上面描述的是桌面端如何唤醒 Gateway 进程。接下来才是聊天主链路中，Gateway 如何唤醒真正处理消息的 Agent 实例。

---

## 9.1 前端发送消息流程

聊天流程位于：

- `desktop/frontend/src/stores/chat.js`

`sendPrompt(prompt, conversationId)` 主要逻辑如下：

1. 校验输入内容和当前是否正在 streaming
2. 从设置中读取：
   - `gateway.baseUrl`
   - `gateway.defaultAgentId`
3. 如果没有默认 Agent，直接报错
4. 如果没有 conversation，则先调用创建会话接口
5. 本地先追加用户消息草稿和 assistant 草稿
6. 建立 WebSocket 连接
7. 通过 WebSocket 发送 `chat.start`

发送的 WebSocket 消息包括：

- `conversation_id`
- `prompt`
- `request_id`
- `metadata`

---

## 9.2 Gateway WebSocket 控制器接收聊天请求

入口位于：

- `server/gateway/internal/controller/chat_ws_controller.go`

`chat.start` 请求到达后：

1. 校验 `conversation_id` 和 `prompt`
2. 检查当前 socket 上是否已有进行中的请求
3. 创建可取消上下文
4. 调用 `chat.StreamMessage(...)`
5. 先向前端回写 `session.accepted`
6. 再异步持续转发流式事件

这里 Gateway 并不会自己直接运行模型，而是转交给下层 `ChatService` 决定该请求应该由哪个 Agent Instance 处理。

---

## 10. Gateway 如何选择或唤醒执行实例

## 10.1 ChatService 在真正执行前先 prepareRun

在：

- `server/gateway/internal/service/chat_service.go`

`StreamMessage()` 内首先调用 `prepareRun(ctx, conversationID)`。

`prepareRun()` 做三件事：

1. 读取 conversation
2. 读取 conversation 对应的 agent 配置
3. 调用 `router.SelectInstance(ctx, conversation)` 选择一个实例

因此，实例唤醒逻辑真正发生在 `SelectInstance` 阶段。

---

## 10.2 RouterPolicy 优先找 ready 实例，没有就启动新实例

核心逻辑位于：

- `server/gateway/internal/service/router_policy.go`

`SelectInstance()` 的策略是：

1. 先刷新实例健康状态
2. 读取当前所有实例
3. 过滤出：
   - `AgentID` 与当前 conversation 一致
   - `Status == "ready"`
4. 如果当前会话有 `StickyInstanceID`，优先复用该实例
5. 如果没有 sticky 命中，则选 `Inflight` 最少的 ready 实例
6. 如果一个可用 ready 实例都没有，则调用：
   - `startInstance(ctx, conversation.AgentID)`

### 关键结论

真正“唤醒对应网关执行实例”的动作不是桌面端做的，而是 Gateway 在会话路由阶段自动做的。

---

## 10.3 AgentInstanceService.Start 负责拉起 Agent Instance

对应代码位于：

- `server/gateway/internal/service/agent_instance_service.go`

`Start(ctx, req)` 的流程如下：

1. 校验目标 `AgentID` 是否存在
2. 查询当前已有实例列表
3. 检查是否达到 `MaxAgentInstances`
4. 在可用端口范围内分配端口
5. 生成 instance id
6. 基于配置构造进程启动描述 `processSpec`
7. 调用 `supervisor.Start(ctx, spec)` 启动新进程
8. 初始状态标记为 `starting`
9. 轮询探测实例健康状态
10. 健康后更新为 `ready`
11. 持久化保存实例信息

这一步本质上是 Gateway 在拉起一个真正处理请求的 claw agent 服务实例。

---

## 11. 完整时序梳理

## 11.1 阶段一：桌面端唤醒 Gateway

1. 桌面端应用启动
2. 前端执行 `bootstrap()`
3. 加载本地 settings
4. 执行 `refreshGatewayData()`
5. 请求 `GET /health`
6. 如果网关不可达，则调用 `SystemService.EnsureBundledGateway(baseUrl)`
7. Go 层判断是否是本地地址
8. 判断应使用 bundled 模式还是 custom 模式
9. 启动 Gateway 进程
10. 轮询 `/health` 等待就绪
11. 调用 `ensureDefaultAgent()` 确保默认 Agent 存在
12. 前端再次拉取网关健康、Agent 列表和会话列表
13. 网关状态变为 `connected`

---

## 11.2 阶段二：Gateway 唤醒 Agent Instance

1. 用户在桌面端输入问题
2. 前端若无会话则先创建 conversation
3. 前端建立 WebSocket 连接并发送 `chat.start`
4. Gateway WebSocket 控制器调用 `ChatService.StreamMessage`
5. `ChatService.prepareRun()` 加载 conversation 和 agent
6. `RouterPolicy.SelectInstance()` 尝试选择 ready 实例
7. 如果没有 ready 实例，则调用 `AgentInstanceService.Start()`
8. Gateway 拉起新的 claw 实例
9. 实例健康探测通过后状态变为 `ready`
10. Gateway 将请求转发给该实例执行
11. 流式输出通过 WebSocket 回传桌面端
12. 前端收到 `message.delta` 与 `message.completed`

---

## 12. 配置项对唤醒流程的影响

桌面端网关配置定义位于：

- `desktop/internal/config/config.go`

关键字段如下：

### 12.1 `baseUrl`

决定桌面端请求 Gateway 的地址，同时决定是否允许自动唤醒：

- `127.0.0.1` / `localhost`：支持自动唤醒
- 远程地址：不执行本地自动唤醒

### 12.2 `programPath`

指定网关程序或 bundle 目录。若配置后可被解析为 bundle 根目录，则优先按 bundled 模式处理；否则按 custom gateway 程序启动。

### 12.3 `configPath`

在 custom 启动模式下作为 `--config` 参数传入；在部分 bundle 识别流程中也可辅助定位 package root。

### 12.4 `defaultAgentId`

前端发起聊天时默认使用的 Agent ID。

需要注意的是：

- 前端聊天依赖 `settings.gateway.defaultAgentId`
- Go 层自动保障的是固定 ID：`agent_desktop_default`

这意味着如果用户把 `defaultAgentId` 改成其他值，而该 Agent 在 Gateway 中不存在，则桌面端虽然已经成功唤醒 Gateway，但聊天时仍然可能因为找不到指定 Agent 而失败。

---

## 13. 当前实现的设计特点

## 13.1 优点

### 13.1.1 桌面端具备自动恢复能力

对于本地 Gateway 场景，桌面端在检测到网关不可达后可以自动尝试拉起进程，提升了启动成功率和可用性。

### 13.1.2 支持 bundled 与 custom 两种启动模式

同一套桌面端逻辑兼容：

- 应用自带的 bundle 运行环境
- 用户指定的独立 gateway 程序

扩展性较好。

### 13.1.3 进程层和实例层职责分离

- 桌面端负责 Gateway 进程级唤醒
- Gateway 负责 Agent Instance 级唤醒

这种分层让系统职责边界比较清晰。

---

## 13.2 注意点

### 13.2.1 自动唤醒只对本地地址生效

如果未来存在远程部署或局域网 Gateway 场景，当前桌面端不会主动帮助用户完成远程侧进程拉起。

### 13.2.2 默认 Agent 保障与前端默认 Agent 配置不是完全同一个概念

Go 层固定保障 `agent_desktop_default`，但前端真正发消息时读取的是 `settings.gateway.defaultAgentId`。如果两者不一致，可能出现“网关已启动但聊天仍失败”的情况。

### 13.2.3 唤醒发生在网关数据刷新阶段，不是每次发消息都兜底

当前自动唤醒主要挂在应用初始化后的 `refreshGatewayData()`。如果应用运行中 Gateway 后续异常退出，聊天链路未必总能自动重新拉起。

### 13.2.4 bundled 模式当前写入的是 fake runner 配置

在 bundled 模式动态生成的配置中，`claw_runner_mode` 当前写为 `fake`。这说明这套链路当前更偏向本地演示、测试或桌面 MVP 自举能力，而不一定等同正式生产运行模式。

---

## 14. 总结

当前项目中，桌面端对“网关唤醒”的实现可以概括为两段式：

### 第一段：桌面端唤醒 Gateway

桌面前端启动后先做健康检查；如果发现本地 Gateway 不可达，就通过 Wails Go 服务调用 `EnsureBundledGateway`，按 bundled 或 custom 模式启动网关，并在启动完成后自动确保默认 Agent 存在。

### 第二段：Gateway 唤醒对应 Agent Instance

当用户真正发送聊天请求时，Gateway 会在 `SelectInstance` 阶段查找可用实例；如果没有 ready 实例，则通过 `AgentInstanceService.Start` 启动新的 Agent Instance，再继续执行流式对话。

因此从系统行为上看，桌面端负责“把 Gateway 拉起来”，而 Gateway 负责“把真正执行请求的 Agent 实例拉起来”。
