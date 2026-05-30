package api

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"icoo_claw/common/core/agent_sdk/config"
	"icoo_claw/common/core/agent_sdk/model"
	"icoo_claw/common/core/agent_sdk/runtime/skills"
	"icoo_claw/common/core/agent_sdk/runtime/subagents"
	"icoo_claw/common/core/agent_sdk/sandbox"
	"icoo_claw/common/core/agent_sdk/tool"
	toolbuiltin "icoo_claw/common/core/agent_sdk/tool/builtin"
)

func registerTools(registry *tool.Registry, opts Options, settings *config.Settings, skReg *skills.Registry, subMgr *subagents.Manager) error {
	entry := effectiveEntryPoint(opts)
	tools := opts.Tools

	if len(tools) == 0 {
		sandboxDisabled := settings != nil && settings.Sandbox != nil && settings.Sandbox.Enabled != nil && !*settings.Sandbox.Enabled
		if skReg == nil {
			skReg = skills.NewRegistry()
		}

		factories := builtinToolFactories(opts.ProjectRoot, opts.Sandbox.NetworkAllow, sandboxDisabled, entry, settings, skReg, subMgr)
		names := builtinOrder(entry)
		selectedNames := filterBuiltinNames(opts.EnabledBuiltinTools, names)
		for _, name := range selectedNames {
			ctor := factories[name]
			if ctor == nil {
				continue
			}
			impl := ctor()
			if impl == nil {
				continue
			}
			tools = append(tools, impl)
		}

		if len(opts.CustomTools) > 0 {
			tools = append(tools, opts.CustomTools...)
		}
	}
	tools = withToolSearch(tools)

	disallowed := toLowerSet(opts.DisallowedTools)
	if settings != nil && len(settings.DisallowedTools) > 0 {
		if disallowed == nil {
			disallowed = map[string]struct{}{}
		}
		for _, name := range settings.DisallowedTools {
			if key := canonicalToolName(name); key != "" {
				disallowed[key] = struct{}{}
			}
		}
		if len(disallowed) == 0 {
			disallowed = nil
		}
	}

	seen := make(map[string]struct{})
	for _, impl := range tools {
		if impl == nil {
			continue
		}
		name := strings.TrimSpace(impl.Name())
		if name == "" {
			continue
		}
		canon := canonicalToolName(name)
		if disallowed != nil {
			if _, blocked := disallowed[canon]; blocked {
				log.Printf("tool %s skipped: disallowed", name)
				continue
			}
		}
		if _, ok := seen[canon]; ok {
			log.Printf("tool %s skipped: duplicate name", name)
			continue
		}
		seen[canon] = struct{}{}
		if err := registry.Register(impl); err != nil {
			return fmt.Errorf("api: register tool %s: %w", impl.Name(), err)
		}
	}

	return nil
}

func withToolSearch(tools []tool.Tool) []tool.Tool {
	hasDeferred := false
	for _, impl := range tools {
		if impl != nil && canonicalToolName(impl.Name()) == canonicalToolName(toolbuiltin.ToolSearchName) {
			return tools
		}
		if impl != nil && tool.ShouldDefer(impl) {
			hasDeferred = true
		}
	}
	if !hasDeferred {
		return tools
	}
	return append(tools, toolbuiltin.NewToolSearchTool(tools))
}

func builtinToolFactories(root string, networkAllow []string, sandboxDisabled bool, entry EntryPoint, settings *config.Settings, skReg *skills.Registry, subMgr *subagents.Manager) map[string]func() tool.Tool {
	var networkPolicy sandbox.NetworkPolicy
	if !sandboxDisabled {
		if len(networkAllow) == 0 {
			networkAllow = defaultNetworkAllowList(entry)
		}
		networkPolicy = sandbox.NewDomainAllowList(networkAllow...)
	}

	respectGitignore := true
	if settings != nil && settings.RespectGitignore != nil {
		respectGitignore = *settings.RespectGitignore
	}

	cfg := builtinToolFactoryConfig{
		root:             root,
		entry:            entry,
		sandboxDisabled:  sandboxDisabled,
		networkPolicy:    networkPolicy,
		respectGitignore: respectGitignore,
		skReg:            skReg,
		subMgr:           subMgr,
	}
	return map[string]func() tool.Tool{
		"bash":          cfg.bash,
		"read":          cfg.read,
		"write":         cfg.write,
		"edit":          cfg.edit,
		"find":          cfg.find,
		"fetch":         cfg.fetch,
		"web_search":    cfg.webSearch,
		"grep":          cfg.grep,
		"glob":          cfg.glob,
		"skill":         cfg.skill,
		"skill_execute": cfg.skillExecute,
	}
}

type builtinToolFactoryConfig struct {
	root             string
	entry            EntryPoint
	sandboxDisabled  bool
	networkPolicy    sandbox.NetworkPolicy
	respectGitignore bool
	skReg            *skills.Registry
	subMgr           *subagents.Manager
}

func (c builtinToolFactoryConfig) bash() tool.Tool {
	var bash *toolbuiltin.BashTool
	if c.sandboxDisabled {
		bash = toolbuiltin.NewBashToolWithSandbox(c.root, nil)
	} else {
		bash = toolbuiltin.NewBashToolWithRoot(c.root)
	}
	if c.entry == EntryPointCLI {
		bash.AllowShellMetachars(true)
	}
	return bash
}

func (c builtinToolFactoryConfig) read() tool.Tool {
	if c.sandboxDisabled {
		return toolbuiltin.NewReadToolWithSandbox(c.root, nil)
	}
	return toolbuiltin.NewReadToolWithRoot(c.root)
}

func (c builtinToolFactoryConfig) write() tool.Tool {
	if c.sandboxDisabled {
		return toolbuiltin.NewWriteToolWithSandbox(c.root, nil)
	}
	return toolbuiltin.NewWriteToolWithRoot(c.root)
}

func (c builtinToolFactoryConfig) edit() tool.Tool {
	if c.sandboxDisabled {
		return toolbuiltin.NewEditToolWithSandbox(c.root, nil)
	}
	return toolbuiltin.NewEditToolWithRoot(c.root)
}

func (c builtinToolFactoryConfig) grep() tool.Tool {
	var grep *toolbuiltin.GrepTool
	if c.sandboxDisabled {
		grep = toolbuiltin.NewGrepToolWithSandbox(c.root, nil)
	} else {
		grep = toolbuiltin.NewGrepToolWithRoot(c.root)
	}
	grep.SetRespectGitignore(c.respectGitignore)
	return grep
}

func (c builtinToolFactoryConfig) glob() tool.Tool {
	var glob *toolbuiltin.GlobTool
	if c.sandboxDisabled {
		glob = toolbuiltin.NewGlobToolWithSandbox(c.root, nil)
	} else {
		glob = toolbuiltin.NewGlobToolWithRoot(c.root)
	}
	glob.SetRespectGitignore(c.respectGitignore)
	return glob
}

func (c builtinToolFactoryConfig) find() tool.Tool {
	var find *toolbuiltin.FindTool
	if c.sandboxDisabled {
		find = toolbuiltin.NewFindToolWithSandbox(c.root, nil)
	} else {
		find = toolbuiltin.NewFindToolWithRoot(c.root)
	}
	find.SetRespectGitignore(c.respectGitignore)
	return find
}

func (c builtinToolFactoryConfig) fetch() tool.Tool {
	return toolbuiltin.NewFetchToolWithNetworkPolicy(c.networkPolicy)
}

func (c builtinToolFactoryConfig) webSearch() tool.Tool {
	return toolbuiltin.NewWebSearchToolWithNetworkPolicy(c.networkPolicy)
}

func (c builtinToolFactoryConfig) skill() tool.Tool {
	return toolbuiltin.NewSkillToolWithSubagent(c.skReg, nil, c.subMgr)
}

func (c builtinToolFactoryConfig) skillExecute() tool.Tool {
	return toolbuiltin.NewSkillExecuteToolWithSubagent(c.skReg, nil, c.subMgr)
}

func builtinOrder(entry EntryPoint) []string {
	_ = entry
	return []string{"bash", "read", "write", "edit", "find", "fetch", "web_search", "glob", "grep", "skill_execute", "skill"}
}

func filterBuiltinNames(enabled []string, order []string) []string {
	if enabled == nil {
		return append([]string(nil), order...)
	}
	if len(enabled) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(enabled))
	repl := strings.NewReplacer("-", "_", " ", "_")
	for _, name := range enabled {
		key := strings.ToLower(strings.TrimSpace(name))
		key = repl.Replace(key)
		if key != "" {
			set[key] = struct{}{}
		}
	}
	var filtered []string
	for _, name := range order {
		if _, ok := set[name]; ok {
			filtered = append(filtered, name)
		}
	}
	return filtered
}

func effectiveEntryPoint(opts Options) EntryPoint {
	entry := opts.EntryPoint
	if entry == "" {
		entry = opts.Mode.EntryPoint
	}
	if entry == "" {
		entry = defaultEntrypoint
	}
	return entry
}

func registerMCPServers(ctx context.Context, registry *tool.Registry, manager *sandbox.Manager, servers []mcpServer) error {
	for _, server := range servers {
		spec := server.Spec
		if err := enforceSandboxHost(manager, spec); err != nil {
			return err
		}
		opts := tool.MCPServerOptions{
			Headers:       server.Headers,
			Env:           server.Env,
			EnabledTools:  server.EnabledTools,
			DisabledTools: server.DisabledTools,
		}
		if server.TimeoutSeconds > 0 {
			opts.Timeout = time.Duration(server.TimeoutSeconds) * time.Second
		}
		if server.ToolTimeoutSeconds > 0 {
			opts.ToolTimeout = time.Duration(server.ToolTimeoutSeconds) * time.Second
		}

		var err error
		if !hasMCPServerOptions(opts) {
			err = registry.RegisterMCPServer(ctx, spec, server.Name)
		} else {
			err = registry.RegisterMCPServerWithOptions(ctx, spec, server.Name, opts)
		}
		if err != nil {
			return fmt.Errorf("api: register MCP %s: %w", spec, err)
		}
	}
	return nil
}

func hasMCPServerOptions(opts tool.MCPServerOptions) bool {
	return len(opts.Headers) > 0 ||
		len(opts.Env) > 0 ||
		opts.Timeout > 0 ||
		len(opts.EnabledTools) > 0 ||
		len(opts.DisabledTools) > 0 ||
		opts.ToolTimeout > 0
}

func enforceSandboxHost(manager *sandbox.Manager, server string) error {
	if manager == nil || strings.TrimSpace(server) == "" {
		return nil
	}
	u, err := url.Parse(server)
	if err != nil || u == nil || strings.TrimSpace(u.Scheme) == "" {
		return nil
	}
	scheme := strings.ToLower(strings.TrimSpace(u.Scheme))
	base, _, _ := strings.Cut(scheme, "+")
	switch base {
	case "http", "https", "sse":
		if err := manager.CheckNetwork(u.Host); err != nil {
			return fmt.Errorf("api: MCP host denied: %w", err)
		}
	}
	return nil
}

func resolveModel(ctx context.Context, opts Options) (model.Model, error) {
	if opts.Model != nil {
		return opts.Model, nil
	}
	if opts.ModelFactory != nil {
		mdl, err := opts.ModelFactory.Model(ctx)
		if err != nil {
			return nil, fmt.Errorf("api: model factory: %w", err)
		}
		return mdl, nil
	}
	return nil, ErrMissingModel
}

func defaultSessionID(entry EntryPoint) string {
	prefix := strings.TrimSpace(string(entry))
	if prefix == "" {
		prefix = string(defaultEntrypoint)
	}
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
