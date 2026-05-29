package api

import (
	"context"

	"icoo_claw/common/core/agent_sdk/middleware"
	"icoo_claw/common/core/agent_sdk/model"
	"icoo_claw/common/core/agent_sdk/tool"
)

type RuntimeBuilder struct {
	opts Options
}

func NewRuntimeBuilder(projectRoot string) *RuntimeBuilder {
	return &RuntimeBuilder{opts: Options{ProjectRoot: projectRoot}}
}

func NewCodingAgent(projectRoot string) *RuntimeBuilder {
	return NewRuntimeBuilder(projectRoot).WithBuiltinTools("bash", "read", "write", "edit", "find", "fetch", "glob", "grep", "skill")
}

func NewResearchAgent(projectRoot string) *RuntimeBuilder {
	return NewRuntimeBuilder(projectRoot).WithBuiltinTools("read", "find", "fetch", "web_search", "glob", "grep")
}

func NewToolOnlyAgent(projectRoot string, tools ...tool.Tool) *RuntimeBuilder {
	return NewRuntimeBuilder(projectRoot).WithTools(tools...)
}

func NewSafeLocalAgent(projectRoot string) *RuntimeBuilder {
	return NewRuntimeBuilder(projectRoot).WithBuiltinTools("read", "find", "glob", "grep")
}

func (b *RuntimeBuilder) WithModel(m model.Model) *RuntimeBuilder {
	b.opts.Model = m
	return b
}

func (b *RuntimeBuilder) WithModelFactory(factory ModelFactory) *RuntimeBuilder {
	b.opts.ModelFactory = factory
	return b
}

func (b *RuntimeBuilder) WithSystemPrompt(prompt string) *RuntimeBuilder {
	b.opts.SystemPrompt = prompt
	return b
}

func (b *RuntimeBuilder) WithTools(tools ...tool.Tool) *RuntimeBuilder {
	b.opts.Tools = append([]tool.Tool(nil), tools...)
	return b
}

func (b *RuntimeBuilder) WithCustomTools(tools ...tool.Tool) *RuntimeBuilder {
	b.opts.CustomTools = append([]tool.Tool(nil), tools...)
	return b
}

func (b *RuntimeBuilder) WithBuiltinTools(names ...string) *RuntimeBuilder {
	b.opts.EnabledBuiltinTools = append([]string(nil), names...)
	return b
}

func (b *RuntimeBuilder) WithoutTools(names ...string) *RuntimeBuilder {
	b.opts.DisallowedTools = append([]string(nil), names...)
	return b
}

func (b *RuntimeBuilder) WithPermissionPrompter(prompter PermissionPrompter) *RuntimeBuilder {
	b.opts.PermissionPrompter = prompter
	return b
}

func (b *RuntimeBuilder) WithMiddleware(items ...middleware.Middleware) *RuntimeBuilder {
	b.opts.Middleware = append([]middleware.Middleware(nil), items...)
	return b
}

func (b *RuntimeBuilder) WithSandbox(sandbox SandboxOptions) *RuntimeBuilder {
	b.opts.Sandbox = sandbox
	return b
}

func (b *RuntimeBuilder) WithModelPool(pool map[ModelTier]model.Model) *RuntimeBuilder {
	b.opts.ModelPool = pool
	return b
}

func (b *RuntimeBuilder) WithSubagentModelMapping(mapping map[string]ModelTier) *RuntimeBuilder {
	b.opts.SubagentModelMapping = mapping
	return b
}

func (b *RuntimeBuilder) WithMaxIterations(n int) *RuntimeBuilder {
	b.opts.MaxIterations = n
	return b
}

func (b *RuntimeBuilder) WithAutoCompact(config CompactConfig) *RuntimeBuilder {
	b.opts.AutoCompact = config
	return b
}

func (b *RuntimeBuilder) Options() Options {
	return b.opts.frozen()
}

func (b *RuntimeBuilder) Build(ctx context.Context) (*Runtime, error) {
	return New(ctx, b.Options())
}
