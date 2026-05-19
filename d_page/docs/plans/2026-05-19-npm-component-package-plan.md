# d_page npm 组件包改造方案

日期：2026-05-19  
适用范围：`d_page` 动态页面渲染内核  
目标形态：可发布、可版本化、可被 `desktop/frontend` chat 消费的 Vue npm 组件包

## 1. 当前状态分析

`d_page` 目前只有两类设计文档：

- `docs/requirements/2026-05-19-dynamic-page-system-requirements.md`：描述动态页面系统需求。
- `docs/standards/2026-05-19-card-rendering-standard.md`：定义卡片树渲染协议。

当前尚未存在源码、构建配置、包入口、示例、测试或发布流程。因此下一步不应直接把 `d_page` 做成完整应用，而应先把它建设为一个稳定的 Vue 渲染组件包。

## 2. 新定位

`d_page` 不再优先定位为独立 Vite 单页应用，而是定位为：

> 一个 JSON/Card Schema 驱动的 Vue 3 动态页面渲染组件包，提供渲染器、注册表、基础组件、动作执行器和 schema 校验能力。成熟后由 chat 通过 npm 依赖引入，用于渲染消息里的动态卡片、表单、工具结果和可交互页面片段。

这种定位有几个好处：

- `chat` 不需要复制动态渲染逻辑。
- `d_page` 可以独立测试、独立版本化、独立发布。
- 后续也可以被设置页、插件页、自动化页等其他桌面模块复用。
- 先收敛到组件包，避免过早建设低代码编辑器和完整配置平台。

## 3. 包名与发布形态

建议包名先使用 workspace 私有作用域：

```text
@icoo-claw/d-page
```

初期可以不发布到公网 npm，先采用本地 workspace/file 依赖：

```json
{
  "dependencies": {
    "@icoo-claw/d-page": "file:../../d_page"
  }
}
```

成熟后再切换为 npm registry 或私有 registry：

```json
{
  "dependencies": {
    "@icoo-claw/d-page": "^0.1.0"
  }
}
```

## 4. 目标目录结构

建议将 `d_page` 改造成标准包目录：

```text
d_page/
  package.json
  vite.config.js
  README.md
  src/
    index.js
    style.css
    components/
      DPageRenderer.vue
      DCardRenderer.vue
      DUnknownComponent.vue
      DRenderError.vue
      base/
        DText.vue
        DHeading.vue
        DButton.vue
        DInput.vue
        DCardSurface.vue
        DTable.vue
    registry/
      createComponentRegistry.js
      createActionRegistry.js
      defaultComponents.js
      defaultActions.js
    runtime/
      createDPageRuntime.js
      resolveBinding.js
      executeAction.js
      normalizeSchema.js
      validateSchema.js
    schemas/
      examples/
        chat-tool-result.json
        simple-form.json
        table-card.json
  tests/
    resolveBinding.spec.js
    validateSchema.spec.js
    renderer.spec.js
  docs/
    requirements/
    standards/
    plans/
```

## 5. 包对外 API

`src/index.js` 应只导出稳定 API，不暴露内部文件结构。

建议第一阶段导出：

```js
export { default as DPageRenderer } from './components/DPageRenderer.vue'
export { default as DCardRenderer } from './components/DCardRenderer.vue'

export { createDPageRuntime } from './runtime/createDPageRuntime'
export { createComponentRegistry } from './registry/createComponentRegistry'
export { createActionRegistry } from './registry/createActionRegistry'
export { defaultComponents } from './registry/defaultComponents'
export { defaultActions } from './registry/defaultActions'

export { normalizeSchema } from './runtime/normalizeSchema'
export { validateSchema } from './runtime/validateSchema'
export { resolveBinding } from './runtime/resolveBinding'
```

包使用方只需要：

```js
import { DPageRenderer, createDPageRuntime } from '@icoo-claw/d-page'
import '@icoo-claw/d-page/style.css'
```

## 6. 组件核心协议

包协议继续沿用“卡片树渲染”：

- `root` 是唯一渲染入口。
- 所有可渲染节点都是 `type: "card"`。
- `component.type` 通过组件注册表解析。
- 交互事件通过 action registry 执行。
- 表达式只支持安全路径读取和少量白名单能力。

第一阶段必须避免支持任意 JavaScript 表达式，避免给 chat 引入安全风险。

## 7. 第一阶段 MVP

MVP 只做能被 chat 试用的最小能力。

### 7.1 渲染能力

- `DPageRenderer`：接收 `schema`、`runtime`、`context`。
- `DCardRenderer`：递归渲染 `root`、`children`、`slots`。
- 未知组件降级显示。
- 单卡片错误边界。

### 7.2 内置组件

- `text`
- `heading`
- `button`
- `input`
- `cardSurface`
- `table`

### 7.3 动作能力

- `setState`
- `emit`
- `openUrl`
- `copyText`
- `chain`

MVP 阶段不直接内置 `request`，因为 chat 场景里网络请求应该先由宿主应用掌控，动态卡片只发出事件或使用宿主注入的安全 action。

### 7.4 状态能力

- 页面初始 `state`。
- 组件 props 中读取 `{{ state.xxx }}`、`{{ data.xxx }}`、`{{ context.xxx }}`。
- `setState` 后局部响应式更新。

## 8. chat 接入方式

成熟后，chat 不直接解析 d_page schema，而是把动态消息交给 `DPageRenderer`。

### 8.1 消息结构建议

后端或 agent 输出消息可以在 metadata 中携带动态页面 schema：

```json
{
  "role": "assistant",
  "content": "已生成一个配置表单。",
  "metadata": {
    "render_type": "d_page",
    "d_page_schema": {
      "schemaVersion": "0.1.0",
      "root": {
        "id": "toolResultCard",
        "type": "card",
        "kind": "display",
        "component": {
          "type": "text",
          "props": {
            "text": "{{ data.summary }}"
          }
        }
      },
      "data": {
        "summary": "执行完成"
      }
    }
  }
}
```

### 8.2 chat 前端渲染策略

`ChatMessageItem.vue` 后续可以按 metadata 分流：

```vue
<DPageRenderer
  v-if="message.metadata?.render_type === 'd_page'"
  :schema="message.metadata.d_page_schema"
  :runtime="dPageRuntime"
  :context="{ conversationId, messageId: message.id }"
/>
<pre v-else>{{ message.content }}</pre>
```

### 8.3 宿主 action 注入

chat 应注入自己的 action，而不是让 d_page 直接控制宿主：

- `sendChatPrompt`
- `openConversation`
- `copyToComposer`
- `saveArtifact`
- `openExternalUrl`

这样 d_page 包保持通用，chat 保持权限边界清晰。

## 9. 构建方案

使用 Vite library mode：

```js
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  build: {
    lib: {
      entry: 'src/index.js',
      name: 'DPage',
      formats: ['es', 'cjs'],
      fileName: (format) => `d-page.${format}.js`,
    },
    rollupOptions: {
      external: ['vue', 'pinia'],
    },
  },
})
```

`package.json` 建议：

```json
{
  "name": "@icoo-claw/d-page",
  "version": "0.1.0",
  "type": "module",
  "private": true,
  "main": "dist/d-page.cjs.js",
  "module": "dist/d-page.es.js",
  "exports": {
    ".": {
      "import": "./dist/d-page.es.js",
      "require": "./dist/d-page.cjs.js"
    },
    "./style.css": "./dist/style.css"
  },
  "files": ["dist"],
  "peerDependencies": {
    "vue": "^3.5.0",
    "pinia": "^3.0.0"
  },
  "scripts": {
    "dev": "vite --host 127.0.0.1 --port 9360",
    "build": "vite build",
    "test": "vitest run",
    "preview": "vite preview --host 127.0.0.1 --port 9361"
  }
}
```

## 10. 版本成熟度分级

### 0.1.0 internal alpha

- 渲染卡片树。
- 支持基础组件。
- 支持基础状态和动作。
- 可在 story/demo 中预览。
- 不接入 chat 主流程。

### 0.2.0 chat preview

- chat 可以通过 feature flag 渲染 `render_type = d_page` 的消息。
- 支持宿主 action 注入。
- 有基本安全校验和错误边界。
- UI smoke 覆盖普通文本消息和 d_page 消息共存。

### 0.3.0 reusable package

- API 稳定。
- schema 版本迁移机制。
- 更完整表单和表格。
- 文档示例完整。
- 可发布到内部 npm registry。

### 1.0.0 stable

- 已在 chat、设置页或其他模块至少两个场景稳定使用。
- 有兼容性策略。
- 有 schema 版本说明。
- 有明确安全边界和扩展指南。

## 11. 推荐实施步骤

1. 在 `d_page` 下初始化 npm 包骨架。
2. 实现 `DPageRenderer`、`DCardRenderer`、未知组件和错误组件。
3. 实现 registry 和最小基础组件。
4. 实现 `resolveBinding`、`setState`、`emit`。
5. 增加 `chat-tool-result.json` 示例，模拟 chat 动态消息。
6. 跑 `npm run test` 和 `npm run build`。
7. 在 `desktop/frontend` 以 `file:../../d_page` 方式试接入，但先放在实验路由或 feature flag 后面。
8. 成熟后再接入 `ChatMessageItem.vue` 的 metadata 渲染分支。

## 12. 本阶段不做

- 不做可视化编辑器。
- 不做远程 schema 管理平台。
- 不做公网安全沙箱承诺。
- 不让动态 schema 执行任意 JavaScript。
- 不直接替换 chat 现有消息渲染。
- 不让 d_page 包直接依赖 desktop 的 Pinia stores 或 QQ UED 组件。

## 13. 验收标准

- `d_page` 可以独立 `npm run test`。
- `d_page` 可以独立 `npm run build` 并产出 library dist。
- 一个示例 schema 可以渲染卡片、按钮和基础状态更新。
- 未知组件和错误 schema 不会白屏。
- 包导出的 API 可以被外部 Vue 应用 import。
- chat 接入方案明确，但默认不开启 chat 动态渲染。

