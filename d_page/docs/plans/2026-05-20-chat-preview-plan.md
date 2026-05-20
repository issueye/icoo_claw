# d_page 0.2.0 Chat Preview 接入方案

日期：2026-05-20

## 目标

在 chat 中实验性预览 `d_page` 渲染结果，但不替换现有消息主流程。`d_page` 继续保持为独立组件包，只通过 schema、runtime options、host adapters 与 chat 宿主通信。

## Message Metadata

chat 消息可以通过 metadata 声明动态页面渲染：

```json
{
  "metadata": {
    "render_type": "d_page",
    "d_page_schema": {}
  }
}
```

## Feature Flag

- 默认关闭。
- 仅在 demo/dev 环境或明确实验开关开启时渲染。
- flag 关闭时，chat 继续展示普通文本消息。
- `render_type !== "d_page"` 时，chat 继续展示普通文本消息。
- `d_page_schema` 为空、类型错误或校验失败时，回退普通文本消息，并记录可观测错误。

## Chat 侧文件范围

建议只在 chat 消息展示层增加薄适配：

- 消息卡片/消息内容渲染组件：判断 metadata 和 feature flag。
- d_page runtime adapter 文件：创建 runtime，注入 host actions 和 adapters。
- 实验开关配置：控制 dev/demo 与生产默认行为。
- 可选测试文件：覆盖 flag 关闭、schema 无效、action 失败三条回退路径。

chat store、会话状态、消息发送流程不应被 `d_page` 包直接 import 或修改。

## Host Action 清单

chat 宿主按需注入以下 action，`d_page` 包只调用 runtime action，不直接触碰 chat 业务状态：

- `copyToComposer`: 将文本复制到输入框草稿。
- `sendChatPrompt`: 使用指定文本触发一次用户确认后的发送。
- `saveArtifact`: 将 schema 结果或文本保存为 artifact。
- `openArtifact`: 打开已有 artifact。
- `openExternalUrl`: 打开外部链接，宿主负责安全校验和权限提示。

## 回退与错误处理

- schema 无效：展示普通文本消息。
- 未注册组件：卡片内显示 unknown component fallback，不阻塞整条消息。
- action 执行失败：卡片内显示当前 action 错误，不影响消息本体。
- adapter 缺失：runtime 返回 `ADAPTER_MISSING`，由卡片错误区或宿主 telemetry 记录。
- 外链打开失败：宿主 action 返回失败结果，卡片内降级提示。

## 包边界

- `d_page` 不 import chat store。
- `d_page` 不直接操作 desktop 业务状态。
- `d_page` 不内置任意网络请求 action。
- `d_page` 不执行任意 JavaScript 表达式。
- chat 宿主负责权限、审计、外链策略和用户确认。

## 0.2.0 验收

- flag 关闭时默认行为完全不变。
- flag 开启且 schema 有效时，消息内容区域能渲染 `DPageRenderer`。
- schema 无效时回退普通文本消息。
- host action 失败时仅卡片内降级。
- 测试覆盖默认关闭、有效 schema、无效 schema 和 action 失败。
