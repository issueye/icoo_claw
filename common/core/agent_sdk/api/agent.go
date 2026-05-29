package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"icoo_claw/common/core/agent_sdk/config"
	hooks "icoo_claw/common/core/agent_sdk/hooks"
	"icoo_claw/common/core/agent_sdk/message"
	"icoo_claw/common/core/agent_sdk/sandbox"
	"icoo_claw/common/core/agent_sdk/tool"
)

var newTracer = NewTracer

type streamContextKey string

const streamEmitCtxKey streamContextKey = "agentsdk.stream.emit"

const agentEmitCtxKey streamContextKey = "agentsdk.stream.agent.emit"

func withStreamEmit(ctx context.Context, emit streamEmitFunc) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if emit == nil {
		return ctx
	}
	return context.WithValue(ctx, streamEmitCtxKey, emit)
}

func streamEmitFromContext(ctx context.Context) streamEmitFunc {
	if ctx == nil {
		return nil
	}
	if emit, ok := ctx.Value(streamEmitCtxKey).(streamEmitFunc); ok {
		return emit
	}
	return nil
}

func withAgentEmit(ctx context.Context, emit agentEmitFunc) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if emit == nil {
		return ctx
	}
	return context.WithValue(ctx, agentEmitCtxKey, emit)
}

func agentEmitFromContext(ctx context.Context) agentEmitFunc {
	if ctx == nil {
		return nil
	}
	if emit, ok := ctx.Value(agentEmitCtxKey).(agentEmitFunc); ok {
		return emit
	}
	return nil
}

// Runtime exposes the unified SDK surface that powers CLI/CI/enterprise entrypoints.
type Runtime struct {
	opts      Options
	sbRoot    string
	registry  *tool.Registry
	executor  *tool.Executor
	hooks     *hooks.Executor
	histories *historyStore
	compactor *compactor
	deferred  *deferredToolState
	perm      *permissionEvaluator

	mu sync.RWMutex

	runMu     sync.Mutex
	runWG     sync.WaitGroup
	closeOnce sync.Once
	closeErr  error
	closed    bool
}

// New instantiates a unified runtime bound to the provided options.
func New(ctx context.Context, opts Options) (*Runtime, error) {
	opts = opts.withDefaults()
	opts = opts.frozen()

	// 初始化文件系统抽象层
	fsLayer := config.NewFS(opts.ProjectRoot, opts.EmbedFS)
	opts.fsLayer = fsLayer

	if err := materializeEmbeddedClaudeHooks(opts.ProjectRoot, opts.EmbedFS); err != nil {
		log.Printf("claude hooks materializer warning: %v", err)
	}

	builder := opts.SystemPromptBuilder
	if builder == nil {
		builder = NewSystemPromptBuilder()
	} else {
		builder = builder.Clone()
	}
	if text := strings.TrimSpace(opts.SystemPrompt); text != "" {
		builder.AddSection(SystemPromptSectionIdentity, text, SystemPromptPriorityIdentity)
	}

	if memory, err := config.LoadAgentsMD(opts.ProjectRoot, fsLayer); err != nil {
		log.Printf("agents.md loader warning: %v", err)
	} else if strings.TrimSpace(memory) != "" {
		builder.AddSection(SystemPromptSectionMemory, fmt.Sprintf("## Memory\n\n%s", strings.TrimSpace(memory)), SystemPromptPriorityMemory)
	}

	settings, err := loadSettings(opts)
	if err != nil {
		return nil, err
	}
	opts.settingsSnapshot = settings

	mdl, err := resolveModel(ctx, opts)
	if err != nil {
		return nil, err
	}
	opts.Model = mdl

	sbox, sbRoot := buildSandboxManager(opts, settings)

	// 初始化技能注册器
	skReg, skErrs := buildSkillsRegistry(opts)
	for _, err := range skErrs {
		log.Printf("skill loader warning: %v", err)
	}
	opts.skReg = skReg

	// 初始化子智能管理器
	subMgr, subErrs := buildSubagentsManager(opts)
	for _, err := range subErrs {
		log.Printf("subagent loader warning: %v", err)
	}
	opts.subMgr = subMgr

	registry := tool.NewRegistry()
	if err := registerTools(registry, opts, settings, opts.skReg, opts.subMgr); err != nil {
		return nil, err
	}
	mcpServers := collectMCPServers(settings, opts.MCPServers)
	if err := registerMCPServers(ctx, registry, sbox, mcpServers); err != nil {
		return nil, err
	}
	executor := tool.NewExecutor(registry, sbox).
		WithOutputPersister(tool.NewOutputPersister()).
		WithMaxOutputSize(opts.MaxToolOutputSize)

	hooks := newHookExecutor(opts, settings)
	compactor := newCompactor(opts.AutoCompact, opts.TokenLimit)
	permissions := newPermissionEvaluator(settings, registry, opts.PermissionPrompter)

	tracer, err := newTracer(opts.OTEL)
	if err != nil {
		return nil, fmt.Errorf("otel tracer init: %w", err)
	}
	opts.tracer = tracer

	if opts.RulesEnabled == nil || (opts.RulesEnabled != nil && *opts.RulesEnabled) {
		loader := config.NewRulesLoader(opts.ProjectRoot)
		if _, err := loader.LoadRules(); err != nil {
			log.Printf("rules loader warning: %v", err)
		} else if rules := strings.TrimSpace(loader.GetContent()); rules != "" {
			builder.AddSection(SystemPromptSectionRules, fmt.Sprintf("## Project Rules\n\n%s", rules), SystemPromptPriorityRules)
		}
		if err := loader.Close(); err != nil {
			log.Printf("rules loader close warning: %v", err)
		}
	}
	opts.SystemPromptBuilder = builder
	opts.SystemPrompt = builder.Build()

	// 组装 loader & saver 适配器
	var loader func(string) ([]message.Message, error)
	if opts.SessionStore != nil {
		loader = func(id string) ([]message.Message, error) {
			return opts.SessionStore.LoadHistory(context.Background(), id)
		}
	} else if opts.HistoryLoader != nil {
		loader = opts.HistoryLoader
	}

	var saver func(string, []message.Message) error
	if opts.SessionStore != nil {
		saver = func(id string, msgs []message.Message) error {
			return opts.SessionStore.SaveHistory(context.Background(), id, msgs)
		}
	} else if opts.HistorySaver != nil {
		saver = opts.HistorySaver
	}

	histories := newHistoryStoreWithSaver(opts.MaxSessions, loader, saver)

	rt := &Runtime{
		opts:      opts,
		sbRoot:    sbRoot,
		registry:  registry,
		executor:  executor,
		hooks:     hooks,
		histories: histories,
		compactor: compactor,
		deferred:  newDeferredToolState(registry),
		perm:      permissions,
	}

	if rt.deferred != nil {
		histories.onEvict = func(evicted string) {
			rt.deferred.evict(evicted)
		}
		if opts.SessionStore != nil {
			rt.deferred.saver = func(sessionID string, activeTools []string) error {
				return opts.SessionStore.SaveDeferredState(context.Background(), sessionID, activeTools)
			}
		}
	}

	rt.bindRuntimeSubagents()
	rt.bindSubagentCallbacks()
	return rt, nil
}

func (rt *Runtime) beginRun() error {
	rt.runMu.Lock()
	defer rt.runMu.Unlock()
	if rt.closed {
		return ErrRuntimeClosed
	}
	rt.runWG.Add(1)
	return nil
}

func (rt *Runtime) endRun() {
	rt.runWG.Done()
}

// Run executes the unified pipeline synchronously.
func (rt *Runtime) Run(ctx context.Context, req Request) (*Response, error) {
	if rt == nil {
		return nil, ErrRuntimeClosed
	}
	if err := rt.beginRun(); err != nil {
		return nil, err
	}
	defer rt.endRun()

	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		mode := rt.opts.modeContext()
		sessionID = defaultSessionID(mode.EntryPoint)
	}
	req.SessionID = sessionID

	prep, err := rt.prepare(ctx, req)
	if err != nil {
		return nil, err
	}
	result, err := rt.runAgent(prep)
	if err != nil {
		return nil, err
	}
	return rt.buildResponse(prep, result), nil
}

// RunStream executes the pipeline asynchronously and returns events over a channel.
// The returned events use the default Anthropic SSE protocol for backward
// compatibility. Use RunStreamWithProtocol for alternative wire formats.
func (rt *Runtime) RunStream(ctx context.Context, req Request) (<-chan StreamEvent, error) {
	return rt.RunStreamWithProtocol(ctx, req, nil)
}

// RunStreamWithProtocol executes the pipeline asynchronously using the given
// StreamProtocol to encode events. If protocol is nil, the default
// AnthropicSSEProtocol is used, preserving backward compatibility with RunStream.
func (rt *Runtime) RunStreamWithProtocol(ctx context.Context, req Request, protocol StreamProtocol) (<-chan StreamEvent, error) {
	if rt == nil {
		return nil, ErrRuntimeClosed
	}
	if strings.TrimSpace(req.Prompt) == "" && len(req.ContentBlocks) == 0 {
		return nil, errors.New("api: prompt is empty")
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		mode := rt.opts.modeContext()
		sessionID = defaultSessionID(mode.EntryPoint)
	}
	req.SessionID = sessionID

	if err := rt.beginRun(); err != nil {
		return nil, err
	}

	if protocol == nil {
		protocol = NewAnthropicSSEProtocol()
	}

	out := make(chan StreamEvent, 512)
	progressChan := make(chan StreamEvent, 256)
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	progressMW := newProgressMiddlewareWithProtocol(progressChan, protocol)
	ctxWithEmit := withStreamEmit(baseCtx, progressMW.streamEmit())
	ctxWithAgentEmit := withAgentEmit(ctxWithEmit, progressMW.streamEmitAgent())

	go func() {
		defer rt.endRun()
		defer close(out)

		prep, err := rt.prepare(ctxWithAgentEmit, req)
		if err != nil {
			isErr := true
			out <- StreamEvent{Type: EventError, Output: err.Error(), IsError: &isErr}
			return
		}

		done := make(chan struct{})
		go func() {
			defer close(done)
			dropping := false
			for event := range progressChan {
				if dropping {
					continue
				}
				select {
				case out <- event:
				case <-ctxWithAgentEmit.Done():
					dropping = true
				}
			}
		}()

		var runErr error
		var result runResult
		defer func() {
			if rt.hooks != nil {
				reason := "completed"
				if runErr != nil {
					reason = "error"
				}
				//nolint:errcheck // session end events are non-critical notifications
				rt.hooks.Publish(hooks.Event{
					Type:      hooks.SessionEnd,
					SessionID: req.SessionID,
					Payload:   hooks.SessionEndPayload{SessionID: req.SessionID, Reason: reason},
				})
			}
		}()

		result, runErr = rt.runAgentWithMiddleware(prep, progressMW)
		close(progressChan)
		<-done

		if runErr != nil {
			isErr := true
			out <- StreamEvent{Type: EventError, Output: runErr.Error(), IsError: &isErr}
			return
		}
		rt.buildResponse(prep, result)
	}()
	return out, nil
}

// Close releases held resources.
func (rt *Runtime) Close() error {
	if rt == nil {
		return nil
	}
	rt.closeOnce.Do(func() {
		rt.runMu.Lock()
		rt.closed = true
		rt.runMu.Unlock()

		rt.runWG.Wait()

		var err error
		if rt.histories != nil {
			for _, sessionID := range rt.histories.SessionIDs() {
				if cleanupErr := cleanupBashOutputSessionDir(sessionID); cleanupErr != nil {
					log.Printf("api: session %q temp cleanup failed: %v", sessionID, cleanupErr)
				}
				if cleanupErr := cleanupToolOutputSessionDir(sessionID); cleanupErr != nil {
					log.Printf("api: session %q tool output cleanup failed: %v", sessionID, cleanupErr)
				}
			}
		}
		if rt.registry != nil {
			rt.registry.Close()
		}
		if rt.opts.tracer != nil {
			if e := rt.opts.tracer.Shutdown(); e != nil {
				err = errors.Join(err, e)
			}
		}
		rt.closeErr = err
	})
	return rt.closeErr
}

// Config returns the last loaded project config.
func (rt *Runtime) Config() *config.Settings {
	if rt == nil {
		return nil
	}
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return projectConfigFromSettings(rt.opts.settingsSnapshot)
}

// Settings exposes the merged settings.json snapshot for callers that need it.
func (rt *Runtime) Settings() *config.Settings {
	if rt == nil {
		return nil
	}
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return config.MergeSettings(nil, rt.opts.settingsSnapshot)
}

// Sandbox exposes the sandbox manager.
func (rt *Runtime) Sandbox() *sandbox.Manager {
	if rt == nil || rt.executor == nil {
		return nil
	}
	return rt.executor.Sandbox()
}

// SessionHistory returns a cloned snapshot of an existing session history.
func (rt *Runtime) SessionHistory(sessionID string) ([]message.Message, bool) {
	if rt == nil || rt.histories == nil {
		return nil, false
	}
	return rt.histories.Snapshot(sessionID)
}

// SummarizeSession 手动触发并生成指定会话的摘要，若配置了 SessionStore，摘要也会自动同步存入元数据。
func (rt *Runtime) SummarizeSession(ctx context.Context, sessionID string) (string, error) {
	if rt == nil {
		return "", errors.New("api: runtime is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", errors.New("api: sessionID cannot be empty")
	}

	// 1. 获取会话历史
	history, err := rt.histories.Get(sessionID)
	if err != nil {
		return "", err
	}
	snapshot := history.All()
	if len(snapshot) == 0 {
		return "", nil
	}

	// 2. 获取模型
	mdl := rt.opts.Model
	if mdl == nil {
		return "", errors.New("api: model is not configured")
	}

	// 3. 剥离工具 I/O 并提交模型生成摘要
	stripped := stripToolIO(snapshot)
	summary, err := compressMessages(ctx, mdl, stripped, rt.opts.AutoCompact.SummaryPrompt)
	if err != nil {
		return "", fmt.Errorf("api: summarize session %q failed: %w", sessionID, err)
	}

	summary = strings.TrimSpace(summary)
	if summary == "" {
		return "", nil
	}

	// 4. 持久化存入 SessionStore
	if rt.opts.SessionStore != nil {
		if err := rt.opts.SessionStore.UpdateSessionSummary(ctx, sessionID, summary); err != nil {
			log.Printf("api: warning: failed to save summary to SessionStore for session %q: %v", sessionID, err)
		}
	}

	return summary, nil
}

// ----------------- internal helpers -----------------
