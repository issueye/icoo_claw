# Agent SDK 成熟度提升开发计划

日期：2026-05-29
范围：`common/core/agent_sdk`
当前评分：7.4 / 10
目标评分：8.6+ / 10

## 背景

当前 `agent_sdk` 已经具备可用的 agent runtime 基础能力：模型适配、工具系统、MCP、hooks、middleware、sandbox、skills、subagents、流式输出、上下文压缩、trace 和 session history 都已经存在。

近期重构也已经收敛了一批重复点，包括模型 retry、provider 缓存、runtime loop helper、内置工具工厂、skills/subagents 激活选择逻辑等。

下一阶段目标是把它从“内部可运行 runtime”推进到“稳定、易用、可被业务开发长期依赖的 SDK”。当前最关键的缺口集中在权限决策闭环、模型 provider 统一表达、开发者 API 易用性、调试回放能力和测试覆盖。

## 目标

1. 让工具执行权限显式、可测试、可强制执行。
2. 统一不同模型供应商的对话编码，减少 OpenAI Chat、OpenAI Responses、Anthropic 的行为差异。
3. 通过 builder、preset 和示例降低 SDK 使用门槛。
4. 增强调试能力，提供结构化 trace 和 replay artifact。
5. 补强高风险行为的测试覆盖。
6. 所有不兼容变更都要明确记录迁移方式。

## 非目标

1. 不替换现有模型 provider。
2. 不一次性重写整个 runtime loop。
3. 不在 `agent_sdk` 内部直接实现 UI 级确认弹窗。
4. 不改动 desktop、gateway、server 等无关模块行为。

## 重构策略

本计划采用覆盖式、不兼容优先的重构方式。目标是收敛重复路径和隐式行为，而不是为了短期兼容保留双轨实现。

1. 优先建立单一事实来源：权限、模型编码、工具注册、trace/replay 都应只有一个主入口。
2. 可以删除旧路径、旧默认行为和兼容 shim，但必须同步迁移说明和测试。
3. 新逻辑一旦接入 runtime 主链路，就不再保留旧链路并行绕行。
4. 对外行为变化必须明确写入文档，例如缺少 permission prompter 时 `ask` 默认拒绝。
5. 每次破坏式调整都要用小步提交范围控制风险，并保持 `go test ./common/core/agent_sdk/...` 通过。

## 阶段一：权限决策引擎

优先级：P0
风险：中

### 任务

1. 新增 `PermissionEvaluator`，可放在 `api` 或新的内部 policy 包。
2. 解析并执行 `settings.Permissions.Allow`、`Ask`、`Deny`、`DefaultMode`。
3. 增加 SDK 级确认回调，例如 `Options.PermissionPrompter`。
4. 被 deny 的工具必须在 hook 和真实工具执行前阻断。
5. 明确权限优先级：
   `deny > ask 拒绝 > hook deny > allow > default mode`。
6. 将权限决策写入 middleware/trace metadata。

### 验收标准

1. `deny` 规则会阻断工具执行，并写入结构化 tool error。
2. `ask` 规则会调用确认回调；没有回调时默认拒绝。
3. `acceptReadOnly` 模式下只自动接受只读工具。
4. 测试覆盖 allow、ask approve、ask reject、deny、default mode、subagent whitelist 交互。

## 阶段二：模型无关的对话 IR

优先级：P0
风险：高

### 任务

1. 引入内部 conversation IR，统一表达 message、tool call、tool result、多模态 block、reasoning content、cache hint。
2. runtime history 先转换为 IR，再交给各 provider encoder。
3. 实现 provider encoder：
   - Anthropic Messages
   - OpenAI Chat Completions
   - OpenAI Responses
4. 修复 OpenAI Responses 将 assistant/tool history 折叠成 user-only text 的问题。
5. 为多轮工具对话增加 golden tests。

### 验收标准

1. 多轮 assistant tool calls 和 tool results 在所有 provider encoder 下都能正确表达。
2. 多模态 user content 保持 provider 兼容。
3. Anthropic prompt cache hint 继续可用，且不会泄漏到不支持的 provider。
4. 现有 model/runtime 测试全部通过。

## 阶段三：开发者友好的 SDK API

优先级：P1
风险：中

### 任务

1. 新增 `AgentBuilder` 或 `RuntimeBuilder`。
2. 提供预设：
   - `CodingAgent`
   - `ResearchAgent`
   - `ToolOnlyAgent`
   - `SafeLocalAgent`
3. 增加常见 provider 的便捷构造方法。
4. 在 `common/core/agent_sdk/examples` 或 `docs` 下增加示例。
5. 保留 `Options`，但推荐业务方优先使用 builder。

### 验收标准

1. 创建一个最小 agent 不超过 15 行代码。
2. 示例覆盖 streaming、自定义工具、MCP server、permission prompter、subagent。
3. Builder 最终仍然清晰映射到 `Options`，不绕过 runtime 主路径。

## 阶段四：Trace Replay 与调试产物

优先级：P1
风险：中

### 任务

1. 在 trace 输出中增加稳定的 run id 和 iteration id。
2. 增加 replay artifact，包含脱敏后的 model request、tool call、tool result、settings snapshot、selected model metadata。
3. 增加 secret 和大 payload 的 redaction 层。
4. 提供一个 replay loader，用于测试和调试。

### 验收标准

1. 失败 run 能产出足够复现 model/tool 边界的结构化数据。
2. env、headers、API key 等敏感信息默认脱敏。
3. replay artifact 足够稳定，可用于回归测试。

## 阶段五：测试覆盖扩展

优先级：P1
风险：低

### 任务

1. 增加 model encoder golden tests。
2. 增加 permission evaluator tests。
3. 增加 MCP 注册和刷新测试。
4. 增加 hook 优先级和阻断测试。
5. 增加 compact/reactive compact 测试。
6. 增加 runtime loop 测试，覆盖 max iterations、max token escalation、stream stall fallback。

### 验收标准

1. 高风险行为都有对应单元测试。
2. `go test ./common/core/agent_sdk/...` 作为必跑验证命令。
3. 新测试不依赖真实网络或真实模型调用。

## 阶段六：文档与迁移说明

优先级：P2
风险：低

### 任务

1. 编写 SDK 总览文档。
2. 记录当前和后续重构产生的不兼容变更。
3. 增加权限配置指南。
4. 增加 Anthropic/OpenAI Chat/OpenAI Responses 的 provider 行为说明。
5. 增加 subagent/skill 编写指南。

### 验收标准

1. 开发者无需阅读 runtime 内部实现，也能搭建基础 agent。
2. 不兼容变更都有迁移步骤。
3. 示例从 SDK 总览文档中可直接访问。

## 推荐执行顺序

1. 阶段一：权限决策引擎。
2. 阶段二：模型无关的对话 IR。
3. 阶段五：补充 policy 和 provider contract 测试。
4. 阶段三：Builder 与示例。
5. 阶段四：Trace replay。
6. 阶段六：文档与迁移说明。

## 风险清单

| 风险 | 影响 | 缓解方式 |
| --- | --- | --- |
| 权限行为阻断现有工作流 | 高 | 仅在存在权限配置时启用严格行为；提供迁移说明。 |
| Provider IR 改动破坏工具调用格式 | 高 | 先写 golden tests，再替换 encoder。 |
| Builder 与 `Options` 语义重复 | 中 | Builder 只组合 `Options`，不绕过 runtime 路径。 |
| Replay artifact 泄漏敏感信息 | 高 | 写入前统一 redaction；测试 API key、headers、env。 |
| Subagent 权限与父 runtime 不一致 | 中 | 父级 whitelist、subagent context、permission rules 统一走同一 policy path。 |

## 完成定义

1. 每个阶段都有自动化测试或明确的人工验证方式。
2. `go test ./common/core/agent_sdk/...` 全部通过。
3. 新公开 API 提供示例。
4. 不兼容变更记录清楚。
5. 重新评分达到 8.6 / 10 或以上。
