# Wails3 Technical Spike

## 目标

在正式启动 `desktop/` 客户端开发前，先验证 `Wails3` 在当前仓库、当前 Windows 环境和当前技术栈组合下是否稳定可用。

本次预研只回答两个问题：

1. `Wails3 + Vue3 + Vite + JavaScript` 能否在本仓库顺利跑通开发和构建链路。
2. `Wails3` 是否能满足桌面端当前最关键的本地能力需求。

## 验证范围

### A. 基础工程能力

- 能否通过 `wails3 init` 或等价方式快速生成可运行项目。
- 能否在仓库内以独立模块形式接入，不污染现有 `server/*`。
- 能否通过 `wails3 dev` 启动开发模式。
- 能否通过 `wails3 build` 产出 Windows 可执行文件。

### B. 前后端桥接能力

- Go service 是否能成功注册。
- 前端是否能使用自动生成的 bindings 调用 Go 方法。
- 修改 Go 方法签名后，bindings 是否能稳定再生成。
- Promise 异常是否能在前端正确捕获。

### C. 本地系统能力

- 目录选择器是否可用。
- 是否能读取用户配置目录。
- 是否能完成本地 TOML 文件读写。
- 是否能获取应用版本和运行信息。

### D. 前端开发体验

- Vue3 + Vite 模板是否可直接使用。
- 前端改动是否能热更新。
- Go 改动是否能触发重编译和应用重启。
- Tailwind CSS 接入是否顺畅。

### E. Windows 交付风险

- 首次构建依赖是否完整。
- WebView2 或 Windows 运行时是否存在额外前置条件。
- 构建输出路径、图标、配置文件结构是否符合后续正式项目预期。

## 验证产出

预研结束后必须给出：

- 结论：可用 / 可用但有约束 / 不建议使用
- 使用的 `Wails3` 版本
- 最小可运行目录结构
- 必装依赖与安装顺序
- Windows 环境注意事项
- 是否建议直接复用 spike 工程为正式 `desktop/` 工程

## 通过标准

满足以下条件视为通过：

- `wails3 dev` 可正常运行。
- `wails3 build` 可正常产出 Windows 可执行文件。
- Go 方法绑定可从前端成功调用。
- 目录选择器可正常选择文件夹。
- Vue3 + Vite + Tailwind CSS 可正常集成。

## 当前预研结论

截至当前验证，结论为：`可用，但有两个需要显式规避的环境约束`。

已确认通过：

- `wails3 doctor` 检查通过。
- `wails3 init -t vue` 可正常生成项目。
- `wails3 generate bindings` 可正常扫描并生成 Go service bindings。
- `npm run build` 可正常产出前端静态资源。
- `wails3 build` 可正常产出 Windows 可执行文件。
- Go service 调用与自定义事件注册链路可编译通过。

已发现约束：

1. 当前仓库 `go.work` 中包含 `./go_pkg/redka`，但该目录下不存在 `go.mod`，会导致子模块执行 `go test`、`wails3 generate bindings` 等命令时失败。
2. 当前机器上的 `9245` 端口已被现有 `node` 进程占用，直接执行默认 `wails3 dev` 容易导致前端 dev server 启动失败，桌面窗口表现为 `wails.localhost HTTP ERROR 502`。

## 当前建议

- 预研和后续桌面模块开发阶段，优先把桌面工程作为独立模块运行，必要时临时使用 `GOWORK=off` 隔离当前 workspace 问题。
- 开发模式优先使用显式端口，例如：
  - `wails3 dev -port 9345`
  - 或通过 `task dev` 使用已调整后的默认端口。
- 正式进入 `desktop/` 工程前，需要决定：
  - 修复根仓库 `go.work`
  - 或让桌面模块在独立 workspace 下运行

## 失败即阻断项

若出现以下任一情况，暂停正式开发：

- `wails3 dev` 在当前环境不可稳定运行。
- bindings 生成或调用链路不稳定。
- Windows 构建链路存在明显阻断。
- 目录选择或本地文件访问无法满足配置管理需求。

## 后续动作

- 若通过：进入 `gateway WebSocket` 改造和正式 `desktop/` 工程搭建。
- 若不通过：重新评估 `Wails3` 版本、模板、目录结构，必要时改用更稳的桌面壳方案。
