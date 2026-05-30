# Gateway ↔ Claw 技能设计优化方案

> **Goal:** 消除 Gateway 与 Claw 之间技能传递的多通道不一致、修复 Agent 级技能绑定失效、补齐 Gateway 端校验缺失、建立版本感知技能激活机制。

**Architecture:** 统一技能注入路径为「Gateway 写文件 + Claw 从 ProjectRoot 加载」的单一通道，per-request 仅传递 `force_skills` 指令，元数据不再重复注入。Gateway 增加 SDK 级名校验。引入技能版本选择能力。

**Tech Stack:** Go 1.25, GORM, Gin, YAML

---

## 现状问题回顾

| 问题 | 严重度 | 影响 |
|------|--------|------|
| Agent `SkillIDsJSON` 被 per-request `SyncSummary()` 全量覆盖 | **高** | Agent 级技能绑定完全失效 |
| 技能元数据经 3 条通道传递（TOML config / per-request profile / SKILL.md 文件） | **高** | 数据不一致风险 |
| 同一技能元数据在 SQLite、YAML frontmatter、TOML JSON、request JSON 四地冗余 | **中** | 维护困难，同步无事务保证 |
| Gateway 端无技能名校验（SDK 有严格 regex） | **中** | 不合法名称技能可创建但运行时静默丢弃 |
| 版本信息未被运行时使用（仅存于元数据） | **中** | 无法版本回退/灰度 |
| `GatewaySkillInfo` 与 `SkillSummaryItem` 字段不一致（缺少 Version） | **低** | 数据不完整 |
| 无技能执行结果反馈回路 | **低** | Gateway 无法统计/限流 |
| Session 粒度无技能隔离 | **低** | 所有会话共享全部技能 |

---

## 当前架构 (As-Is)

```
┌─────────────────────────────────────────────────────────────────┐
│                         Gateway                                  │
│                                                                  │
│  SkillService ──→ SQLite (SkillProfile)                         │
│       │                                                          │
│       ├──→ publishSkillFiles() → {base}/active/.agents/skills/  │
│       │                    └→ {base}/versions/{name}/{version}/  │
│       │                                                          │
│       └──→ SyncSummary() ← 每次请求调用，返回全部 active 技能    │
│                                                                  │
│  AgentInstanceService.Start()                                    │
│       │                                                          │
│       └──→ writeClawConfig() → per-instance TOML                │
│              [gateway_skills]                                     │
│              path = "{base}/active"                              │
│              json = {"skills":[{name,...}]}  ← 只含名称          │
│                                                                  │
│  ChatService.agentProfilePayload()                               │
│       │                                                          │
│       └──→ profile["gateway_skills"] = SyncSummary() ← 全量覆盖 │
│            profile["project_root"] = summary.Path                │
│                                                                  │
└──────────────────────┬──────────────────────────────────────────┘
                       │ RunRequest.Agent["gateway_skills"]
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                          Claw                                    │
│                                                                  │
│  di.NewContainer()                                               │
│       │                                                          │
│       ├──→ GatewaySkillsFromJSON(cfg.GatewaySkills.JSON)         │
│       └──→ runtimeFactory.SetGatewaySkills()                     │
│                                                                  │
│  RuntimeFactory.New()                                            │
│       │                                                          │
│       ├──→ parseAgentProfile(req.Agent)                          │
│       │     profile.GatewaySkills ← 从 req 提取                  │
│       │     若空则回退到 f.gatewaySkills ← 启动时配置            │
│       │                                                          │
│       └──→ api.New(options)                                      │
│              ProjectRoot = gateway_skills.path                   │
│              → buildSkillsRegistry()                             │
│                → LoadFromFS({root}/.agents/skills/**/SKILL.md)   │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

**核心问题**：图中红色路径（per-request `SyncSummary`）每次都覆盖蓝色路径（per-instance config），导致 Agent 的 `SkillIDsJSON` 筛选白费。

---

## 目标架构 (To-Be)

```
┌─────────────────────────────────────────────────────────────────┐
│                         Gateway                                  │
│                                                                  │
│  SkillService                                                    │
│       │                                                          │
│       ├──→ publishSkillFiles() → {base}/active/.agents/skills/  │
│       │          └→ ValidateName() ← 新增 SDK 级正则校验         │
│       │                                                          │
│       └──→ SyncSummary() → 仍保留，供管理端查询用               │
│                                                                  │
│  AgentInstanceService.Start()                                    │
│       │                                                          │
│       └──→ writeClawConfig() → per-instance TOML                │
│              [gateway_skills]                                     │
│              path = "{base}/active"                              │
│              skills = [从 Agent.SkillIDsJSON 筛选]  ← 只传绑定   │
│                                                                  │
│  ChatService.agentProfilePayload()                               │
│       │                                                          │
│       ├──→ 不再注入 gateway_skills ← 移除覆盖逻辑                │
│       ├──→ 保留 project_root = GatewaySkillsDir/active           │
│       └──→ force_skills = req.ForceSkills（用户显式触发）        │
│                                                                  │
└──────────────────────┬──────────────────────────────────────────┘
                       │ RunRequest.Agent["project_root"]
                       │ RunRequest.ForceSkills (optional)
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                          Claw                                    │
│                                                                  │
│  RuntimeFactory.New()                                            │
│       │                                                          │
│       ├──→ ProjectRoot = profile.ProjectRoot                     │
│       │     若空 → f.gatewaySkills.Path                          │
│       │                                                          │
│       ├──→ api.New(options)                                      │
│       │     → buildSkillsRegistry()                              │
│       │       → LoadFromFS({root}/.agents/skills/**/SKILL.md)   │
│       │                                                          │
│       └──→ ForceSkills ← req.ForceSkills 直接驱动               │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

**关键变化**：
1. `SyncSummary()` 不再注入到 per-request agent profile — 消除覆盖
2. `force_skills` 成为 Gateway → Claw 唯一的运行时技能指令通道
3. Agent 的 `SkillIDsJSON` 在实例启动时绑定，Claw 只加载绑定技能
4. Gateway 增加技能名校验，阻止不合法名称入库
5. `GatewaySkillInfo` 补全 `Version` 字段

---

## Phase 1: 修复 Agent 级技能绑定失效 (P0)

**目标：** 移除 per-request `SyncSummary()` 对 `gateway_skills` 的覆盖，让 Agent `SkillIDsJSON` 的绑定在实例级生效。

### Task 1.1: 移除 ChatService 中的技能全量注入

**File:** `server/gateway/internal/service/chat_service.go`

**Step 1:** 修改 `agentProfilePayload()` — 移除 `SyncSummary()` 调用及 `gateway_skills` 注入

```go
// Before (L345-356):
if s != nil && s.skills != nil {
    summary, err := s.skills.SyncSummary(ctx)
    if err != nil {
        return nil, err
    }
    profile["gateway_skills"] = summary
    if projectRoot := strings.TrimSpace(summary.Path); projectRoot != "" {
        if _, exists := profile["project_root"]; !exists {
            profile["project_root"] = projectRoot
        }
    }
}

// After:
if s != nil && s.skills != nil {
    if _, exists := profile["project_root"]; !exists {
        profile["project_root"] = s.skills.ActiveSkillPath()
    }
}
```

**Step 2:** 检查 `ScheduledTaskService` 是否也有同样逻辑 — 如有，同上处理。

### Task 1.2: SkillService 暴露 ActiveSkillPath

**File:** `server/gateway/internal/service/skill_service.go`

将 `activeSkillPath()` 从私有改为公开方法 `ActiveSkillPath()`：

```go
// Before:
func (s *SkillService) activeSkillPath() string {

// After:
func (s *SkillService) ActiveSkillPath() string {
```

### Task 1.3: 验证 Agent.SkillIDsJSON 绑定生效

**File:** `server/gateway/internal/service/agent_instance_service.go`

确认 L67 逻辑正确：
```go
spec.GatewaySkills.Skills = gatewaySkillInfos(parseStringSlice(agent.SkillIDsJSON))
```

若 `SkillIDsJSON` 为空，则 `gatewaySkillInfos` 返回 nil，Claw 端应加载全部技能（保持向后兼容）。

**File:** `server/claw/pkg/agent_sdk/runtime_factory.go`

确认 `parseAgentProfile` 中 `gateway_skills` 路径逻辑 — 移除 per-request 路径覆盖：

```go
// Before (L34-36):
if strings.TrimSpace(profile.GatewaySkills.Path) == "" {
    profile.GatewaySkills = f.gatewaySkills
}

// After — 删除此逻辑，直接使用启动时配置:
// 不从 per-request profile 中提取 gateway_skills，仅从 factory 配置读取
if strings.TrimSpace(profile.GatewaySkills.Path) != "" {
    // per-request 传了 path，优先使用（保持兼容性窗口）
} else {
    profile.GatewaySkills = f.gatewaySkills
}
```

**实际上更好的做法**：`parseAgentProfile` 不再从 request 中提取 `gateway_skills`（即删除 L115 的 `gatewaySkillsValue` 调用），因为 Path 已由 `project_root` 覆盖，Skills 列表应完全由文件系统决定。

### Task 1.4: 端到端验证

1. Gateway 创建 Agent A，绑定 skills = ["git-helper"]
2. Gateway 创建 Agent B，绑定 skills = ["docker-helper"]
3. 启动两个 Agent 实例
4. 对 A 发消息，确认 Claw 日志中加载的 skills 仅包含 "git-helper"
5. 对 B 发消息，确认 Claw 日志中加载的 skills 仅包含 "docker-helper"

---

## Phase 2: 统一技能传递通道 (P1)

**目标：** 技能信息从「三通道」简化为「Gateway 写文件 + Claw 读文件」的单一通道。

### Task 2.1: 简化 per-instance TOML config

**File:** `server/gateway/internal/service/process_supervisor.go`

移除 `writeClawConfig` 中 skills JSON 的注入（L249-252），只保留 path：

```go
// Before:
gatewaySkillsJSON, err := json.Marshal(map[string]any{"gateway_skills": spec.GatewaySkills})
// ...
payload := fmt.Sprintf("...\n[gateway_skills]\npath = %q\njson = %q\n", ...)

// After:
payload := fmt.Sprintf("...\n[gateway_skills]\npath = %q\n", ..., spec.GatewaySkills.Path)
```

### Task 2.2: 移除 Claw 端 GatewaySkills.Skills 依赖

**File:** `server/claw/pkg/agent_sdk/runtime_factory.go`

- `GatewaySkills` struct 移除 `Skills` 字段，保留 `Path`
- `GatewaySkillsFromJSON` 简化：只提取 `path`
- Claw DI 不再通过 JSON 注入技能列表

**File:** `server/claw/internal/di/container.go`

简化 `gatewaySkills` 设置逻辑：

```go
// Before (L33-38):
gatewaySkills := agent_sdk.GatewaySkillsFromJSON(cfg.GatewaySkills.JSON)
if strings.TrimSpace(gatewaySkills.Path) == "" {
    gatewaySkills.Path = cfg.GatewaySkills.Path
}
runtimeFactory.SetGatewaySkills(gatewaySkills)

// After:
runtimeFactory.SetGatewaySkills(agent_sdk.GatewaySkills{
    Path: cfg.GatewaySkills.Path,
})
```

### Task 2.3: 移除 GatewaySkillsConfig.Skills 字段

**File:** `server/gateway/internal/service/process_supervisor.go`

```go
// Before:
type GatewaySkillsConfig struct {
    Path   string             `json:"path"`
    Skills []GatewaySkillInfo `json:"skills"`
}

// After:
type GatewaySkillsConfig struct {
    Path string `json:"path"`
}
```

**File:** `server/gateway/internal/service/agent_instance_service.go`

删除 L67：
```go
// 删除：
spec.GatewaySkills.Skills = gatewaySkillInfos(parseStringSlice(agent.SkillIDsJSON))
```

`gatewaySkillInfos` 函数可一并删除。

### Task 2.4: Agent.SkillIDsJSON 的生效方式调整

Agent 的 `SkillIDsJSON` 不再通过 TOML config 传给 Claw，改为两种可选策略：

**策略 A（推荐）**：在 SkillService 发布时按 Agent 生成隔离目录
- 每个 Agent 实例拥有独立的 skills 目录：`{base}/agents/{agentId}/.agents/skills/`
- `ProjectRoot` 指向该隔离目录

**策略 B（简单）**：Gateway 启动实例前，按 Agent 的 `SkillIDsJSON` 过滤复制 SKILL.md 到实例专属目录。Claw 不变。

**建议采用策略 B** 作为 Phase 2 方案，策略 A 作为 Phase 4 的可选增强。

---

## Phase 3: Gateway 端补齐校验与一致性 (P1)

### Task 3.1: 增加技能名校验

**File:** `server/gateway/internal/service/skill_service.go`

在 `Create` 和 `Update` 方法中增加名称校验：

```go
import "regexp"

var skillNameRegexp = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)

func isValidSkillName(name string) bool {
    return skillNameRegexp.MatchString(strings.TrimSpace(name))
}
```

在 `Create()` L33 后添加：
```go
if !isValidSkillName(skill.Name) {
    return nil, fmt.Errorf(
        "invalid skill name %q (must be 1-64 chars, lowercase alphanumeric + hyphens)",
        skill.Name,
    )
}
```

在 `Update()` L84 后添加相同校验。

### Task 3.2: GatewaySkillInfo 补全 Version 字段

**File:** `server/gateway/internal/service/process_supervisor.go`

```go
type GatewaySkillInfo struct {
    Name        string `json:"name"`
    Description string `json:"Description"`
    Version     string `json:"version"`    // 新增
}
```

同步更新 `gatewaySkillInfos()` 函数，从 SkillProfile 中读取 Version。若该结构最终在 Phase 2 被删除则跳过此项。

### Task 3.3: DTO 字段命名一致性

**File:** `server/gateway/internal/dto/skill.go`

`SkillSummaryItem.Description` 的 JSON tag 是 `"Description"`（大写 D），与 Go 惯例 `"description"` 不一致。统一为小写：

```go
type SkillSummaryItem struct {
    Name        string `json:"name"`
    Description string `json:"description"`  // 修正
    Version     string `json:"version"`
}
```

同时检查前端消费方是否需要同步修改。

---

## Phase 4: 版本感知技能激活 (P2)

**目标：** 技能选择支持指定版本，支持版本回退。

### Task 4.1: Agent 模型增加技能版本绑定

**File:** `server/gateway/internal/model/agent.go`

将 `SkillIDsJSON` 从 `[]string`（纯名称列表）改为结构化绑定：

```go
// 旧格式: ["git-helper", "docker-helper"]
// 新格式: [{"name":"git-helper","version":"v2"},{"name":"docker-helper"}]

// Agent 模型保持不变（SkillIDsJSON string），解析逻辑升级
```

`gatewaySkillInfos` 升级为解析结构体数组。

### Task 4.2: SkillService 按版本生成隔离目录

**File:** `server/gateway/internal/service/skill_service.go`

新增方法 `PublishForAgent(agentID, skillIDsJSON string) error`：
1. 解析 `SkillIDsJSON` → `[]{name, version}`
2. 创建 `{base}/agents/{agentID}/.agents/skills/{name}/SKILL.md`
3. 内容从 `{base}/versions/{name}/{version}/SKILL.md` 复制

### Task 4.3: Agent 实例 ProjectRoot 指向隔离目录

**File:** `server/gateway/internal/service/agent_instance_service.go`

在 `Start()` 中，调用 `SkillService.PublishForAgent()` 生成实例专属 skills 目录，设置 `ProjectRoot` 为该目录。

```go
if s.skillService != nil {
    skillsRoot, err := s.skillService.PublishForAgent(instanceID, agent.SkillIDsJSON)
    if err != nil {
        return nil, fmt.Errorf("publish skills for agent: %w", err)
    }
    spec.GatewaySkills.Path = skillsRoot
}
```

---

## Phase 5: 可观测性与反馈回路 (P3)

### Task 5.1: Claw 暴露技能执行指标

**File:** `common/core/agent_sdk/tool/builtin/skill.go`

`SkillTool.Execute()` 中记录执行结果（成功/失败、耗时），写入 session metadata 或返回给 Gateway：

在 `Execute()` 返回的 `ToolResult` 中携带技能执行统计：
```go
return &tool.ToolResult{
    Success: true,
    Output:  output,
    Data: map[string]interface{}{
        "skill":     result.Skill,
        "output":    result.Output,
        "metadata":  result.Metadata,
        "duration_ms": elapsed.Milliseconds(),  // 新增
    },
}, nil
```

### Task 5.2: Gateway 记录技能使用统计

在 Gateway 的 ChatService 中，解析 StreamEvent 中的 tool_call 事件，提取 skill 调用信息写入数据库或日志。

---

## 实施优先级与依赖

```
Phase 1 (P0) ──→ Phase 2 (P1) ──→ Phase 3 (P1) ──→ Phase 4 (P2) ──→ Phase 5 (P3)
     │                                                  │
     └── 可独立交付 ← 修复核心 bug                        └── 需 Phase 2 完成
```

| Phase | 优先级 | 估时 | 依赖 | 可独立交付 |
|-------|--------|------|------|-----------|
| Phase 1: 修复绑定失效 | P0 | 3h | 无 | 是 |
| Phase 2: 统一传递通道 | P1 | 4h | Phase 1 | 是 |
| Phase 3: 补齐校验 | P1 | 2h | 无 | 是 |
| Phase 4: 版本感知 | P2 | 5h | Phase 2 | 否 |
| Phase 5: 可观测性 | P3 | 3h | Phase 1 | 是 |

---

## 风险与回滚

| 风险 | 缓解措施 |
|------|---------|
| Phase 1 移除 SyncSummary 后前端技能列表显示异常 | SyncSummary 保留用于管理 API，仅移除 injection 路径 |
| Phase 2 删除 TOML JSON 后旧 Claw 二进制不兼容 | 保留 JSON 字段但置空，分两个版本逐步移除 |
| Phase 4 隔离目录导致磁盘空间增长 | 使用符号链接 + 单例共享文件，仅差异部分复制 |

---

## 涉及文件清单

| 文件 | Phase | 操作 |
|------|-------|------|
| `server/gateway/internal/service/chat_service.go` | 1 | 修改 |
| `server/gateway/internal/service/skill_service.go` | 1, 3, 4 | 修改 |
| `server/gateway/internal/service/agent_instance_service.go` | 1, 2, 4 | 修改 |
| `server/gateway/internal/service/process_supervisor.go` | 2, 3 | 修改/删除 |
| `server/gateway/internal/dto/skill.go` | 3 | 修改 |
| `server/claw/pkg/agent_sdk/runtime_factory.go` | 1, 2 | 修改 |
| `server/claw/internal/di/container.go` | 2 | 修改 |
| `server/claw/internal/config/config.go` | 2 | 修改 |
| `common/core/agent_sdk/tool/builtin/skill.go` | 5 | 修改 |
| `server/gateway/internal/model/agent.go` | 4 | 不变 |
