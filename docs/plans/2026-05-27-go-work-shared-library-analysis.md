# Go Work 模式与公共库拆分分析报告

日期：2026-05-27

## 结论摘要

项目已经处于 Go workspace 的雏形状态，根目录 `go.work` 当前纳入了 `desktop`、`server/gateway`、`server/claw`、`server/shared/errors` 四个模块。但目前的模块边界仍然偏“开发便利型”，还没有完全变成清晰的架构边界。

最值得调整的方向不是“再增加更多小模块”，而是把现有 `server/shared/errors` 升级为一个统一的 `server/shared` 公共模块，再把跨 Gateway/Claw 复用的协议类型、Session API 契约、轻量基础设施工具逐步迁入其中。这样可以减少 Gateway 对 Claw 运行时包的直接依赖，也能让公共能力有稳定归属。

推荐目标：

```text
icoo_claw
├─ go.work
├─ desktop                  # 桌面壳与 Wails 服务
├─ server
│  ├─ gateway               # 控制面、HTTP API、DB、进程管理
│  ├─ claw                  # Agent 执行服务与运行时
│  └─ shared                # 公共契约与轻量基础设施
│     ├─ agentproto         # Gateway <-> Claw 运行协议类型
│     ├─ sessionapi         # Session API DTO + HTTP client
│     ├─ errors             # 错误包装工具
│     ├─ configx            # TOML / env / path 读取辅助
│     └─ jsonx              # JSON 字段序列化辅助，可选
└─ go_pkg/redka             # 外部 checkout，不建议纳入主 workspace
```

## 当前状态

### 已存在的 workspace

根目录 `go.work` 当前内容：

```go
go 1.25.0

use (
    ./desktop
    ./server/claw
    ./server/gateway
    ./server/shared/errors
)
```

这说明项目已经开始按多模块组织。`go list -m` 在任一 workspace 模块下都会识别出这四个主模块：

```text
icoo_claw/desktop
icoo_claw/server/claw
icoo_claw/server/gateway
icoo_claw/server/shared/errors
```

### 当前模块职责

| 模块 | 当前职责 | 主要问题 |
|---|---|---|
| `desktop` | Wails 桌面服务与前端绑定 | 与服务端隔离较好，暂不需要重拆 |
| `server/gateway` | 外部 API、控制面、会话存储、Agent 实例管理 | 直接依赖 Claw 的 `agent_sdk` 类型，model/dto 重复偏多 |
| `server/claw` | Agent 执行、模型调用、工具运行、Session API client | 同时承载运行时实现和对外协议类型，边界偏厚 |
| `server/shared/errors` | 错误包装工具 | 模块过细，只有 1 个 Go 文件，公共库形态尚未成型 |

### 独立模块解析问题

在关闭 workspace 后验证 Gateway：

```powershell
$env:GOWORK='off'
go list -deps .\server\gateway\cmd\gateway
```

结果会出现类似错误：

```text
package icoo_claw/server/claw/pkg/agent_sdk is not in std
package icoo_claw/server/shared/errors is not in std
```

这说明当前 `go.work` 能让开发态构建通过，但 Gateway 模块本身并不是一个干净的独立模块。原因有两个：

1. Gateway 直接 import `icoo_claw/server/claw/pkg/agent_sdk`，形成了 Gateway -> Claw 的服务实现依赖。
2. Gateway import `icoo_claw/server/shared/errors`，但该依赖没有在 Gateway `go.mod` 中以 `require/replace` 固化。

如果项目决定正式采用 go work 模式，这不是立即阻塞；但如果未来需要单模块构建、CI 分模块测试、外部复用，就需要整理。

## 公共库候选评估

### 1. Agent 运行协议类型

现状：

- 共享类型定义在 `server/claw/pkg/agent_sdk/types.go`
- Gateway 通过 `server/gateway/internal/client/claw_client.go` 引用这些类型别名
- Claw 内部 DTO 也通过别名复用这些类型

问题：

Gateway 不应该依赖 Claw 的运行时 SDK 包。`agent_sdk` 不只是协议类型，它还包含模型、工具、沙箱、hooks、skills、subagents 等执行侧实现。即便 Gateway 当前只引用 `types.go`，依赖方向也在语义上不干净。

建议：

将以下类型迁入 `server/shared/agentproto`：

- `RunRequest`
- `RunResponse`
- `StreamEvent`
- `SessionUpdate`
- `ContentBlock`
- `ToolCallLocation`
- `UsageUpdate`
- `StreamError`
- Stream event 常量

迁移后：

```text
gateway/internal/client -> shared/agentproto
claw/pkg/agent_sdk     -> shared/agentproto
claw/internal/dto      -> shared/agentproto 或 agent_sdk 别名
```

这样 Gateway 不再依赖 Claw 执行服务，只依赖共享协议。

优先级：高  
风险：中  
收益：高

### 2. Session API 契约与客户端

现状存在三套高度相似结构：

- `server/claw/pkg/sessionstore/types.go`
- `server/gateway/internal/dto/session.go`
- `server/gateway/internal/sessionstore/model/session.go`

Claw 的 `pkg/sessionstore` 同时包含 Session API DTO 和 HTTP client。Gateway 也维护一套 DTO，再在 controller 中转换到 sessionstore model。

建议：

将 API 契约和 HTTP client 迁入 `server/shared/sessionapi`：

```text
server/shared/sessionapi
├─ types.go     # Session / Message / Run / RunEvent / request / response
└─ client.go    # HTTP client
```

迁移后：

- Claw 使用 `shared/sessionapi.Client`
- Gateway controller 使用 `shared/sessionapi` 作为 API DTO
- Gateway 的 DB record 继续留在 `gateway/internal/sessionstore/model`
- Gateway 的领域 model 可逐步简化为共享类型别名，或保留少量 repository 专用类型

注意：

`binding:"required"` 这种 Gin HTTP 入参校验 tag 不建议放到共享契约的核心类型上。可通过 controller 显式校验，或只在 request wrapper 上保留。

优先级：高  
风险：中偏高  
收益：高

### 3. Config 读取辅助

现状：

- `server/gateway/internal/config/config.go`
- `server/claw/internal/config/config.go`
- `desktop/internal/config/store.go`

三处都使用 TOML，且服务端都有 `--config` 参数、默认值、文件读取、TOML 解析、路径处理等相似逻辑。

建议抽取到 `server/shared/configx`：

- `ConfigPath(args []string, fallback string) string`
- `LoadTOML(path string, target any) (found bool, err error)`
- `ResolveDataPath(value, baseDir string) string`
- `ResolveExecutablePath(value, baseDir string) string`
- `EnvInt(key string) int`

不建议把 Gateway/Claw 的完整 `Config` struct 合并。配置字段是服务私有契约，公共库只提供读取和路径辅助即可。

优先级：中  
风险：低  
收益：中

### 4. 错误工具

现状：

- 已有 `server/shared/errors/errors.go`
- 但全项目仍有大量 `fmt.Errorf`
- 当前 shared errors 只在少数位置使用

建议：

把 `server/shared/errors` 保留为 `server/shared` 模块下的一个包，而不是独立模块。是否大规模替换 `fmt.Errorf` 不必强制；优先在跨层边界、repository、config loader 里使用即可。

推荐包名可保持：

```go
import sharedErrors "icoo_claw/server/shared/errors"
```

优先级：中  
风险：低  
收益：中

### 5. JSON 字段序列化辅助

现状：

Gateway 中多处手写 `json.Marshal` / `json.Unmarshal` 到字符串字段，例如 Agent 的 `CommandArgsJSON`、Skill 的 `AllowedToolsJSON`、Session record 的 `MetadataJSON`。

建议：

短期优先使用 GORM `serializer:json` 消除部分 `XXXJSON` 字段；如果仍需手写字符串字段，再抽 `server/shared/jsonx`：

- `MarshalString[T any](value T, fallback T) (string, error)`
- `MustMarshalStringSlice(values []string) string`
- `UnmarshalString[T any](raw string, fallback T) T`

注意：

这个包容易变成“杂物间”。只有当同样逻辑在两个以上模块使用时再抽，不要只为了 Gateway 内部复用而放进 shared。

优先级：中  
风险：低  
收益：中

### 6. Repository 基类

现状：

Gateway 已有 `internal/repository/base.go`，泛型 CRUD 基类已经落地。

建议：

暂时不要抽到 `server/shared`。它依赖 GORM，且目前只有 Gateway 使用。公共库应该先服务跨模块复用，不宜把 Gateway 的持久化实现提前公共化。

优先级：低  
风险：中  
收益：低

### 7. Router / DI

现状：

Gateway 和 Claw 都有 Gin router、DI container，但职责不同。Gateway 有 DB、进程管理、WebSocket、session store；Claw 有 runtime factory、Agent controller、ACP agent。

建议：

暂不抽象 DI 容器。可抽少量 HTTP 辅助，比如 CORS middleware 或统一 error response，但不要为了形式一致建立复杂共享框架。

优先级：低  
风险：中  
收益：低

## 推荐模块方案

### 推荐方案：单一 `server/shared` 模块

将现在的 `server/shared/errors` 模块升级为：

```text
server/shared
├─ go.mod                      # module icoo_claw/server/shared
├─ errors
│  └─ errors.go
├─ agentproto
│  └─ types.go
├─ sessionapi
│  ├─ types.go
│  └─ client.go
├─ configx
│  └─ configx.go
└─ jsonx
   └─ jsonx.go                 # 可选
```

根目录 `go.work` 调整为：

```go
go 1.25.0

use (
    ./desktop
    ./server/claw
    ./server/gateway
    ./server/shared
)
```

优点：

- 模块数量少，认知成本低
- 公共包有统一归属
- 后续新增公共能力不需要频繁修改 `go.work`
- import path 稳定，例如 `icoo_claw/server/shared/errors`

不推荐继续拆成多个极小模块：

```text
server/shared/errors
server/shared/agentproto
server/shared/sessionapi
server/shared/configx
```

这种方式会让 `go.work` 和各模块 `go.mod` 持续膨胀，也会制造大量本地 `replace`。

## 目标依赖方向

推荐依赖关系：

```text
desktop
  └─ 不直接依赖 gateway/claw，必要时只依赖 shared 里的轻量契约

gateway
  ├─ shared/agentproto
  ├─ shared/sessionapi
  ├─ shared/errors
  └─ shared/configx

claw
  ├─ shared/agentproto
  ├─ shared/sessionapi
  ├─ shared/errors
  └─ shared/configx

shared
  └─ 不依赖 desktop/gateway/claw
```

明确禁止的方向：

```text
gateway -> claw/pkg/agent_sdk
shared  -> gateway/internal/...
shared  -> claw/internal/...
desktop -> gateway/internal/...
desktop -> claw/internal/...
```

`shared` 应保持底层、稳定、无业务运行时依赖。它可以依赖标准库和少量低风险基础库，例如 `github.com/pelletier/go-toml/v2`；不建议依赖 Gin、GORM、Wails、模型 SDK。

## 迁移路线

### Phase 0：建立边界规则

目标：

- 明确 `go.work` 是本仓库本地开发的一等入口
- 明确 `go_pkg/redka` 暂不纳入 workspace，除非转为正式模块依赖
- 建立跨模块 import 禁止规则

建议动作：

1. 保留根目录 `go.work`
2. 在 README 增加 workspace 开发说明
3. 在 CI 或本地脚本中加入：

```powershell
go work sync
go test ./server/gateway/...
go test ./server/claw/...
Push-Location .\desktop
go test ./...
Pop-Location
```

验收：

- workspace 模式下全部测试通过
- `rg "icoo_claw/server/claw/pkg/agent_sdk" server/gateway` 后续应逐步归零

### Phase 1：把 `server/shared/errors` 升级为 `server/shared`

目标：

- 将 shared 从单包模块变成公共库模块
- 减少未来新增公共包时的 go.work churn

建议动作：

1. 新建 `server/shared/go.mod`

```go
module icoo_claw/server/shared

go 1.25.0
```

2. 保留 `server/shared/errors/errors.go`
3. 删除 `server/shared/errors/go.mod`
4. 修改 `go.work`：

```go
use (
    ./desktop
    ./server/claw
    ./server/gateway
    ./server/shared
)
```

5. Gateway `go.mod` 添加：

```go
require icoo_claw/server/shared v0.0.0

replace icoo_claw/server/shared => ../shared
```

说明：

workspace 模式下 replace 不是必须，但加上后 `GOWORK=off` 的单模块验证也更稳。

验收：

```powershell
go test ./server/gateway/...
$env:GOWORK='off'
Push-Location .\server\gateway
go list ./...
Pop-Location
```

### Phase 2：抽取 `agentproto`

目标：

- 移除 Gateway 对 Claw `agent_sdk` 的直接依赖
- 让运行协议成为独立契约

建议动作：

1. 创建 `server/shared/agentproto/types.go`
2. 从 `server/claw/pkg/agent_sdk/types.go` 移动协议结构体和事件常量
3. `server/claw/pkg/agent_sdk/types.go` 改为类型别名：

```go
type RunRequest = agentproto.RunRequest
type RunResponse = agentproto.RunResponse
type StreamEvent = agentproto.StreamEvent
```

4. Gateway client 改为 import `icoo_claw/server/shared/agentproto`
5. Claw internal dto 可继续别名到 `agent_sdk`，或直接别名到 `agentproto`

验收：

```powershell
rg "icoo_claw/server/claw/pkg/agent_sdk" server/gateway
go test ./server/gateway/...
go test ./server/claw/...
```

预期：

`server/gateway` 不再 import `server/claw`。

### Phase 3：抽取 `sessionapi`

目标：

- 合并 Session API 契约
- 让 Claw 的 Session HTTP client 不再放在 Claw 模块里

建议动作：

1. 创建 `server/shared/sessionapi/types.go`
2. 将 `server/claw/pkg/sessionstore/types.go` 的 API 类型迁入
3. 创建 `server/shared/sessionapi/client.go`
4. Claw 的 `pkg/sessionstore` 可短期保留兼容别名，长期删除
5. Gateway `internal/dto/session.go` 可改为类型别名，或 controller 直接使用 `sessionapi`

验收：

```powershell
rg "pkg/sessionstore" server/claw server/gateway
go test ./server/claw/...
go test ./server/gateway/...
```

### Phase 4：抽取 `configx`

目标：

- 减少 Gateway / Claw 配置读取重复
- 保留服务私有 Config struct

建议动作：

1. 创建 `server/shared/configx`
2. 抽公共函数：

```go
func ConfigPath(args []string, fallback string) string
func ReadTOMLFile(path string, target any) (found bool, err error)
func ResolveDataPath(value string, baseDir string) string
func ResolveExecutablePath(value string, baseDir string) string
func EnvInt(key string) int
```

3. Gateway 和 Claw 分别保留自己的 defaults/applyFile/applyEnv

验收：

```powershell
go test ./server/gateway/internal/config
go test ./server/claw/internal/config
```

### Phase 5：清理 DTO / Model 重复

目标：

- 在公共契约稳定后，继续处理 Gateway 内部 model/dto 重复

建议动作：

1. Session DTO 优先别名到 `shared/sessionapi`
2. Agent/Skill/ScheduledTask 的 JSON slice 字段考虑用 GORM `serializer:json`
3. 仅在 API 形状和 DB 形状确实不同的地方保留转换函数

验收：

```powershell
rg "func to.*DTO|func .*ToRecord|func recordTo" server/gateway/internal
go test ./server/gateway/...
```

## 风险与控制

| 风险 | 说明 | 控制方式 |
|---|---|---|
| 公共库变成杂物包 | 什么都往 shared 放，后续反而更乱 | shared 只放跨两个以上模块复用且稳定的能力 |
| 模块路径频繁变更 | 多模块 import path 改动会影响大量文件 | 先统一 `server/shared`，避免多个小模块 |
| API tag 与内部模型 tag 冲突 | DTO 需要 JSON/binding，DB record 需要 GORM | API 契约和 DB record 分离，binding 由 controller 处理 |
| Gateway 单模块构建失败 | 当前依赖依靠 go.work 解析 | 抽出 shared 后，给 gateway/claw go.mod 增加 require + replace |
| 迁移影响范围过大 | 一次性移动 agent/session/config 容易产生大改动 | 按 Phase 拆分，每阶段保持类型别名兼容 |

## 推荐优先级

| 优先级 | 工作项 | 原因 |
|---|---|---|
| P0 | 明确 `go.work` 为本地开发入口，排除 `go_pkg/redka` | 防止 workspace 边界继续变模糊 |
| P1 | `server/shared/errors` 升级为 `server/shared` | 后续公共包需要统一承载点 |
| P1 | 抽 `shared/agentproto` | 解除 Gateway -> Claw 运行时依赖 |
| P2 | 抽 `shared/sessionapi` | 消除 Claw/Gateway Session API 类型重复 |
| P2 | 抽 `shared/configx` | 降低配置读取重复，风险较低 |
| P3 | Gateway model/dto 深度收敛 | 影响面大，等契约稳定后做 |
| P3 | 错误包装批量推广 | 有收益但不应阻塞架构拆分 |

## 验收清单

最终状态建议满足：

- `go.work` 只引用一层稳定模块，不引用过细子包模块
- `server/shared` 是单一公共模块
- `server/gateway` 不再 import `icoo_claw/server/claw/...`
- `server/claw` 和 `server/gateway` 都只通过 `shared/agentproto` 共享运行协议
- Session API 类型只在 `shared/sessionapi` 定义一次
- `GOWORK=off` 下 Gateway 和 Claw 至少能 `go list ./...` 通过
- `go test ./server/gateway/...`、`go test ./server/claw/...`、`go test ./desktop/...` 均通过

## 最终建议

本项目适合正式采用 Go work 模式，但需要把它从“能让多模块本地一起编译”的工具，提升为“表达架构边界”的工具。

最优先的结构调整是：

1. 将 `server/shared/errors` 改成统一的 `server/shared` 模块。
2. 把 Gateway 与 Claw 共享的运行协议从 Claw `agent_sdk` 中抽到 `shared/agentproto`。
3. 把 Session API DTO 和 HTTP client 从 Claw `pkg/sessionstore` 中抽到 `shared/sessionapi`。
4. 保持 Gateway/Claw 的业务实现、DB 模型、DI、Router 在各自模块内，不急于公共化。

这样调整后，workspace 的依赖方向会从当前的：

```text
gateway -> claw/pkg/agent_sdk
gateway -> shared/errors
claw    -> own sessionstore client
```

变成：

```text
gateway -> shared/agentproto + shared/sessionapi + shared/errors
claw    -> shared/agentproto + shared/sessionapi + shared/errors
shared  -> no project modules
```

这个方向更适合后续继续拆分服务、独立测试、打包发布和减少代码冗余。
