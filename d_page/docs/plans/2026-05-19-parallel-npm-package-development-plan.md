# d_page npm 组件包并行开发计划

日期：2026-05-19  
目标：将 `d_page` 从文档态推进为可独立构建、测试、预览，并可被 chat 后续接入的 Vue npm 组件包。

## 0. 当前进度快照

完成情况图片：[`../assets/2026-05-19-d-page-completion-status.svg`](../assets/2026-05-19-d-page-completion-status.svg)

截至 2026-05-19 当前收口判断：`d_page` 已经从纯文档阶段推进到可独立测试、构建的 npm 组件包 internal alpha，整体 MVP 完成度约 78%。它已经具备 `@icoo-claw/d-page` 组件包的基本闭环，但还不能直接视为可接入 chat 主流程的成熟版本。

### 0.1 已完成或基本完成

- 包骨架与构建入口：约 90%。已经出现 `package.json`、`vite.config.js`、`src/index.js`、`src/style.css`、`package-lock.json` 和构建产物，包名为 `@icoo-claw/d-page`，并采用 `vue`、`pinia` peer dependency 方案。
- Runtime 与安全动作：约 90%。已经建立 schema normalize/validate、binding resolve、runtime 创建、action 执行、action registry 和默认 actions；binding 限制在 `state`、`data`、`context` 安全路径读取范围内。
- 基础组件与注册表：约 88%。已经具备 text、heading、button、input、textarea、select、card surface、table、alert、stat、list 等基础组件，以及 component registry/default components。
- Vue 页面渲染器：约 75%。`DPageRenderer` 已接入 schema/runtime/component registry/schema 校验，`DCardRenderer` 已支持 root 递归、children、slots、props binding、事件 action、未知组件降级和卡片级错误显示。
- 示例、文档与体验验证：约 78%。已补充 `chat-tool-result.json`、`simple-form.json`、`table-card.json`，并在 README 中说明包 API、基础用法、宿主 action/adapter 注入和 chat metadata 方向。
- Demo 预览：约 82%。已新增 `npm run demo` playground，可在 `http://127.0.0.1:9360` 切换四份示例 schema，并查看 runtime state、data、宿主事件和 adapter 调用。
- 核心单测：已有 `resolveBinding`、`validateSchema`、`runtimeActions`、`componentRegistry`、`examples`、`renderer` 六组测试文件，覆盖运行时、注册表、示例 schema 和 SSR 渲染烟测。

### 0.2 进行中或待补齐

- 浏览器预览与真实交互：约 70%。当前已通过单测、SSR 烟测、library build 和内置浏览器手动验证；后续可补自动化浏览器测试。
- chat 预接入准备：约 25%。协议方向明确，但本阶段默认不改 chat 主流程。下一阶段应先完成 feature flag、宿主 action 清单和失败回退策略，再进入 `0.2.0 chat preview`。

### 0.3 当前风险

- 当前存在本地 `node_modules/`、`dist/` 等安装/构建产物，已经通过忽略规则避免进入提交范围，但仍不应作为交付物提交。
- 浏览器端交互还只经过构建和 SSR 烟测，尚未做真实页面预览截图和点击路径验证。
- chat 接入仍需谨慎：`d_page` 不应直接依赖 chat store，必须由宿主注入 action 和 adapter，并通过 feature flag 灰度。

## 0.4 下一步内容

下一步优先做“可视化验收和 chat preview 设计”，而不是直接替换 chat 主消息渲染：

2026-05-20 的具体执行计划见：`docs/plans/2026-05-20-development-plan.md`。

1. 补自动化浏览器测试，覆盖 button click、input update、table rowSelect 这些真实事件路径。
2. 设计 `0.2.0 chat preview` 的 feature flag、metadata 协议、宿主 action 注入清单和失败回退策略。
3. 继续增强基础组件：checkbox、switch、tag、divider、image，并补充主题变量和移动端布局验证。
4. 将 demo 验证截图纳入开发记录，方便后续 UI 回归对比。
5. 在进入 chat 主流程前，保持 `npm run test` 与 `npm run build` 作为最小门禁。

## 1. 需求分解

`d_page` 的 npm 包化可以拆成六类需求。

### 1.1 工程与包入口

目标是让 `d_page` 成为标准 npm package，而不是普通业务应用。

交付内容：

- `package.json`
- `vite.config.js`
- `README.md`
- `src/index.js`
- `src/style.css`
- `tests/` 基础测试环境
- `schemas/examples/` 示例 schema

关键要求：

- 包名：`@icoo-claw/d-page`
- 构建模式：Vite library mode
- `vue`、`pinia` 使用 `peerDependencies`
- 对外只暴露稳定 API
- 初期 `private: true`，先支持 `file:../../d_page` 本地依赖

### 1.2 Runtime 与协议层

目标是把 schema、state、context、binding、action 执行从 Vue 组件里拆出去，形成可测试的纯逻辑核心。

交付内容：

- `src/runtime/createDPageRuntime.js`
- `src/runtime/normalizeSchema.js`
- `src/runtime/validateSchema.js`
- `src/runtime/resolveBinding.js`
- `src/runtime/executeAction.js`
- `src/registry/createActionRegistry.js`
- `src/registry/defaultActions.js`
- runtime 单测

关键要求：

- 支持 `{{ state.xxx }}`、`{{ data.xxx }}`、`{{ context.xxx }}` 安全路径读取。
- 不支持任意 JavaScript 表达式。
- 默认 action 限制为安全能力：`setState`、`emit`、`copyText`、`openUrl`、`chain`。
- `request` 不作为包内默认 action，后续由宿主注入。
- schema 校验至少覆盖 root、card type、id 唯一性、children/slots 类型、component 注册存在性。

### 1.3 Vue 渲染器层

目标是实现卡片树递归渲染和错误边界。

交付内容：

- `src/components/DPageRenderer.vue`
- `src/components/DCardRenderer.vue`
- `src/components/DUnknownComponent.vue`
- `src/components/DRenderError.vue`
- renderer 相关测试或 demo 验证

关键要求：

- `DPageRenderer` 接收 `schema`、`runtime`、`context`。
- `DCardRenderer` 从 `root` 开始递归渲染 `children` 和 `slots`。
- 未知组件显示降级占位，不白屏。
- 单个卡片渲染异常不影响整页。
- 组件 props 在传入前完成 binding resolve。

### 1.4 内置基础组件层

目标是提供可用的最小组件集，先服务 chat 工具结果、配置表单、摘要表格等场景。

交付内容：

- `src/components/base/DText.vue`
- `src/components/base/DHeading.vue`
- `src/components/base/DButton.vue`
- `src/components/base/DInput.vue`
- `src/components/base/DCardSurface.vue`
- `src/components/base/DTable.vue`
- `src/registry/createComponentRegistry.js`
- `src/registry/defaultComponents.js`

关键要求：

- 基础组件必须有默认样式和空状态。
- 不直接读取原始 schema。
- 不直接修改全局状态，通过事件交给 runtime/action。
- `defaultComponents` 可被宿主扩展或覆盖。

### 1.5 示例、测试与文档

目标是保证包可以独立验证，也方便 chat 后续试接入。

交付内容：

- `src/schemas/examples/chat-tool-result.json`
- `src/schemas/examples/simple-form.json`
- `src/schemas/examples/table-card.json`
- `tests/resolveBinding.spec.js`
- `tests/validateSchema.spec.js`
- `tests/runtimeActions.spec.js`
- `tests/componentRegistry.spec.js`
- `README.md` 使用示例

关键要求：

- 示例覆盖 chat 工具结果、基础表单、表格卡片。
- 单测覆盖安全 binding、schema 错误、action 执行和注册表扩展。
- 文档给出 `file:../../d_page` 的本地使用方式。

### 1.6 chat 预接入方案

目标不是立即改 chat 主流程，而是准备一个可控接入点。

交付内容：

- `d_page` 包内 `chat-tool-result.json` 示例。
- `desktop/frontend` 后续接入说明或实验计划。
- 可选：后续新增 `desktop/frontend/src/services/d_page/runtime.js`，但本阶段默认不改 chat 主消息渲染。

关键要求：

- chat 只按 `message.metadata.render_type === 'd_page'` 分支渲染。
- d_page schema 来自 `message.metadata.d_page_schema`。
- chat 注入宿主 action，例如 `copyToComposer`、`sendChatPrompt`、`saveArtifact`。
- 默认不开启，等包 `0.2.0 chat preview` 再接。

## 2. 并行工作流划分

### Workstream A：包骨架与公共 API

可并行性：第一优先级，其他工作流依赖它的目录和导出约定。

写入范围：

- `d_page/package.json`
- `d_page/vite.config.js`
- `d_page/README.md`
- `d_page/src/index.js`
- `d_page/src/style.css`
- `d_page/.gitignore`（如需要）

任务：

1. 初始化 npm package。
2. 配置 Vite library mode。
3. 配置 scripts：`dev`、`build`、`test`、`preview`。
4. 配置 `peerDependencies`。
5. 先导出占位 API，保证其他分支能接入。

验收：

- `npm install` 可执行。
- `npm run build` 至少能跑到缺少实现前的明确错误，或在占位实现下通过。
- package exports 清晰。

### Workstream B：Runtime 与校验

可并行性：可在 A 的目录结构确定后独立开发，不依赖 Vue 组件。

写入范围：

- `d_page/src/runtime/*`
- `d_page/src/registry/createActionRegistry.js`
- `d_page/src/registry/defaultActions.js`
- `d_page/tests/resolveBinding.spec.js`
- `d_page/tests/validateSchema.spec.js`
- `d_page/tests/runtimeActions.spec.js`

任务：

1. 实现 `createDPageRuntime`。
2. 实现 schema normalize/validate。
3. 实现安全 binding resolver。
4. 实现 action registry 和默认 actions。
5. 单测覆盖核心纯函数。

验收：

- `npm run test -- resolveBinding` 通过。
- `npm run test -- validateSchema` 通过。
- 不引入浏览器专属依赖。

### Workstream C：Vue 渲染器

可并行性：依赖 A 的入口约定，部分依赖 B 的 runtime API；可以先以 mock runtime 开发。

写入范围：

- `d_page/src/components/DPageRenderer.vue`
- `d_page/src/components/DCardRenderer.vue`
- `d_page/src/components/DUnknownComponent.vue`
- `d_page/src/components/DRenderError.vue`
- `d_page/tests/renderer.spec.js`（如测试环境支持 Vue mount）

任务：

1. 实现根 schema 渲染。
2. 实现 card 递归渲染。
3. 支持 `children` 和 `slots`。
4. 接入 component registry。
5. 接入错误降级。

验收：

- 合法 schema 能渲染基础组件。
- 未知组件显示 `DUnknownComponent`。
- 单卡片错误显示 `DRenderError`。

### Workstream D：基础组件与组件注册表

可并行性：可在 A 后独立开发；与 C 只通过 registry 对接。

写入范围：

- `d_page/src/components/base/*`
- `d_page/src/registry/createComponentRegistry.js`
- `d_page/src/registry/defaultComponents.js`
- `d_page/tests/componentRegistry.spec.js`

任务：

1. 实现基础展示组件：text、heading、cardSurface。
2. 实现操作和输入组件：button、input。
3. 实现基础 table 空状态。
4. 实现 component registry 的 register/get/has/list。
5. 定义组件事件名称约定。

验收：

- registry 可注册、覆盖、读取组件。
- 基础组件空 props 不崩溃。
- 默认组件列表包含 MVP 组件。

### Workstream E：Examples 与包文档

可并行性：可与 B/C/D 并行，但最终需要按实现调整示例字段。

写入范围：

- `d_page/src/schemas/examples/*.json`
- `d_page/README.md`
- `d_page/docs/plans/*`

任务：

1. 编写 `chat-tool-result.json`。
2. 编写 `simple-form.json`。
3. 编写 `table-card.json`。
4. 在 README 中说明安装、构建、基础使用和 chat 预接入。

验收：

- 示例 schema 满足 validateSchema。
- README 可以指导外部 Vue 应用 import 包。

### Workstream F：chat 预接入设计与保护边界

可并行性：文档和设计可并行；代码接入必须等 A-E 验收后再做。

写入范围：

- `d_page/docs/plans/*chat*.md`
- 可选后续：`desktop/frontend/src/services/d_page/*`
- 可选后续：`desktop/frontend/src/components/chat/ChatMessageItem.vue`

任务：

1. 明确 chat message metadata 协议。
2. 明确 feature flag 策略。
3. 明确宿主 action 注入列表。
4. 明确失败降级：schema 无效时回落普通文本内容。

验收：

- 本阶段不默认修改 chat 主流程。
- 后续接入有清晰文件范围和回滚路径。

## 3. 推荐并行顺序

### 当前推荐任务分配（从现在开始）

| 并行角色 | 当前任务 | 文件范围 | 交付物 | 前置依赖 |
| --- | --- | --- | --- | --- |
| 主线集成 | demo 预览和真实交互验证 | `d_page/src/demo/*`、`d_page/tests/*` | 三个示例 schema 可浏览器预览，交互路径已手动验证，后续补自动化 | 当前 internal alpha |
| Runtime 补强 | action 与 binding 边界增强 | `d_page/src/runtime/*`、`d_page/tests/*` | 错误路径、生命周期、事件上下文覆盖更完整 | 当前 runtime 雏形 |
| 组件扩展 | MVP 组件继续补齐 | `d_page/src/components/base/*`、`d_page/src/registry/defaultComponents.js` | select、textarea、alert/stat/list 等候选组件 | 当前组件注册表 |
| 文档与可视化 | 使用手册和状态图维护 | `d_page/docs/assets/*`、`d_page/docs/plans/*`、`d_page/README.md` | 完成度图、API 说明、下一步任务分配持续同步 | 当前代码状态快照 |
| chat 设计 | 预接入协议细化 | `d_page/docs/plans/*chat*.md` 或现有计划文档 | metadata 协议、feature flag、宿主 action 清单、回退策略 | renderer 示例通过验收后进入 |

### 3.1 Demo 验收记录

本地 demo 地址：`http://127.0.0.1:9360`，启动命令：`npm run demo`。

已在内置浏览器完成以下手动验证：

- 表格卡片：点击“运行时 基本完成 90%”行后，预览区显示“已选择：运行时”，右侧 runtime state 更新为 `{ "selectedName": "运行时" }`。
- 基础表单：输入 `Ada` 后，预览区显示“当前输入：Ada”；点击“提交”后显示“已提交：Ada”，右侧 state 更新为 `{ "name": "Ada", "submittedName": "Ada" }`。
- Chat 结果卡片：点击“复制摘要”后，预览区显示“已复制”，右侧显示最近复制内容；点击“打开详情”后，右侧事件日志出现 `emit: openDetails`。
- 即时输入预览：输入标题 `实时联动成功`、填写说明 `这段文字刚刚输入，右侧预览会同步显示。`、选择“危险”状态后，右侧 preview alert 和 runtime state 立即同步为对应值。

并行原则：主线集成和 Runtime 补强可以同时推进；示例与说明先按既有协议起草，等 renderer 字段稳定后再校准；chat 设计只做文档，不改 chat 主流程。

### 第 0 轮：准备

由主线完成：

1. 确认 `d_page` 包名和目标 API。
2. 合并当前文档变更。
3. 决定是否将 `d_page` 加入 root workspace 或先保持独立 npm 包。

### 第 1 轮：三路并行

- Worker A：包骨架与公共 API。
- Worker B：Runtime 与校验。
- Worker D：基础组件与组件注册表。

理由：A 建目录，B 和 D 写不同目录，冲突小。

### 第 2 轮：两路并行

- Worker C：Vue 渲染器，集成 B/D。
- Worker E：Examples 与 README，根据 B/D 的最终字段微调。

### 第 3 轮：主线集成

1. 跑 `npm run test`。
2. 跑 `npm run build`。
3. 用示例 schema 做 demo preview。
4. 检查包导出。
5. 写 chat 预接入计划，不默认改 chat 主流程。

## 4. 文件冲突控制

为避免并行开发冲突：

- `src/index.js` 只由 Workstream A 和主线集成修改。
- `runtime/*` 只由 Workstream B 修改。
- `components/D*.vue` 只由 Workstream C 修改。
- `components/base/*` 和 `registry/defaultComponents.js` 只由 Workstream D 修改。
- `schemas/examples/*` 和 README 示例段落由 Workstream E 修改。
- `desktop/frontend` 默认不在本阶段修改。

## 5. 最小验收命令

在 `d_page` 目录执行：

```powershell
npm install
npm run test
npm run build
```

如果新增 demo preview：

```powershell
npm run dev
```

## 6. Definition of Done

- `@icoo-claw/d-page` 包骨架存在。
- 包可以独立安装依赖、测试、构建。
- `DPageRenderer` 可以渲染 `chat-tool-result.json`。
- 未知组件和非法 schema 有降级 UI。
- `resolveBinding` 不执行任意 JavaScript。
- 默认 action 不包含任意网络请求能力。
- README 说明 npm/file 依赖使用方式。
- chat 接入方案清晰，但默认不接入主流程。

## 6.1 当前 DoD 差距

| 验收项 | 当前状态 | 差距 |
| --- | --- | --- |
| 包骨架存在 | 已达成 | 后续确认发布 files 是否需要包含 examples |
| 独立测试 | 已达成 | `npm run test` 已通过，6 个测试文件 25 个用例 |
| 独立构建 | 已达成 | `npm run build` 已通过并产出 ESM/CJS/CSS |
| 示例 schema 可渲染 | 已基本达成 | 三份示例通过 schema 校验，renderer SSR 烟测和 demo 手动验证通过 |
| 未知组件和非法 schema 降级 | 已基本达成 | 未知组件 SSR 烟测通过，非法 schema 走页面级错误 |
| 安全 binding | 已基本达成 | 后续继续补充异常路径测试即可 |
| 默认 action 不含任意网络请求 | 已基本达成 | request 继续保持由宿主注入，不作为默认 action |
| README 使用说明 | 已基本达成 | 已补本地 file 依赖、最小 schema、API 和宿主 action 注入示例 |
| chat 接入方案 | 设计中 | 当前停留在 metadata 方向，下一阶段细化 feature flag 和回退策略 |
| demo 预览 | 已基本达成 | `npm run demo` 可运行，四份示例可切换并完成手动交互验证 |

## 7. 暂不做事项

- 不做可视化编辑器。
- 不做远程 schema 后台。
- 不做公网安全沙箱承诺。
- 不做 TypeScript 迁移。
- 不把 `d_page` 直接耦合到 desktop 的 QQ UED 组件或 Pinia stores。
- 不默认让 chat 渲染 d_page 消息，除非进入后续 `0.2.0 chat preview` 阶段。
