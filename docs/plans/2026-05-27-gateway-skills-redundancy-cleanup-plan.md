# Gateway .skills 冗余清理开发方案

## 目标

在 `.skills` 不兼容式设计落地后，继续清理代码中的旧设计影子，避免后续功能重新长出多条技能传递通道。

## 保留语义

- `project_root` request metadata 保留，语义限定为用户项目工作目录。
- Gateway 管理技能仍固定在 `{gateway_work_dir}/.skills`。
- Claw 启动配置中的 `[gateway_skills] path` 暂时保留为配置格式，但代码内部映射为 `DefaultProjectRoot`。

## 优化项

1. 将 Gateway/Claw 内部的一字段 `GatewaySkills{Path}` 包装改为 `DefaultProjectRoot string`。
2. 将 `RuntimeFactory.SetGatewaySkills()` 改为 `SetDefaultProjectRoot()`。
3. 将 Claw 配置加载结果从 `GatewaySkills.Path` 映射到 `DefaultProjectRoot`。
4. 将 Gateway `StartAgentInstanceSpec.GatewaySkills.Path` 改为 `DefaultProjectRoot`。
5. 将 `SkillService` 构造从 `SkillGatewayConfig{BaseDir}` 简化为 `NewSkillService(baseDir, repo)`。
6. 从 `SkillSummary` DTO 删除 `path` 字段，避免管理 API 暴露运行路径。
7. 更新测试名称和断言，使默认项目根目录语义清晰。

## 非目标

- 不删除 request metadata 的 `project_root`，因为桌面项目上下文仍依赖它决定工具执行目录。
- 不立即修改 Claw 配置文件格式 `[gateway_skills] path`，避免扩大本轮改动面。
- 不删除 DB 与 `SKILL.md` 双写，因为当前 DB 是管理态，`.skills` 是运行态派生产物。

## 验收

- 代码中不再存在 `GatewaySkillsConfig`、`GatewaySkills{Path}`、`SetGatewaySkills`、`SkillGatewayConfig`。
- `/v1/skills/sync` 只返回技能列表，不返回运行目录。
- Gateway 启动 Claw 时仍能把 `.skills/active` 或 `.skills/agents/{instance_id}` 写入配置。
- 后端测试全部通过。
