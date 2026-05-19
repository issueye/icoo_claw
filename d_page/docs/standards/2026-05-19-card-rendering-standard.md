# 卡片化动态渲染标准

日期：2026-05-19  
适用项目：d_page  
技术栈：Vite + Vue 3 + JavaScript + Tailwind CSS + Pinia

## 1. 标准目标

本标准定义动态页面系统的核心渲染规则：以“卡片 Card”作为最小动态渲染单元。页面、区域、表单、表格、详情、统计块、弹窗内容、业务组件都应通过卡片协议描述和渲染。

标准目标包括：

- 统一页面 JSON 的结构表达。
- 降低渲染器复杂度。
- 让页面由可组合、可嵌套、可扩展的卡片组成。
- 支持卡片内承载表单、表格、图表、列表、自定义业务组件。
- 支持页面本身作为一种特殊卡片渲染。
- 支持通过注册机制扩展动态组件。

## 2. 核心原则

### 2.1 卡片是最小动态渲染单元

系统不直接渲染任意散乱组件树，而是渲染卡片树。每一个可配置渲染块都必须是一个 `card` 节点。

卡片可以很小，例如一个统计数字；也可以很大，例如一个完整页面。

### 2.2 页面是特殊卡片

页面不再单独维护一套完全不同的结构。页面本质上是 `type = page` 或 `kind = page` 的卡片。

这样可以保证：

- 页面和普通卡片使用同一套 schema。
- 页面可以嵌套页面片段。
- 弹窗、抽屉、标签页里的内容也可以复用页面级配置。

### 2.3 卡片负责结构，组件负责能力

卡片定义布局、标题、区域、数据上下文、动作和子节点。具体能力由卡片内的 `component` 承载，例如表单、表格、图表、自定义 Vue 组件。

### 2.4 卡片可以嵌套卡片

卡片通过 `children` 或具名 `slots` 组合其他卡片。布局类卡片负责组织子卡片，功能类卡片负责承载具体能力。

### 2.5 扩展通过注册完成

系统不应在渲染器里硬编码所有业务组件。新增动态组件时，应通过组件注册表声明：

- 组件类型。
- Vue 组件实现。
- props 解析规则。
- 默认配置。
- 支持的事件。
- 可选校验规则。

## 3. 标准术语

### 3.1 Card

卡片，动态渲染的最小单元。每个卡片都有唯一 `id`、渲染类型、布局配置、数据绑定、事件动作和子卡片。

### 3.2 Card Tree

卡片树，由一个根卡片和多个子卡片组成。页面渲染时从根卡片开始递归渲染。

### 3.3 Component

组件，卡片内部承载的实际 Vue 能力，例如 `form`、`table`、`chart`、`customUserPanel`。

### 3.4 Slot

具名插槽，用于描述卡片内部的固定区域，例如 `header`、`toolbar`、`body`、`footer`、`actions`。

### 3.5 Action

动作，用户交互或生命周期触发的执行逻辑，例如请求接口、更新状态、打开弹窗、提交表单。

### 3.6 Context

上下文，卡片渲染和动作执行时可访问的数据，包括页面状态、卡片状态、接口数据、父级上下文和路由参数。

## 4. 卡片 Schema 标准

### 4.1 最小卡片结构

每个卡片节点建议使用以下结构：

```json
{
  "id": "userTableCard",
  "type": "card",
  "kind": "data",
  "title": "用户列表",
  "component": {
    "type": "table",
    "props": {}
  },
  "children": []
}
```

### 4.2 完整卡片结构

```json
{
  "id": "string",
  "type": "card",
  "kind": "page | layout | form | data | display | action | custom",
  "title": "string",
  "description": "string",
  "visible": true,
  "disabled": false,
  "layout": {
    "mode": "block | grid | flex | tabs | steps",
    "columns": 1,
    "gap": "md",
    "class": "string"
  },
  "data": {
    "source": "string",
    "bind": "string",
    "defaultValue": {}
  },
  "state": {},
  "component": {
    "type": "string",
    "props": {},
    "events": {}
  },
  "slots": {
    "header": [],
    "toolbar": [],
    "body": [],
    "footer": [],
    "actions": []
  },
  "children": [],
  "events": {},
  "lifecycle": {
    "onInit": [],
    "onMounted": [],
    "onVisible": [],
    "onDestroy": []
  },
  "permissions": [],
  "metadata": {}
}
```

### 4.3 字段说明

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `id` | 是 | 卡片唯一标识，页面内不可重复。 |
| `type` | 是 | 固定为 `card`，用于统一渲染入口。 |
| `kind` | 是 | 卡片语义类型，用于默认样式、校验和编辑器分类。 |
| `title` | 否 | 卡片标题。 |
| `description` | 否 | 卡片辅助说明。 |
| `visible` | 否 | 是否显示，支持布尔值或绑定表达式。 |
| `disabled` | 否 | 是否禁用，支持布尔值或绑定表达式。 |
| `layout` | 否 | 卡片内部布局配置。 |
| `data` | 否 | 卡片数据来源和绑定配置。 |
| `state` | 否 | 卡片局部状态默认值。 |
| `component` | 否 | 卡片承载的动态组件。 |
| `slots` | 否 | 具名区域，每个区域都是卡片数组。 |
| `children` | 否 | 默认内容区子卡片。 |
| `events` | 否 | 卡片级事件。 |
| `lifecycle` | 否 | 卡片生命周期动作。 |
| `permissions` | 否 | 权限标识。 |
| `metadata` | 否 | 非渲染元信息。 |

## 5. 页面 Schema 标准

页面 schema 顶层仍可保留全局信息，但渲染入口必须是一个根卡片。

```json
{
  "schemaVersion": "1.0.0",
  "page": {
    "id": "userManagePage",
    "title": "用户管理"
  },
  "state": {
    "keyword": "",
    "users": []
  },
  "actions": {},
  "root": {
    "id": "rootPageCard",
    "type": "card",
    "kind": "page",
    "layout": {
      "mode": "grid",
      "columns": 12,
      "gap": "md",
      "class": "p-6"
    },
    "children": []
  }
}
```

标准要求：

- `root` 是唯一页面渲染入口。
- `root.type` 必须为 `card`。
- `root.kind` 建议为 `page`。
- 页面内所有可渲染内容必须挂在 `root.children` 或 `root.slots` 下。

## 6. 卡片类型标准

### 6.1 page 卡片

页面级卡片，作为页面根节点或可复用页面片段。

适用场景：

- 完整页面。
- 子页面。
- 弹窗内页面。
- 抽屉内页面。

### 6.2 layout 卡片

布局卡片，只负责排列子卡片，不承载复杂业务能力。

常见布局：

- `block`：上下排列。
- `grid`：网格布局。
- `flex`：弹性布局。
- `tabs`：标签页布局。
- `steps`：步骤布局。

### 6.3 form 卡片

表单卡片，内部 `component.type` 通常为 `form`。

表单字段可以作为 `component.props.fields` 配置，也可以拆成子卡片。MVP 阶段建议字段先配置在 `fields` 中，复杂表单再扩展字段卡片模式。

### 6.4 data 卡片

数据卡片，承载表格、列表、统计、图表等数据展示组件。

常见组件：

- `table`
- `list`
- `stat`
- `chart`
- `description`

### 6.5 display 卡片

展示卡片，承载文本、图片、说明、分割线、提示块等静态或半静态内容。

### 6.6 action 卡片

操作卡片，承载按钮组、工具栏、筛选栏、批量操作区。

### 6.7 custom 卡片

自定义卡片，承载通过注册机制接入的业务组件。

## 7. 卡片承载表单标准

表单卡片示例：

```json
{
  "id": "searchFormCard",
  "type": "card",
  "kind": "form",
  "title": "筛选条件",
  "component": {
    "type": "form",
    "props": {
      "model": "searchForm",
      "layout": "inline",
      "fields": [
        {
          "name": "keyword",
          "label": "关键词",
          "type": "input",
          "placeholder": "请输入姓名或手机号",
          "bind": "state.keyword"
        },
        {
          "name": "status",
          "label": "状态",
          "type": "select",
          "options": [
            { "label": "全部", "value": "" },
            { "label": "启用", "value": "enabled" },
            { "label": "停用", "value": "disabled" }
          ],
          "bind": "state.status"
        }
      ]
    },
    "events": {
      "submit": "queryUsers",
      "reset": "resetSearchForm"
    }
  }
}
```

标准要求：

- 表单整体必须放在卡片中。
- 表单字段可配置化。
- 字段值必须绑定到 Pinia 页面状态或卡片局部状态。
- 表单提交、重置、校验通过动作系统完成。

## 8. 卡片承载表格标准

表格卡片示例：

```json
{
  "id": "userTableCard",
  "type": "card",
  "kind": "data",
  "title": "用户列表",
  "data": {
    "bind": "state.users"
  },
  "component": {
    "type": "table",
    "props": {
      "rowKey": "id",
      "columns": [
        { "key": "name", "title": "姓名" },
        { "key": "phone", "title": "手机号" },
        { "key": "status", "title": "状态" }
      ],
      "pagination": {
        "page": "state.page",
        "pageSize": "state.pageSize",
        "total": "state.total"
      }
    },
    "events": {
      "pageChange": "queryUsers",
      "rowClick": "openUserDetail"
    }
  },
  "slots": {
    "toolbar": [
      {
        "id": "createUserButtonCard",
        "type": "card",
        "kind": "action",
        "component": {
          "type": "button",
          "props": {
            "text": "新增用户",
            "variant": "primary"
          },
          "events": {
            "click": "openCreateUserModal"
          }
        }
      }
    ]
  }
}
```

标准要求：

- 表格整体必须是一个 data 卡片。
- 表格工具栏、批量操作、空状态等应通过 `slots` 扩展。
- 表格数据来源必须通过 `data.bind` 或组件 props 绑定声明。
- 分页、排序、筛选事件必须通过动作系统触发。

## 9. 卡片作为页面标准

完整页面示例：

```json
{
  "schemaVersion": "1.0.0",
  "page": {
    "id": "userManagePage",
    "title": "用户管理"
  },
  "state": {
    "keyword": "",
    "status": "",
    "users": [],
    "page": 1,
    "pageSize": 10,
    "total": 0
  },
  "actions": {
    "queryUsers": {
      "type": "request",
      "payload": {
        "url": "/api/users",
        "method": "GET",
        "params": {
          "keyword": "{{ state.keyword }}",
          "status": "{{ state.status }}",
          "page": "{{ state.page }}",
          "pageSize": "{{ state.pageSize }}"
        },
        "saveTo": {
          "users": "data.list",
          "total": "data.total"
        }
      }
    }
  },
  "root": {
    "id": "rootPageCard",
    "type": "card",
    "kind": "page",
    "layout": {
      "mode": "block",
      "gap": "md",
      "class": "p-6 bg-slate-50 min-h-screen"
    },
    "children": [
      {
        "id": "pageHeaderCard",
        "type": "card",
        "kind": "display",
        "component": {
          "type": "heading",
          "props": {
            "text": "用户管理",
            "level": 1
          }
        }
      },
      {
        "id": "searchFormCard",
        "type": "card",
        "kind": "form",
        "component": {
          "type": "form",
          "props": {
            "layout": "inline",
            "fields": [
              {
                "name": "keyword",
                "label": "关键词",
                "type": "input",
                "bind": "state.keyword"
              }
            ]
          },
          "events": {
            "submit": "queryUsers"
          }
        }
      },
      {
        "id": "userTableCard",
        "type": "card",
        "kind": "data",
        "data": {
          "bind": "state.users"
        },
        "component": {
          "type": "table",
          "props": {
            "rowKey": "id",
            "columns": [
              { "key": "name", "title": "姓名" },
              { "key": "phone", "title": "手机号" }
            ]
          }
        }
      }
    ]
  }
}
```

## 10. 动态组件扩展标准

### 10.1 组件注册结构

每个动态组件必须注册到组件注册表。

```js
export const componentRegistry = {
  table: {
    name: 'table',
    component: DTable,
    category: 'data',
    defaultProps: {},
    events: ['rowClick', 'pageChange', 'sortChange'],
    validate: validateTableProps
  },
  userProfile: {
    name: 'userProfile',
    component: UserProfileCard,
    category: 'custom',
    defaultProps: {},
    events: ['edit', 'remove'],
    validate: validateUserProfileProps
  }
}
```

### 10.2 扩展组件约束

动态组件必须遵守以下约束：

- 不直接读取原始 schema，接收解析后的 props。
- 不直接修改全局状态，通过事件和动作系统提交变更。
- 组件事件必须在注册信息中声明。
- 组件必须有默认空状态，不能因数据为空导致崩溃。
- 组件内部错误应被卡片级错误边界捕获。

### 10.3 自定义组件卡片示例

```json
{
  "id": "userProfileCard",
  "type": "card",
  "kind": "custom",
  "title": "用户画像",
  "component": {
    "type": "userProfile",
    "props": {
      "userId": "{{ state.selectedUserId }}",
      "showTags": true
    },
    "events": {
      "edit": "openEditUserModal",
      "remove": "removeUser"
    }
  }
}
```

## 11. 渲染流程标准

渲染器必须按以下流程执行：

1. 加载页面 schema。
2. 校验顶层 schema 和 `root` 卡片。
3. 初始化 Pinia 页面状态。
4. 初始化卡片局部状态。
5. 从 `root` 开始递归渲染卡片树。
6. 解析卡片 `visible`、`disabled`、`data`、`component.props` 中的绑定表达式。
7. 根据 `component.type` 从组件注册表获取 Vue 组件。
8. 合并组件默认 props 和 JSON props。
9. 绑定组件事件到动作系统。
10. 渲染 `slots` 和 `children`。
11. 执行生命周期动作。
12. 捕获并展示卡片级错误。

## 12. 状态与上下文标准

卡片渲染时可以访问以下上下文：

```js
{
  page: {},
  state: {},
  data: {},
  card: {},
  parent: {},
  route: {},
  request: {}
}
```

绑定表达式优先支持路径读取：

- `{{ state.keyword }}`
- `{{ data.users }}`
- `{{ card.state.expanded }}`
- `{{ route.params.id }}`

标准限制：

- 不允许在 JSON 中执行任意 JavaScript。
- 不允许通过表达式访问浏览器全局对象。
- 后续如需函数能力，只能使用白名单函数。

## 13. 卡片事件标准

卡片级事件和组件级事件都应映射到动作。

```json
{
  "events": {
    "click": "openDetail",
    "refresh": ["queryUsers", "showRefreshToast"]
  }
}
```

事件值支持：

- 字符串：单个动作 id。
- 字符串数组：按顺序执行多个动作。
- 内联动作对象：仅用于简单局部动作，复杂动作应放入全局 `actions`。

## 14. 布局标准

卡片布局由 `layout` 描述。

```json
{
  "layout": {
    "mode": "grid",
    "columns": 12,
    "span": 6,
    "gap": "md",
    "align": "start",
    "class": "bg-white rounded-lg border"
  }
}
```

标准字段：

- `mode`：布局模式。
- `columns`：网格列数。
- `span`：当前卡片跨列数。
- `gap`：间距级别，建议 `xs | sm | md | lg | xl`。
- `align`：对齐方式。
- `class`：Tailwind 扩展类。

## 15. 校验标准

必须校验以下内容：

- `root` 是否存在。
- 所有渲染节点 `type` 是否为 `card`。
- 每个卡片 `id` 是否唯一。
- `kind` 是否在允许范围内。
- `component.type` 是否已注册。
- `events` 引用的动作是否存在。
- `slots` 和 `children` 是否为卡片数组。
- 表单字段配置是否合法。
- 表格列配置是否合法。

## 16. 错误降级标准

错误必须控制在卡片范围内。

- 页面 schema 无法加载：显示页面级错误。
- 根卡片非法：显示页面级错误。
- 单个卡片配置非法：显示卡片错误占位。
- 组件未注册：显示未知组件占位。
- 动作执行失败：显示交互错误提示。
- 数据请求失败：保留卡片结构，显示失败状态和重试入口。

## 17. MVP 建议实现范围

第一阶段建议实现以下卡片能力：

- `page` 卡片。
- `layout` 卡片：block、grid。
- `display` 卡片：heading、text、cardContent。
- `form` 卡片：input、select、submit、reset。
- `data` 卡片：table。
- `action` 卡片：button。
- `custom` 卡片注册机制。
- 卡片级 `children` 递归渲染。
- 卡片级 `slots.header`、`slots.toolbar`、`slots.footer`。
- Pinia 页面状态。
- 简单绑定表达式。
- setState、request、showToast、openModal、closeModal 动作。

## 18. 推荐结论

建议将系统协议从“组件树渲染”调整为“卡片树渲染”。所有页面内容统一挂载在根卡片下，卡片作为最小动态渲染边界，内部通过 `component.type` 承载表单、表格、自定义组件等能力。

这种方式能让渲染器保持稳定，同时给业务扩展留出清晰入口：新增业务能力时扩展组件注册表，不修改卡片渲染主流程。

