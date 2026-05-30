package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"maps"
	"strings"

	hooks "icoo_claw/common/core/agent_sdk/hooks"
	"icoo_claw/common/core/agent_sdk/message"
	"icoo_claw/common/core/agent_sdk/middleware"
	"icoo_claw/common/core/agent_sdk/model"
	"icoo_claw/common/core/agent_sdk/runtime/skills"
	"icoo_claw/common/core/agent_sdk/runtime/subagents"
	toolbuiltin "icoo_claw/common/core/agent_sdk/tool/builtin"

	"github.com/google/uuid"
)

type preparedRun struct {
	ctx            context.Context
	prompt         string
	modelPrompt    string
	contentBlocks  []model.ContentBlock
	history        *message.History
	normalized     Request
	recorder       *hookRecorder
	skillResults   []SkillExecution
	subagentResult *subagents.Result
	mode           ModeContext
	toolWhitelist  map[string]struct{}
	userInputIndex int
}

type runResult struct {
	response *model.Response
}

func (rt *Runtime) prepare(ctx context.Context, req Request) (preparedRun, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	mode := rt.opts.modeContext()
	fallbackSession := defaultSessionID(mode.EntryPoint)
	normalized := req.normalized(mode, fallbackSession)
	prompt := strings.TrimSpace(normalized.Prompt)
	if prompt == "" && len(normalized.ContentBlocks) == 0 {
		return preparedRun{}, errors.New("api: prompt is empty")
	}

	// Auto-generate RequestID if not provided (UUID tracking)
	if normalized.RequestID == "" {
		normalized.RequestID = uuid.New().String()
	}

	history, err := rt.histories.Get(normalized.SessionID)
	if err != nil {
		return preparedRun{}, err
	}
	userInputIndex := history.Len()

	if rt.deferred != nil && rt.opts.SessionStore != nil {
		loader := func(c context.Context, id string) ([]string, error) {
			return rt.opts.SessionStore.LoadDeferredState(c, id)
		}
		if err := rt.deferred.loadIfMissing(ctx, normalized.SessionID, loader); err != nil {
			log.Printf("api: warning: failed to load deferred tools state for session %q: %v", normalized.SessionID, err)
		}
	}
	recorder := defaultHookRecorder()

	activation := normalized.activationContext(prompt)

	skillRes, modelPrompt, err := rt.executeSkills(ctx, prompt, activation, &normalized)
	if err != nil {
		return preparedRun{}, err
	}
	activation.Prompt = modelPrompt
	subRes, promptAfterSubagent, err := rt.executeSubagent(ctx, modelPrompt, activation, &normalized)
	if err != nil {
		return preparedRun{}, err
	}
	modelPrompt = promptAfterSubagent
	activation.Prompt = modelPrompt
	whitelist := combineToolWhitelists(normalized.ToolWhitelist, nil)
	return preparedRun{
		ctx:            ctx,
		prompt:         prompt,
		modelPrompt:    modelPrompt,
		contentBlocks:  normalized.ContentBlocks,
		history:        history,
		normalized:     normalized,
		recorder:       recorder,
		skillResults:   skillRes,
		subagentResult: subRes,
		mode:           normalized.Mode,
		toolWhitelist:  whitelist,
		userInputIndex: userInputIndex,
	}, nil
}

func (rt *Runtime) runAgent(prep preparedRun) (runResult, error) {
	return rt.runAgentWithMiddleware(prep)
}

func (rt *Runtime) runAgentWithMiddleware(prep preparedRun, extras ...middleware.Middleware) (runResult, error) {
	// Select model based on request tier or subagent mapping
	selectedModel, selectedTier := rt.selectModelForSubagent(prep.normalized.TargetSubagent, prep.normalized.Model)
	_ = selectedTier

	// Determine cache enablement: request-level overrides global default
	enableCache := rt.opts.DefaultEnableCache
	if prep.normalized.EnablePromptCache != nil {
		enableCache = *prep.normalized.EnablePromptCache
	}

	hookAdapter := &runtimeHookAdapter{
		executor:          rt.hooks,
		recorder:          prep.recorder,
		disableSafetyHook: rt.opts.DisableSafetyHook,
	}

	toolExec := &runtimeToolExecutor{
		executor:  rt.executor,
		hooks:     hookAdapter,
		history:   prep.history,
		allow:     prep.toolWhitelist,
		root:      rt.sbRoot,
		host:      "localhost",
		sessionID: prep.normalized.SessionID,
		deferred:  rt.deferred,
		perm:      rt.perm,
	}

	chainItems := make([]middleware.Middleware, 0, len(rt.opts.Middleware)+len(extras))
	if len(rt.opts.Middleware) > 0 {
		chainItems = append(chainItems, rt.opts.Middleware...)
	}
	if len(extras) > 0 {
		chainItems = append(chainItems, extras...)
	}
	chain := middleware.NewChain(chainItems, middleware.WithTimeout(rt.opts.MiddlewareTimeout))

	resp, err := rt.runLoop(prep, selectedModel, hookAdapter, toolExec, chain, enableCache)
	if err != nil {
		return runResult{response: resp}, err
	}
	return runResult{response: resp}, nil
}

func (rt *Runtime) runLoop(prep preparedRun, mdl model.Model, hookAdapter *runtimeHookAdapter, tools *runtimeToolExecutor, chain *middleware.Chain, enableCache bool) (*model.Response, error) {
	if prep.history == nil {
		return nil, errors.New("api: history is nil")
	}
	if mdl == nil {
		return nil, errors.New("api: model is nil")
	}
	if chain == nil {
		return nil, errors.New("api: middleware chain is nil")
	}

	ctx := prep.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	appendUserInput(prep.history, prep.prompt, prep.contentBlocks)

	state := newRunState(prep.normalized, rt.opts.skReg)

	ctx = context.WithValue(ctx, model.MiddlewareStateKey, state)

	systemPrompt := rt.systemPromptForSession(prep.normalized.SessionID, prep.toolWhitelist)

	trimmer := rt.newTrimmer()
	budgetTracker := newTokenBudgetTracker(rt.opts.TokenBudget)

	tracer := rt.opts.tracer
	agentSpan := SpanContext(nil)
	if tracer != nil {
		agentSpan = tracer.StartAgentSpan(prep.normalized.SessionID, prep.normalized.RequestID, 0)
	}
	var iterations int
	var runErr error
	defer func() {
		if tracer == nil {
			return
		}
		tracer.EndSpan(agentSpan, map[string]any{
			"session_id":  strings.TrimSpace(prep.normalized.SessionID),
			"request_id":  strings.TrimSpace(prep.normalized.RequestID),
			"iterations":  iterations,
			"entry_point": string(prep.normalized.Mode.EntryPoint),
		}, runErr)
	}()

	var last *model.Response
	stopReinjections := 0
	for iteration := 0; ; iteration++ {
		iterations = iteration + 1
		if err := ctx.Err(); err != nil {
			runErr = err
			return last, err
		}
		if rt.opts.MaxIterations > 0 && iteration >= rt.opts.MaxIterations {
			runErr = ErrMaxIterations
			return last, ErrMaxIterations
		}

		state.Iteration = iteration

		if rt.compactor != nil {
			if _, err := rt.compactor.maybeCompact(ctx, prep.normalized.SessionID, prep.history, mdl, rt.opts.SessionStore); err != nil {
				runErr = err
				return last, err
			}
		}

		snapshot := prep.history.All()
		snapshot = applyTransientModelPrompt(snapshot, prep)
		if trimmer != nil {
			snapshot = trimmer.Trim(snapshot)
		}
		req := rt.modelRequestForIteration(prep, snapshot, systemPrompt, enableCache)
		state.ModelInput = &req
		state.Values["model.request"] = req
		if err := chain.Execute(ctx, middleware.StageBeforeAgent, state); err != nil {
			runErr = err
			return last, err
		}

		resp, err := rt.completeWithRecovery(ctx, mdl, req, prep.history, tracer, agentSpan, prep.normalized)
		if err != nil {
			runErr = err
			return last, err
		}
		if resp == nil {
			if tracer != nil {
				tracer.EndSpan(nil, map[string]any{
					"session_id": strings.TrimSpace(prep.normalized.SessionID),
					"request_id": strings.TrimSpace(prep.normalized.RequestID),
				}, errors.New("api: model returned no final response"))
			}
			runErr = errors.New("api: model returned no final response")
			return last, errors.New("api: model returned no final response")
		}
		last = resp
		state.ModelOutput = resp
		state.Values["model.response"] = resp
		state.Values["model.usage"] = resp.Usage
		state.Values["model.stop_reason"] = resp.StopReason
		stopErr := budgetTracker.Observe(resp.Usage)

		appendAssistantResponse(prep.history, resp)

		if err := chain.Execute(ctx, middleware.StageAfterAgent, state); err != nil {
			runErr = err
			return last, err
		}
		if len(resp.Message.ToolCalls) > 0 {
			if err := rt.executeToolCalls(ctx, resp.Message.ToolCalls, tools, chain, state, tracer, agentSpan, prep.normalized); err != nil {
				runErr = err
				return last, err
			}
			if stopErr != nil {
				runErr = stopErr
				return resp, stopErr
			}
		}
		if len(resp.Message.ToolCalls) == 0 {
			if stopErr != nil {
				runErr = stopErr
				return resp, stopErr
			}
			if hookAdapter != nil {
				blockingError, err := hookAdapter.evaluateStop(ctx, resp.StopReason)
				if err != nil {
					runErr = err
					return resp, err
				}
				if blockingError != "" {
					stopReinjections++
					if stopReinjections > rt.opts.StopReinjectionLimit {
						runErr = fmt.Errorf("hooks: stop blocked: %s", blockingError)
						return resp, runErr
					}
					prep.history.Append(message.Message{
						Role:    "user",
						Content: fmt.Sprintf("[System] Stop blocked: %s. Please address this issue.", blockingError),
					})
					continue
				}
			}
			runErr = nil
			return resp, nil
		}
	}
}

func appendUserInput(history *message.History, prompt string, blocks []model.ContentBlock) {
	if history == nil {
		return
	}
	if strings.TrimSpace(prompt) == "" && len(blocks) == 0 {
		return
	}
	userMsg := message.Message{Role: "user", Content: strings.TrimSpace(prompt)}
	if len(blocks) > 0 {
		userMsg.ContentBlocks = convertAPIContentBlocks(blocks)
	}
	history.Append(userMsg)
}

func applyTransientModelPrompt(snapshot []message.Message, prep preparedRun) []message.Message {
	modelPrompt := strings.TrimSpace(prep.modelPrompt)
	if modelPrompt == "" || modelPrompt == strings.TrimSpace(prep.prompt) {
		return snapshot
	}
	idx := prep.userInputIndex
	if idx < 0 || idx >= len(snapshot) || strings.TrimSpace(snapshot[idx].Role) != "user" {
		idx = findMatchingUserInput(snapshot, prep.prompt)
	}
	if idx < 0 {
		return snapshot
	}
	out := message.CloneMessages(snapshot)
	out[idx].Content = modelPrompt
	if len(out[idx].ContentBlocks) > 0 {
		out[idx].ContentBlocks = replaceTextBlock(out[idx].ContentBlocks, modelPrompt)
	}
	return out
}

func findMatchingUserInput(snapshot []message.Message, prompt string) int {
	prompt = strings.TrimSpace(prompt)
	for i := len(snapshot) - 1; i >= 0; i-- {
		msg := snapshot[i]
		if strings.TrimSpace(msg.Role) != "user" {
			continue
		}
		if strings.TrimSpace(msg.Content) == prompt {
			return i
		}
	}
	return -1
}

func replaceTextBlock(blocks []message.ContentBlock, prompt string) []message.ContentBlock {
	out := append([]message.ContentBlock(nil), blocks...)
	for i := range out {
		if out[i].Type == message.ContentBlockText {
			out[i].Text = prompt
			return out
		}
	}
	return append([]message.ContentBlock{{Type: message.ContentBlockText, Text: prompt}}, out...)
}

func newRunState(req Request, skReg *skills.Registry) *middleware.State {
	state := &middleware.State{Values: map[string]any{}}
	if sessionID := strings.TrimSpace(req.SessionID); sessionID != "" {
		state.Values["session_id"] = sessionID
	}
	if requestID := strings.TrimSpace(req.RequestID); requestID != "" {
		state.Values["request_id"] = requestID
	}
	if len(req.ForceSkills) > 0 {
		state.Values["request.force_skills"] = append([]string(nil), req.ForceSkills...)
	}
	if skReg != nil {
		state.Values["skills.registry"] = skReg
	}
	return state
}

func (rt *Runtime) modelRequestForIteration(prep preparedRun, snapshot []message.Message, systemPrompt string, enableCache bool) model.Request {
	return model.Request{
		Messages:          convertMessages(snapshot),
		Tools:             availableToolsForSession(rt.registry, prep.toolWhitelist, rt.deferred, prep.normalized.SessionID),
		System:            systemPrompt,
		EnablePromptCache: enableCache,
	}
}

func appendAssistantResponse(history *message.History, resp *model.Response) {
	if history == nil || resp == nil {
		return
	}
	assistant := message.Message{
		Role:             resp.Message.Role,
		Content:          strings.TrimSpace(resp.Message.Content),
		ReasoningContent: resp.Message.ReasoningContent,
	}
	if len(resp.Message.ToolCalls) > 0 {
		assistant.ToolCalls = make([]message.ToolCall, len(resp.Message.ToolCalls))
		for i, call := range resp.Message.ToolCalls {
			assistant.ToolCalls[i] = message.ToolCall{ID: call.ID, Name: call.Name, Arguments: call.Arguments}
		}
	}
	history.Append(assistant)
}

func (rt *Runtime) buildResponse(prep preparedRun, result runResult) *Response {
	events := []hooks.Event(nil)
	if prep.recorder != nil {
		events = prep.recorder.Drain()
	}
	resp := &Response{
		Mode:            prep.mode,
		RequestID:       prep.normalized.RequestID,
		Result:          convertRunResult(result),
		SkillResults:    prep.skillResults,
		Subagent:        prep.subagentResult,
		HookEvents:      events,
		ProjectConfig:   rt.Settings(),
		Settings:        rt.Settings(),
		SandboxSnapshot: rt.sandboxReport(),
		Tags:            maps.Clone(prep.normalized.Tags),
	}
	return resp
}

func (rt *Runtime) sandboxReport() SandboxReport {
	report := snapshotSandbox(rt.Sandbox())

	var roots []string
	if root := strings.TrimSpace(rt.sbRoot); root != "" {
		roots = append(roots, root)
	}
	report.Roots = cloneStrings(roots)

	allowed := make([]string, 0, len(rt.opts.Sandbox.AllowedPaths))
	for _, path := range rt.opts.Sandbox.AllowedPaths {
		if clean := strings.TrimSpace(path); clean != "" {
			allowed = append(allowed, clean)
		}
	}
	for _, path := range additionalSandboxPaths(rt.opts.settingsSnapshot) {
		if clean := strings.TrimSpace(path); clean != "" {
			allowed = append(allowed, clean)
		}
	}
	report.AllowedPaths = cloneStrings(allowed)

	domains := rt.opts.Sandbox.NetworkAllow
	if len(domains) == 0 {
		domains = defaultNetworkAllowList(rt.opts.EntryPoint)
	}
	var cleanedDomains []string
	for _, domain := range domains {
		if host := strings.TrimSpace(domain); host != "" {
			cleanedDomains = append(cleanedDomains, host)
		}
	}
	report.AllowedDomains = cloneStrings(cleanedDomains)
	return report
}

func convertRunResult(res runResult) *Result {
	if res.response == nil {
		return nil
	}
	return &Result{
		Output:     strings.TrimSpace(res.response.Message.Content),
		StopReason: res.response.StopReason,
		Usage:      res.response.Usage,
		ToolCalls:  append([]model.ToolCall(nil), res.response.Message.ToolCalls...),
	}
}

func (rt *Runtime) executeSkills(ctx context.Context, prompt string, activation skills.ActivationContext, req *Request) ([]SkillExecution, string, error) {
	if rt.opts.skReg == nil {
		return nil, prompt, nil
	}
	matches := rt.opts.skReg.Match(activation)
	forced := orderedForcedSkills(rt.opts.skReg, req.ForceSkills)
	matches = append(matches, forced...)
	if len(matches) == 0 {
		return nil, prompt, nil
	}
	prefix := ""
	execs := make([]SkillExecution, 0, len(matches))
	seen := map[string]struct{}{}
	for _, match := range matches {
		skill := match.Skill
		if skill == nil {
			continue
		}
		name := skill.Definition().Name
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		loaded, err := skill.Execute(ctx, activation)
		if err != nil {
			return execs, "", err
		}
		res, err := rt.executeSkillViaSubagent(ctx, loaded, activation, req)
		execs = append(execs, SkillExecution{Definition: skill.Definition(), Result: res, Err: err})
		if err != nil {
			return execs, "", err
		}
		prefix = combinePrompt(prefix, res.Output)
		activation.Metadata = mergeMetadata(activation.Metadata, res.Metadata)
		mergeTags(req, res.Metadata)
		applyCommandMetadata(req, res.Metadata)
	}
	prompt = prependPrompt(prompt, prefix)
	prompt = applyPromptMetadata(prompt, activation.Metadata)
	return execs, prompt, nil
}

func (rt *Runtime) executeSkillViaSubagent(ctx context.Context, loaded skills.Result, activation skills.ActivationContext, req *Request) (skills.Result, error) {
	if rt == nil || rt.opts.subMgr == nil {
		return skills.Result{}, errors.New("api: skill execution requires subagent manager")
	}
	subReq := toolbuiltin.SkillSubagentRequest(loaded, activation)
	if strings.TrimSpace(subReq.Instruction) == "" {
		return skills.Result{}, errors.New("api: skill instruction is empty")
	}
	if subReq.Metadata == nil {
		subReq.Metadata = map[string]any{}
	}
	if sessionID := isolatedSkillSessionID(loaded, req); sessionID != "" {
		subReq.Metadata["session_id"] = sessionID
		defer func() {
			_ = cleanupBashOutputSessionDir(sessionID)
			_ = cleanupToolOutputSessionDir(sessionID)
		}()
	}
	subRes, err := rt.opts.subMgr.Dispatch(ctx, subReq)
	if err != nil {
		return loaded, err
	}
	summary := toolbuiltin.FormatSubagentOutput(subRes)
	loaded.Output = summary
	if loaded.Metadata == nil {
		loaded.Metadata = map[string]any{}
	}
	loaded.Metadata["subagent"] = subRes.Subagent
	loaded.Metadata["subagent_metadata"] = subRes.Metadata
	return loaded, nil
}

func isolatedSkillSessionID(loaded skills.Result, req *Request) string {
	parts := []string{"skill"}
	if req != nil {
		if id := strings.TrimSpace(req.SessionID); id != "" {
			parts = append(parts, id)
		}
		if id := strings.TrimSpace(req.RequestID); id != "" {
			parts = append(parts, id)
		}
	}
	if name := strings.TrimSpace(loaded.Skill); name != "" {
		parts = append(parts, name)
	}
	return sanitizePathComponent(strings.Join(parts, "-"))
}

func (rt *Runtime) executeSubagent(ctx context.Context, prompt string, activation skills.ActivationContext, req *Request) (*subagents.Result, string, error) {
	if req == nil {
		return nil, prompt, nil
	}

	def, builtin := applySubagentTarget(req)
	if rt.opts.subMgr == nil {
		return nil, prompt, nil
	}
	meta := map[string]any{
		"entrypoint": req.Mode.EntryPoint,
	}
	if len(req.Metadata) > 0 {
		for k, v := range req.Metadata {
			meta[k] = v
		}
	}
	if session := strings.TrimSpace(req.SessionID); session != "" {
		meta["session_id"] = session
	}
	request := subagents.Request{
		Target:        req.TargetSubagent,
		Instruction:   prompt,
		Activation:    activation,
		ToolWhitelist: cloneStrings(req.ToolWhitelist),
		Metadata:      meta,
	}
	dispatchCtx := ctx
	if dispatchCtx == nil {
		dispatchCtx = context.Background()
	}
	if subCtx, ok := buildSubagentContext(*req, def, builtin); ok {
		dispatchCtx = subagents.WithContext(dispatchCtx, subCtx)
	}
	res, err := rt.opts.subMgr.Dispatch(dispatchCtx, request)
	if err != nil {
		if errors.Is(err, subagents.ErrNoMatchingSubagent) && req.TargetSubagent == "" {
			return nil, prompt, nil
		}
		return nil, "", err
	}
	text := fmt.Sprint(res.Output)
	if strings.TrimSpace(text) != "" {
		prompt = strings.TrimSpace(text)
	}
	prompt = applyPromptMetadata(prompt, res.Metadata)
	mergeTags(req, res.Metadata)
	applyCommandMetadata(req, res.Metadata)
	return &res, prompt, nil
}

// selectModelForSubagent returns the appropriate model for the given subagent type.
// Priority: 1) Request.Model override, 2) SubagentModelMapping, 3) default Model.
// Returns the selected model and the tier used (empty string if default).
func (rt *Runtime) selectModelForSubagent(subagentType string, requestTier ModelTier) (model.Model, ModelTier) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	// Priority 1: Request-level override (方案 C)
	if requestTier != "" {
		if m, ok := rt.opts.ModelPool[requestTier]; ok && m != nil {
			return m, requestTier
		}
	}

	// Priority 2: Subagent type mapping (方案 A)
	if rt.opts.SubagentModelMapping != nil {
		canonical := strings.ToLower(strings.TrimSpace(subagentType))
		if tier, ok := rt.opts.SubagentModelMapping[canonical]; ok {
			if rt.opts.ModelPool != nil {
				if m, ok := rt.opts.ModelPool[tier]; ok && m != nil {
					return m, tier
				}
			}
		}
	}

	// Priority 3: Default model
	return rt.opts.Model, ""
}

func (rt *Runtime) newTrimmer() *message.Trimmer {
	if rt.opts.TokenLimit <= 0 {
		return nil
	}
	return message.NewTrimmer(rt.opts.TokenLimit, nil)
}
