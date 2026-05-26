# 代码冗余消除实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 消除项目中三处重复定义的核心类型（DTO/StreamEvent 等）、Gateway 内部 model/dto 双向重复、以及模板化的 CRUD 和基础设施代码。

**Architecture:** 以 `server/claw/pkg/agent_sdk/types.go` 作为共享类型的唯一真理来源，`claw/internal/dto` 和 `gateway/internal/client` 均从它导入，消除 15+ 个跨模块重复结构体。Gateway 内部用 DTO 携带 Model 嵌入（复合而非重复）来消除 model↔dto 手动转换。GCursor CRUD DB 用泛型减少 28 个重复方法。

**Tech Stack:** Go 1.25 (已支持泛型), GORM, Wails3

---

## 冗余现状回顾

| 冗余类型 | 严重度 | 影响范围 |
|---------|--------|---------|
| `RunRequest/RunResponse/StreamEvent/SessionUpdate/ContentBlock/ToolCallLocation/UsageUpdate/StreamError` 在 `claw/dto` + `agent_sdk/types` + `gateway/client` 三处重复 | **高** | 15+ 个结构体 |
| Gateway `model/` (GORM) 与 `dto/` (JSON) 7 组类型完全对应 | **高** | 14 个文件, 7 组转换 |
| 7 个 Repository 中 28 次 CRUD 方法模板重复 | **中** | gateway/repository/ 全量 |
| `claw/DI` 与 `gateway/DI` 容器模板重复 | **低** | 2 个 container.go |
| `Config` 结构体在 3 个模块中各自定义 | **低** | 3 个 config.go |
| 408 处 `fmt.Errorf("...: %w", err)` 机械模板 | **低** | 全项目 |

---

## Phase 1: 共享类型统一 — RunRequest/RunResponse/StreamEvent 族

**目标：** 将 `RunRequest`、`RunResponse`、`StreamEvent`、`SessionUpdate`、`ContentBlock`、`ToolCallLocation`、`UsageUpdate`、`StreamError` 统一到 `agent_sdk/types.go`，claw/dto 和 gateway/client 仅做类型别名或直接导入。

### Task 1.1: 审计三处类型定义差异

**Files:**
- Read: `server/claw/pkg/agent_sdk/types.go`
- Read: `server/claw/internal/dto/agent.go`
- Read: `server/gateway/internal/client/claw_client.go:13-?` (RunRequest/RunResponse/StreamEvent 定义处)
- Read: `server/gateway/internal/dto/chat_ws.go` (SessionUpdate/ContentBlock 定义处)

**Step 1: 逐字段对比三处定义，记录差异清单**

差异对比（已知）：
- `claw/dto/agent.go` 的 `RunRequest` 有 `binding:"required"` tag，`agent_sdk/types.go` 没有——后者是纯 SDK 类型，不应有 binding tag
- `claw/dto/agent.go` 的 `ContentBlock.Data` 是 `any`，`agent_sdk/types.go` 是 `json.RawMessage`
- `claw/dto/agent.go` 的 `StreamEvent` 有完整 json tag，`agent_sdk/types.go` 部分字段无 json tag
- `gateway/client/claw_client.go` 的类型字段完全相同但以 HTTP client 视角组织

**Step 2: 确认 `agent_sdk/types.go` 是各字段的超集**

需确认：
```bash
rg "^type (RunRequest|RunResponse|StreamEvent|SessionUpdate|ContentBlock|ToolCallLocation|UsageUpdate|StreamError) struct" --type go
```

**Step 3: 记录结论：差异处理方案**

- `binding` tag → 保留在 claw/dto 包装层（HTTP 入参校验是 controller 职责，不属于 SDK）
- `Data` 字段类型 → 统一为 `json.RawMessage`（更安全）
- json tag 缺失 → 补充到 `agent_sdk/types.go`
- 所有 json tag 命名使用 snake_case 风格（与 `agent_sdk/types.go` 一致）

**Step 4: Commit**

```bash
git add docs/plans/2026-05-26-code-redundancy-elimination.md
git commit -m "docs: add code redundancy elimination plan"
```

---

### Task 1.2: 补充 agent_sdk/types.go 为完整共享类型

**Files:**
- Modify: `server/claw/pkg/agent_sdk/types.go`

**Step 1: 统一 `Runner` 接口和 `RunRequest`**

确保 `agent_sdk/types.go` 内的 `RunRequest` 字段完整：
```go
type RunRequest struct {
    SessionID     string         `json:"session_id"`
    RequestID     string         `json:"request_id"`
    Prompt        string         `json:"prompt"`
    Agent         map[string]any `json:"agent,omitempty"`
    ToolWhitelist []string       `json:"tool_whitelist,omitempty"`
    ForceSkills   []string       `json:"force_skills,omitempty"`
    Metadata      map[string]any `json:"metadata,omitempty"`
}
```

**Step 2: 统一所有 JSON event 类型**

`RunResponse`、`StreamEvent`、`SessionUpdate`、`ContentBlock`、`ToolCallLocation`、`UsageUpdate`、`StreamError` 全部确保字段完整且 json tag 一致。

需关注 `ContentBlock.Data` 类型：`json.RawMessage`（agent_sdk/types.go 现状）vs `any`（claw/dto 现状）。选择 `json.RawMessage` 更安全。

**Step 3: 添加 `StreamEvent` 相关常量和辅助方法（如有必要）**

保持现有：
```go
const (
    StreamEventSessionUpdate    = "session/update"
    StreamEventSessionCompleted = "session/completed"
    StreamEventSessionError     = "session/error"
)
```

**Step 4: 运行已有测试确保兼容**

```bash
go test ./pkg/agent_sdk/... -v
```
工作目录: `server/claw`

**Step 5: Commit**

```bash
git add server/claw/pkg/agent_sdk/types.go
git commit -m "refactor(agent_sdk): complete shared types with full json tags"
```

---

### Task 1.3: 重构 claw/internal/dto 为 agent_sdk 类型的包装层

**Files:**
- Modify: `server/claw/internal/dto/agent.go`
- Check: `server/claw/internal/controller/agent_controller.go` (依赖 dto 的地方)
- Check: `server/claw/internal/service/agent_service.go` (依赖 dto 的地方)

**Step 1: 将 dto/agent.go 改为类型别名或嵌入**

选项 A（类型别名，最简单）：
```go
package dto

import "github.com/claw/pkg/agent_sdk"

type RunRequest = agent_sdk.RunRequest
type RunResponse = agent_sdk.RunResponse
type StreamEvent = agent_sdk.StreamEvent
type SessionUpdate = agent_sdk.SessionUpdate
type ContentBlock = agent_sdk.ContentBlock
type ToolCallLocation = agent_sdk.ToolCallLocation
type UsageUpdate = agent_sdk.UsageUpdate
type StreamError = agent_sdk.StreamError
```

选项 B（嵌入包装，保留 binding tag）：
```go
package dto

import "github.com/claw/pkg/agent_sdk"

type RunRequest struct {
    agent_sdk.RunRequest
}

type StreamEvent struct {
    agent_sdk.StreamEvent
}
```

**推荐选项 A**，因为 controller 层需要 binding tag 的地方可单独处理（见 Task 1.4）。

**Step 2: 全量搜索并更新所有 `dto.RunRequest` 引用**

```bash
rg "dto\.(RunRequest|RunResponse|StreamEvent|SessionUpdate|ContentBlock|ToolCallLocation|UsageUpdate|StreamError)" --type go
```
工作目录: `server/claw`

确保所有引用不受影响（类型别名是透明的）。

**Step 3: 运行 claw 模块测试**

```bash
go test ./... -v
```
工作目录: `server/claw`

**Step 4: Commit**

```bash
git add server/claw/internal/dto/agent.go
git commit -m "refactor(claw/dto): use type aliases to agent_sdk shared types"
```

---

### Task 1.4: 重构 gateway/internal/client 为 agent_sdk 类型的消费者

**Files:**
- Modify: `server/gateway/internal/client/claw_client.go`
- Modify: `server/gateway/internal/client/stream_event_adapter.go`
- Modify: `server/gateway/internal/client/stream_events.go`
- Modify: `server/gateway/internal/dto/chat_ws.go`

**Step 1: 移除 gateway/internal/client 中的重复类型定义**

当前 `claw_client.go` 文件内定义了 `RunRequest`、`RunResponse`、`StreamEvent`、`StreamError` 等。

由于 `gateway/go.mod` 并不 require `server/claw/pkg/agent_sdk`，需要评估：

方案 A：添加 gateway 对 claw 模块的依赖（go.work 已声明，可直接 import）
方案 B：将共享类型提取到独立共享包 `server/shared/`

**推荐方案 A**：`go.work` 已包含两个模块，gateway 可以直接 `import "github.com/claw/pkg/agent_sdk"`。

**Step 2: 更新 gateway/internal/client/claw_client.go**

```go
import (
    "github.com/claw/pkg/agent_sdk"
)

type RunRequest = agent_sdk.RunRequest
type RunResponse = agent_sdk.RunResponse
type StreamEvent = agent_sdk.StreamEvent
```

删除原有的类型定义代码块。

**Step 3: 更新 gateway/internal/client/stream_event_adapter.go**

搜索并替换类型引用为 agent_sdk 版本。

**Step 4: 更新 gateway/internal/dto/chat_ws.go**

同样将 `SessionUpdate`、`ContentBlock`、`ToolCallLocation`、`UsageUpdate` 的重复定义替换为别名。

**Step 5: 确保编译通过**

```bash
go build ./...
```
工作目录: `server/gateway`

**Step 6: 运行 gateway 测试**

```bash
go test ./... -v
```
工作目录: `server/gateway`

**Step 7: Commit**

```bash
git add server/gateway/internal/client/claw_client.go server/gateway/internal/client/stream_event_adapter.go server/gateway/internal/client/stream_events.go server/gateway/internal/dto/chat_ws.go
git commit -m "refactor(gateway): use agent_sdk shared types, remove duplicate definitions"
```

---

## Phase 2: Gateway 内部 model↔dto 合并

**目标：** 消除 `gateway/model/` 和 `gateway/dto/` 中 7 组完全对应的类型重复。

### Task 2.1: 审计 model 与 dto 的差异

**Files:**
- Read: `server/gateway/internal/model/agent.go`
- Read: `server/gateway/internal/dto/agent.go`
- Read: `server/gateway/internal/model/agent_instance.go`
- Read: `server/gateway/internal/dto/agent_instance.go`
- Read: `server/gateway/internal/model/conversation.go`
- Read: `server/gateway/internal/dto/conversation.go`
- Read: `server/gateway/internal/model/scheduled_task.go`
- Read: `server/gateway/internal/dto/scheduled_task.go`
- Read: `server/gateway/internal/model/scheduled_task_run.go`
- Read: `server/gateway/internal/dto/scheduled_task_run.go`
- Read: `server/gateway/internal/model/provider.go`
- Read: `server/gateway/internal/dto/provider.go`

**Step 1: 记录差异模式**

核心差异是 tags：
- `model/` 使用 GORM tag（`gorm:"primaryKey"` 等）
- `dto/` 使用 JSON tag（`json:"id"` 等）

特殊差异（以 AgentProfile 为例）：
- `model/AgentProfile` 的切片字段（CommandArgs、ToolWhitelist 等）用 `*JSON` 后缀存储为 JSON 字符串
- `dto/AgentProfile` 的对应字段直接是 `[]string` 类型

**Step 2: 确认转换函数的位置**

搜索 model↔dto 的转换函数：
```bash
rg "func.*(ToDTO|ToModel|FromModel)" --type go
```
工作目录: `server/gateway`

**Step 3: Commit**

无需 commit（审计阶段，只读操作）。
```

---

### Task 2.2: 采用嵌入模式消除 model↔dto 重复

**文件：**
- 修改: `server/gateway/internal/dto/agent.go`（及所有 dto 文件）
- 修改: `server/gateway/internal/model/agent.go`（及所有 model 文件涉及的转换函数）

**步骤 1: 方案设计**

推荐方案：**Model 携带 DTO 嵌入 + 自定义 GORM tag**。

模型定义用复合方式：
```go
// model/agent.go
type AgentProfile struct {
    dto.AgentProfile `gorm:"embedded"` // 嵌入 JSON 字段，共享字段定义
    _ json.RawMessage //（用于检查编译）
}
```

但由于 GORM 的 `embedded` 机制和 JSON tag 不完全兼容，实际上更务实的方案是：

**实际可行方案：保留 dto 为 JSON 标签定义，model 用结构体组合方式，通过泛型转换器自动转换。**

更简单且可行：
```go
// model/agent.go
type AgentProfile struct {
    ID                string `gorm:"primaryKey;size:64" json:"id"`
    Name              string `gorm:"size:128;not null" json:"name"`
    ProviderID        string `gorm:"size:64;index" json:"provider_id,omitempty"`
    // ... 同时携带 gorm 和 json tag
}
```

这样 agent.go model 一个结构体即可同时用于 GORM 和 JSON 序列化。

**步骤 2: 合并 AgentProfile**

- 将 `dto/agent.go` 的 `AgentProfile` JSON tag 复制到 `model/agent.go`
- `dto/agent.go` 改为 `type AgentProfile = model.AgentProfile`（类型别名）

对于 `*JSON` 后缀的字段（tool_whitelist 等），在 model 中保留 `[]string` 类型并使用 GORM 的 `serializer:json`：

```go
ToolWhitelist []string `gorm:"column:tool_whitelist;serializer:json;type:text" json:"tool_whitelist,omitempty"`
```

**步骤 3: 合并其余实体**

按此模式处理：
- `AgentInstance` (model + dto)
- `Conversation` (model + dto)
- `ProviderProfile` (model + dto)
- `ScheduledTask` (model + dto)
- `ScheduledTaskRun` (model + dto)
- Session/Message/Run/RunEvent (sessionstore model + dto)

**步骤 4: 更新所有转换函数**

搜索并删除/调整手动 `ToDTO()` / `FromModel()` 代码。

```bash
rg "func.*(toDTO|ToDTO|fromModel|FromModel|toModel|ToModel)" --type go
```
工作目录: `server/gateway`

**步骤 5: 运行所有测试**

```bash
go test ./... -v
```
工作目录: `server/gateway`

**步骤 6: Commit**

```bash
git add server/gateway/internal/model/ server/gateway/internal/dto/
git commit -m "refactor(gateway): merge model and dto, eliminate duplicate struct definitions"
```

---

### Task 2.3: 移除 CRUD 中的手动转换代码

**文件：**
- 可能修改: `server/gateway/internal/service/agent_service.go`
- 可能修改: `server/gateway/internal/service/agent_instance_service.go`
- 可能修改: `server/gateway/internal/service/provider_service.go`
- 可能修改: `server/gateway/internal/service/scheduled_task_service.go`
- 可能修改: `server/gateway/internal/service/chat_service.go`

**步骤 1: 定位所有 model↔dto 转换代码**

```bash
rg "func.*(ToDTO|ToModel|toResponse|fromRequest)" --type go
```
工作目录: `server/gateway`

**步骤 2: 逐个替换为直接赋值或类型别名**

由于 model 已融合 dto 字段，转换逻辑大部分变为空操作。

**步骤 3: 运行测试**

```bash
go test ./... -v
```
工作目录: `server/gateway`

**步骤 4: Commit**

```bash
git add server/gateway/internal/service/
git commit -m "refactor(gateway/service): remove redundant model↔dto conversion code"
```

---

## Phase 3: Repository 层 CRUD 模板泛型化

**目标：** 提取 7 个 Repository 中 28 个重复的 CRUD 方法为泛型基类。

### Task 3.1: 创建泛型基类 Repository

**文件：**
- 创建: `server/gateway/internal/repository/base.go`
- 创建: `server/gateway/internal/repository/base_test.go`

**步骤 1: 设计 `BaseRepository[T]` 接口**

```go
package repository

import (
    "context"
    "gorm.io/gorm"
)

type BaseRepository[T any] interface {
    GetByID(ctx context.Context, id string) (*T, error)
    List(ctx context.Context, opts ...ListOption) ([]T, error)
    Create(ctx context.Context, entity *T) error
    Update(ctx context.Context, entity *T) error
    Delete(ctx context.Context, id string) error
}

type ListOption func(*gorm.DB) *gorm.DB

func WithLimit(limit int) ListOption {
    return func(db *gorm.DB) *gorm.DB { return db.Limit(limit) }
}

func WithOffset(offset int) ListOption {
    return func(db *gorm.DB) *gorm.DB { return db.Offset(offset) }
}

func WithOrder(order string) ListOption {
    return func(db *gorm.DB) *gorm.DB { return db.Order(order) }
}
```

**步骤 2: 实现 `GormBaseRepository[T]`**

```go
type GormBaseRepository[T any] struct {
    DB *gorm.DB
}

func (r *GormBaseRepository[T]) GetByID(ctx context.Context, id string) (*T, error) {
    var entity T
    if err := r.DB.WithContext(ctx).Where("id = ?", id).First(&entity).Error; err != nil {
        return nil, err
    }
    return &entity, nil
}

func (r *GormBaseRepository[T]) List(ctx context.Context, opts ...ListOption) ([]T, error) {
    var entities []T
    query := r.DB.WithContext(ctx)
    for _, opt := range opts {
        query = opt(query)
    }
    if err := query.Find(&entities).Error; err != nil {
        return nil, err
    }
    return entities, nil
}

func (r *GormBaseRepository[T]) Create(ctx context.Context, entity *T) error {
    return r.DB.WithContext(ctx).Create(entity).Error
}

func (r *GormBaseRepository[T]) Update(ctx context.Context, entity *T) error {
    return r.DB.WithContext(ctx).Save(entity).Error
}

func (r *GormBaseRepository[T]) Delete(ctx context.Context, id string) error {
    var entity T
    return r.DB.WithContext(ctx).Where("id = ?", id).Delete(&entity).Error
}
```

**步骤 3: 编写泛型测试**

```go
func TestGormBaseRepository_CRUD(t *testing.T) {
    db := setupTestDB(t)
    repo := &GormBaseRepository[TestEntity]{DB: db}
    
    // Test Create
    entity := &TestEntity{ID: "1", Name: "test"}
    err := repo.Create(context.Background(), entity)
    assert.NoError(t, err)
    
    // Test GetByID
    found, err := repo.GetByID(context.Background(), "1")
    assert.NoError(t, err)
    assert.Equal(t, "test", found.Name)
    
    // Test Update
    found.Name = "updated"
    err = repo.Update(context.Background(), found)
    assert.NoError(t, err)
    
    // Test List
    list, err := repo.List(context.Background())
    assert.NoError(t, err)
    assert.Len(t, list, 1)
    
    // Test Delete
    err = repo.Delete(context.Background(), "1")
    assert.NoError(t, err)
}
```

**步骤 4: 运行测试**

```bash
go test ./internal/repository -v -run TestGormBaseRepository
```
工作目录: `server/gateway`

**步骤 5: Commit**

```bash
git add server/gateway/internal/repository/base.go server/gateway/internal/repository/base_test.go
git commit -m "feat(repository): add generic BaseRepository to eliminate CRUD boilerplate"
```

---

### Task 3.2: 重构现有 Repository 使用泛型基类

**文件：**
- 修改: `server/gateway/internal/repository/agent_repository.go`
- 修改: `server/gateway/internal/repository/agent_instance_repository.go`
- 修改: `server/gateway/internal/repository/conversation_repository.go`
- 修改: `server/gateway/internal/repository/provider_repository.go`
- 修改: `server/gateway/internal/repository/scheduled_task_repository.go`
- 修改: `server/gateway/internal/repository/scheduled_task_run_repository.go`
- 修改: `server/gateway/internal/repository/agent_repository_test.go`

**步骤 1: 逐个重构每个 Repository**

以 AgentRepository 为例：

重构前：
```go
type GormAgentRepository struct {
    DB *gorm.DB
}

func (r *GormAgentRepository) GetByID(ctx context.Context, id string) (*model.AgentProfile, error) { ... }
func (r *GormAgentRepository) List(ctx context.Context) ([]model.AgentProfile, error) { ... }
func (r *GormAgentRepository) Create(ctx context.Context, a *model.AgentProfile) error { ... }
func (r *GormAgentRepository) Update(ctx context.Context, a *model.AgentProfile) error { ... }
func (r *GormAgentRepository) Delete(ctx context.Context, id string) error { ... }
```

重构后：
```go
type GormAgentRepository struct {
    *GormBaseRepository[model.AgentProfile]
}

func NewGormAgentRepository(db *gorm.DB) *GormAgentRepository {
    return &GormAgentRepository{
        GormBaseRepository: &GormBaseRepository[model.AgentProfile]{DB: db},
    }
}

// 只保留特殊方法（如按 Name 查询等非标准 CRUD）
func (r *GormAgentRepository) GetByName(ctx context.Context, name string) (*model.AgentProfile, error) { ... }
```

**步骤 2: 逐个文件修改并运行测试确认**

```bash
go test ./internal/repository -v
```
工作目录: `server/gateway`

**步骤 3: 更新 DI 容器中的 Repository 构建**

```bash
rg "NewGorm.*Repository" --type go
```
工作目录: `server/gateway`

**步骤 4: 整体编译和测试**

```bash
go test ./... -v
```
工作目录: `server/gateway`

**步骤 5: Commit**

```bash
git add server/gateway/internal/repository/
git commit -m "refactor(repository): use generic BaseRepository for all entities"
```

---

## Phase 4: Shared Infrastructure 合并（低优先级）

**目标：** 抽取 claw 和 gateway 的公共基础设施（DI 容器、Router 模式、Config 加载）。

### Task 4.1: 创建 server/shared 共享包

**文件：**
- 创建: `server/shared/go.mod`
- 创建: `server/shared/di/di.go`
- 创建: `server/shared/config/loader.go`
- 创建: `server/shared/router/helpers.go`

**步骤 1: 初始化共享模块**

```bash
mkdir server/shared && cd server/shared && go mod init github.com/icoo_claw/server/shared
```
工作目录根目录

**步骤 2: 抽取 DI 容器基类**

```go
// server/shared/di/di.go
package di

type Container struct {
    // 通用依赖（如 DB、Logger 等）
}

func NewContainer() *Container {
    return &Container{}
}
```

将 `claw/internal/di/container.go` 和 `gateway/internal/di/container.go` 中的公共部分（如数据库连接管理、日志初始化）抽取到 shared 包。

**步骤 3: 抽取 Config 加载器**

```go
// server/shared/config/loader.go
package config

import "github.com/BurntSushi/toml"

func LoadTOML(path string, cfg interface{}) error { ... }
```

统一 `claw/config/config.go` 和 `gateway/config/config.go` 中相同的 TOML 加载逻辑。

**步骤 4: 更新 go.work**

```go
use (
    ./desktop
    ./server/claw
    ./server/gateway
    ./server/shared
)
```

**步骤 5: 运行全量测试**

```bash
go test ./...
```
工作目录根目录（通过 go.work）

**步骤 6: Commit**

```bash
git add server/shared/ go.work
git commit -m "refactor: extract shared infrastructure package"
```

---

### Task 4.2: 重构 claw DI 使用 shared 包

**文件：**
- 修改: `server/claw/internal/di/container.go`
- 修改: `server/claw/cmd/claw/main.go`

**步骤 1: 重构 Container 继承 shared Container**

```go
import sharedDI "github.com/icoo_claw/server/shared/di"

type Container struct {
    *sharedDI.Container
    // claw-specific dependencies
}
```

**步骤 2: 运行测试**

```bash
go test ./... -v
```
工作目录: `server/claw`

**步骤 3: Commit**

---

### Task 4.3: 重构 gateway DI 使用 shared 包

同理 Task 4.2。

---

## Phase 5: Error Handling 统一（低优先级）

**目标：** 将全项目 408 处 `fmt.Errorf("...: %w", err)` 包装统一为项目级错误工具。

### Task 5.1: 创建 shared 错误包

**文件：**
- 创建: `server/shared/errors/errors.go`
- 创建: `server/shared/errors/errors_test.go`

**步骤 1: 定义常见错误包装器**

```go
// server/shared/errors/errors.go
package errors

import "fmt"

func Wrap(op string, err error) error {
    if err == nil {
        return nil
    }
    return fmt.Errorf("%s: %w", op, err)
}

func Wrapf(op string, err error, format string, args ...any) error {
    if err == nil {
        return nil
    }
    detail := fmt.Sprintf(format, args...)
    return fmt.Errorf("%s(%s): %w", op, detail, err)
}
```

**步骤 2: 编写测试**

---

### Task 5.2: 逐步替换高频文件

**步骤 1: 按优先级替换**

优先级（按 `fmt.Errorf` 调用次数）：
1. `sdk/tool/registry.go` (38 处)
2. `sdk/config/validator.go` (37 处)
3. `sessionstore/repository/gorm_session_repository.go` (16 处)

**步骤 2: 逐文件替换并运行测试**

---

## 实施顺序总结

| 阶段 | 任务 | 预计工时 | 风险 |
|------|------|---------|------|
| Phase 1.1 | 审计差异 | 0.5h | 低 |
| Phase 1.2 | 补充 agent_sdk/types.go | 0.5h | 低 |
| Phase 1.3 | 重构 claw/dto 为别名 | 1h | 中—需确认所有引用兼容 |
| Phase 1.4 | 重构 gateway/client | 1.5h | 中—跨模块依赖调整 |
| Phase 2.1 | 审计 model↔dto 差异 | 0.5h | 低 |
| Phase 2.2 | model↔dto 合并 | 2h | 高—影响范围广 |
| Phase 2.3 | 移除转换代码 | 0.5h | 中 |
| Phase 3.1 | 创建泛型 BaseRepository | 1h | 低 |
| Phase 3.2 | 重构 7 个 Repository | 1.5h | 中 |
| Phase 4.1 | 创建 shared 包 | 1h | 低 |
| Phase 4.2-4.3 | 重构 DI | 1h | 中 |
| Phase 5.1-5.2 | Error 工具封装 | 1.5h | 低 |
| **合计** | | **~12h** | |

## 前置条件检查

1. `go.work` 已包含 `server/claw` 和 `server/gateway` → gateway 可直接 import claw 的包
2. Go 1.25 已支持泛型（GORM v2+ 也兼容）
3. 现有测试覆盖：`server/claw` 和 `server/gateway` 均有测试，可作为回归验证

## 注意事项

- 每个 Task 完成后必须运行对应模块的 `go test ./...`，不可跳跃提交
- 类型别名（`type X = Y`）在 Go 中是完全透明的，不会破坏现有代码
- Gateway 通过 go.work 引用 claw 的包时，注意 go.mod 中是否需要添加 require 声明
- GORM `serializer:json` 是 v2 特性，需确认当前使用的 GORM 版本
