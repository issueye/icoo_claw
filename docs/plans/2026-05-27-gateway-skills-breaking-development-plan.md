# Gateway .skills 不兼容式开发计划

## 背景

当前 Gateway 已经具备技能管理能力，但技能目录通过 `gateway_skills_dir` 配置项指定，默认落在 `data/gateway_skills`。这会让技能路径成为部署配置的一部分，也容易在后续扩展技能安装、版本管理、Agent 级隔离时产生兼容分支和重复逻辑。

本次采用不兼容式设计：删除旧配置入口，固定 Gateway 工作目录下的 `.skills` 为唯一技能管理根目录。

## 目标

1. Gateway 工作目录下固定创建 `.skills` 目录，用于存放 Gateway 管理安装的技能。
2. Claw 默认加载 Gateway `.skills/active` 下的所有技能。
3. Agent 绑定技能时，Gateway 生成实例隔离目录，Claw 只加载该实例绑定技能。
4. 删除 `gateway_skills_dir` 和 `GATEWAY_SKILLS_DIR`，不做旧目录兼容。
5. 删除 request 级技能元数据注入，执行链路只通过文件系统和启动配置传递技能。

## 目录结构

```text
{gateway_work_dir}/.skills/
  active/
    .agents/
      skills/
        {skill_name}/
          SKILL.md
  versions/
    {skill_name}/
      {version}/
        SKILL.md
  agents/
    {instance_id}/
      .agents/
        skills/
          {skill_name}/
            SKILL.md
```

## 关键规则

- `{gateway_work_dir}` 为 Gateway 进程启动工作目录。
- Gateway 技能管理只能读写 `{gateway_work_dir}/.skills`。
- Claw 不直接理解 `.skills` 业务目录，只读取 `ProjectRoot/.agents/skills/**/SKILL.md`。
- 无 Agent 绑定技能时，Gateway 写入 Claw 配置：
  - `gateway_skills.path = "{gateway_work_dir}/.skills/active"`
- 有 Agent 绑定技能时，Gateway 写入 Claw 配置：
  - `gateway_skills.path = "{gateway_work_dir}/.skills/agents/{instance_id}"`
- `SyncSummary()` 只作为管理 API 使用，不参与聊天请求执行。

## 开发阶段

### Phase 1: 配置收敛

- 从 Gateway 配置结构删除 `GatewaySkillsDir`。
- 删除 TOML 字段 `gateway_skills_dir`。
- 删除环境变量 `GATEWAY_SKILLS_DIR`。
- 增加内部派生字段 `GatewayWorkDir`，来自 Gateway 进程工作目录。
- 增加内部方法或函数派生 `.skills` 根路径。
- 更新配置示例、README、release 脚本。

### Phase 2: SkillService 固定目录

- `SkillService` 使用固定 `.skills` 根目录。
- Gateway 启动时创建 `.skills` 基础目录。
- `Create/Update/Delete/Download/SyncSummary/PublishForAgent` 全部基于 `.skills`。
- 删除外部可传入 `SkillGatewayConfig.BaseDir` 的业务语义。

### Phase 3: Claw 启动路径单通道

- `processSpecFromConfig()` 默认使用 `.skills/active`。
- `AgentInstanceService.Start()` 在 Agent 绑定技能时生成 `.skills/agents/{instance_id}`。
- `writeClawConfig()` 只写 `[gateway_skills] path = "..."`。
- 不再传递技能列表 JSON 或技能 summary。

### Phase 4: Claw 请求级技能清理

- `parseAgentProfile()` 不再解析 request payload 中的 `gateway_skills`。
- `RuntimeFactory` 只使用启动配置里的 `GatewaySkills.Path` 作为默认 ProjectRoot。
- 保留 request metadata 中显式 `project_root` 的开发调试能力。

### Phase 5: 测试与验证

- Gateway 配置测试：旧 `gateway_skills_dir` 不再被读取。
- SkillService 测试：技能写入 `.skills/active/.agents/skills`。
- AgentInstanceService 测试：未绑定技能默认使用 `.skills/active`，绑定技能使用 `.skills/agents/{instance_id}`。
- Claw RuntimeFactory 测试：request 中的 `gateway_skills` 不生效。
- 运行：

```powershell
go test ./server/gateway/...
go test ./server/claw/...
```

## 验收标准

- Gateway 启动后，工作目录下存在 `.skills` 基础结构。
- 新建 active 技能后，`SKILL.md` 出现在 `.skills/active/.agents/skills/{name}/SKILL.md`。
- 启动 Claw 实例时，配置中的 `gateway_skills.path` 指向 `.skills/active` 或 `.skills/agents/{instance_id}`。
- 聊天请求中的 `gateway_skills` 字段不会覆盖 Claw 启动时的技能目录。
- 代码中不存在 `GatewaySkillsDir`、`gateway_skills_dir`、`GATEWAY_SKILLS_DIR`。
