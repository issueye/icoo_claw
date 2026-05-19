# d_page 2026-05-20 开发计划

日期：2026-05-20  
目标：将 `d_page` 从“可跑 demo 的 internal alpha”推进到“可准备 chat preview 的稳定组件包”。

## 1. 总体判断

明天的重点不是继续单纯堆组件，而是把渲染质量、测试边界、demo 验收和 chat 预接入协议稳住。

推荐优先级：

1. 布局稳定性和自动化交互测试。
2. chat preview 协议和宿主 action 边界。
3. 第二批基础组件扩展。

## 2. 上午：Demo 与布局稳定性

目标：让 `npm run demo` 在常见窗口宽度下表现稳定，避免再次出现内容被挤成窄条、文字竖排、卡片错位等问题。

任务：

- 复查 `即时输入预览` demo 的桌面布局。
- 复查 `Chat 结果卡片`、`基础表单`、`表格卡片` 三个已有示例。
- 验证 1280px、980px、375px 三个宽度下的布局表现。
- 优化 demo 分区：组件输入区、实时预览区、状态检查区。
- 检查右侧 runtime inspector 的横向和纵向溢出。

验收标准：

- 三个宽度下页面不出现文字竖排、卡片被挤成窄条、内容重叠。
- 输入区和预览区在桌面端左右排布，在窄屏自然上下堆叠。
- 右侧 inspector 不遮挡主预览内容。

## 3. 上午后半段：自动化交互测试

目标：把目前手动验收过的 demo 流程固化为可重复验证的测试或脚本，降低后续改渲染器时的回归风险。

建议覆盖路径：

- `即时输入预览`：输入标题后，预览标题同步变化。
- `即时输入预览`：textarea 输入后，预览正文同步变化。
- `即时输入预览`：select 切换后，`state.tone` 同步变化。
- `表格卡片`：点击表格行后，`selectedName` 同步变化。
- `Chat 结果卡片`：点击复制按钮后，右侧记录 adapter 调用。
- `Chat 结果卡片`：点击打开详情后，右侧记录 `emit: openDetails`。

验收标准：

- 自动化测试或可重复检查脚本可以独立运行。
- 至少覆盖 button、input、textarea、select、table rowSelect 五类交互。
- 保留 `npm run test` 和 `npm run build` 作为最小门禁。

## 4. 下午：第二批基础组件

目标：补齐 chat 和后台工具常见 UI 所需的小型组件，继续保持组件包通用性。

优先新增组件：

- `tag`：状态标签、分类标签。
- `divider`：内容分隔。
- `checkbox`：布尔选择。
- `switch`：开关选择。
- `image`：基础图片展示，带空状态和加载失败降级。

组件约束：

- 只接收解析后的 props。
- 只通过事件 emit 交互。
- 不直接读取原始 schema。
- 不直接修改 runtime state。
- 必须有默认空状态或降级显示。

配套更新：

- 更新 `defaultComponents`。
- 更新 `componentRegistry` 测试预期。
- 在 README 中补充组件清单。
- 如有必要，新增一个 `component-gallery.json` 示例。

## 5. 下午后半段：chat preview 接入方案

目标：明确 `0.2.0 chat preview` 的接入边界，暂不直接替换 chat 主流程。

建议 message metadata：

```json
{
  "metadata": {
    "render_type": "d_page",
    "d_page_schema": {}
  }
}
```

feature flag 策略：

- 默认关闭。
- 仅 demo/dev 环境或实验开关开启。
- schema 无效时回退普通文本消息。
- action 执行失败时在卡片内降级提示，不影响整条消息。

宿主 action 清单：

- `copyToComposer`
- `sendChatPrompt`
- `saveArtifact`
- `openArtifact`
- `openExternalUrl`

验收标准：

- 形成一份 `0.2.0 chat preview` 计划文档。
- 写清楚 chat 侧文件范围、回退路径和不开启时的默认行为。
- 明确 `d_page` 包不 import chat store、不直接操作 desktop 业务状态。

## 6. 收尾验收

每天结束前执行：

```powershell
npm run test
npm run build
```

同时在浏览器打开：

```text
http://127.0.0.1:9360
```

人工验收：

- 四份 demo schema 都能切换。
- `即时输入预览` 能即时更新。
- `表格卡片` 能点击行更新状态。
- `Chat 结果卡片` 能记录 adapter 和 emit 事件。
- 页面布局没有横向挤压和内容重叠。

## 7. 暂不做事项

- 不直接接入 chat 主消息流。
- 不做可视化编辑器。
- 不做远程 schema 管理后台。
- 不引入任意 JavaScript 表达式执行。
- 不把 `d_page` 包耦合到 desktop Pinia store。

## 8. 明日交付物清单

- 布局稳定性修复和验证记录。
- demo 交互自动化测试或可重复检查脚本。
- 第二批基础组件及注册表更新。
- README 组件清单更新。
- `0.2.0 chat preview` 接入方案文档。
- 最终 `npm run test` 与 `npm run build` 通过记录。
