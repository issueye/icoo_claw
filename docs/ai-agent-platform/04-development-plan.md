# 开发计划

## 当前方向

Gateway Session API 采用 GORM + no-cgo SQLite。额外外部协议不属于 MVP 主线。

## 阶段 0：配置与骨架

已完成：

- Gateway/Claw Gin skeleton。
- 手动 DI。
- TOML 配置加载。
- README TOML 启动说明。

## 阶段 1：Gateway Session API

已完成：

- GORM repository。
- Session CRUD。
- Session list。
- Message append/list/replace snapshot。
- Revision 冲突保护。
- Run append/list。
- Run event append/list。

待完善：

- 更完整的分页 contract。
- run event 在流式链路中的统一写入策略。

## 阶段 2：Claw

已完成：

- AgentService。
- SDKRunner。
- FakeRunner。
- RuntimeFactory。
- HistoryAdapter。
- Gateway Session API client。
- 内部 token middleware。

待完善：

- 强类型 AgentProfile contract。
- SDK runner 的错误分类。
- 流式结束后的 snapshot 错误上报策略。

## 阶段 3：Gateway

已完成：

- AgentProfile CRUD。
- Conversation CRUD/消息入口。
- AgentInstance 启动、停止、重启、drain。
- RouterPolicy: sticky + least inflight + auto start。
- 健康巡检。
- Gateway 启动 Claw 时生成实例 TOML。

待完善：

- 同 session 并发锁。
- draining 超时后的错误响应细化。
- instance 退出回收与端口释放策略。

## 阶段 4：端到端测试

已完成：

- 进程级端到端测试：
  - build Gateway/Claw。
  - 启动 Gateway。
  - Gateway 自动拉起 fake Claw。
  - 创建 AgentProfile。
  - 创建 Conversation。
  - 发送同步消息。
  - 查询历史消息。

待完善：

- 流式端到端测试。
- session busy 端到端测试。
- restart/drain 端到端测试。

## 阶段 5：MVP 加固

下一步建议顺序：

1. 强类型化 Gateway -> Claw RunRequest 中的 AgentProfile。
2. 补流式消息端到端测试。
3. 加同 session 并发锁。
4. 完善 run/run event 写入。
5. 更新接口示例与错误码文档。

## 风险

- 当前 Gateway 使用单机 SQLite 保存控制面与会话数据；并发写入需要谨慎控制。
- Gateway 自动拉起 Claw 依赖本地二进制路径，生产部署需替换为远程或容器调度。
- fake runner 测试链路稳定，但真实模型链路还需要 API key/模型 provider 的配置策略。

