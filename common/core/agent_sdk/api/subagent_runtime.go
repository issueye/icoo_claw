package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"icoo_claw/common/core/agent_sdk/message"
	"icoo_claw/common/core/agent_sdk/middleware"
	"icoo_claw/common/core/agent_sdk/model"
	"icoo_claw/common/core/agent_sdk/runtime/subagents"
)

func (rt *Runtime) bindRuntimeSubagents() {
	if rt == nil || rt.opts.subMgr == nil {
		return
	}
	for _, def := range rt.opts.subMgr.List() {
		handler := rt.opts.subMgr.Handler(def.Name)
		runtimeHandler, ok := handler.(*runtimeSubagentHandler)
		if !ok {
			continue
		}
		runtimeHandler.bind(rt)
	}
}

func (rt *Runtime) runSubagent(ctx context.Context, subCtx subagents.Context, req subagents.Request) (subagents.Result, error) {
	if rt == nil {
		return subagents.Result{}, ErrRuntimeClosed
	}
	if strings.TrimSpace(req.Instruction) == "" {
		return subagents.Result{}, subagents.ErrEmptyInstruction
	}

	sessionID := strings.TrimSpace(subCtx.SessionID)
	if sessionID == "" {
		sessionID = sessionIDFromMetadata(req.Metadata)
	}
	if sessionID == "" {
		sessionID = defaultSessionID(rt.opts.modeContext().EntryPoint)
	}

	history := message.NewHistory()
	history.Append(message.Message{Role: "user", Content: strings.TrimSpace(req.Instruction)})

	allow := subagentToolAllow(rt, subCtx.ToolWhitelist, req.Target)
	toolExec := &runtimeToolExecutor{
		executor:  rt.executor,
		hooks:     &runtimeHookAdapter{executor: rt.hooks, recorder: defaultHookRecorder(), disableSafetyHook: rt.opts.DisableSafetyHook},
		history:   history,
		allow:     allow,
		root:      rt.sbRoot,
		host:      "localhost",
		sessionID: sessionID,
		deferred:  rt.deferred,
		perm:      rt.perm,
	}
	chain := middleware.NewChain(rt.opts.Middleware, middleware.WithTimeout(rt.opts.MiddlewareTimeout))
	selectedModel, modelOverride := rt.selectRuntimeSubagentModel(req.Target, subCtx.Model)
	if selectedModel == nil {
		return subagents.Result{}, errors.New("subagent model is nil")
	}
	prompt := subagentSystemPrompt(rt.systemPromptForSession(sessionID, allow), req.Target)

	for iteration := 0; ; iteration++ {
		if err := ctx.Err(); err != nil {
			return subagents.Result{}, err
		}
		if rt.opts.MaxIterations > 0 && iteration >= rt.opts.MaxIterations {
			return subagents.Result{}, ErrMaxIterations
		}

		modelReq := model.Request{
			Messages:          convertMessages(history.All()),
			Tools:             availableToolsForSession(rt.registry, allow, rt.deferred, sessionID),
			System:            prompt,
			Model:             modelOverride,
			SessionID:         sessionID,
			EnablePromptCache: rt.opts.DefaultEnableCache,
		}
		resp, err := completeViaStream(ctx, selectedModel, modelReq, rt.opts.StreamStall.withDefaults(), streamEmitFromContext(ctx) != nil)
		if err != nil {
			return subagents.Result{}, err
		}
		if resp == nil {
			return subagents.Result{}, errors.New("subagent model returned nil response")
		}

		history.Append(message.Message{
			Role:             resp.Message.Role,
			Content:          strings.TrimSpace(resp.Message.Content),
			ReasoningContent: resp.Message.ReasoningContent,
			ToolCalls:        convertModelToolCalls(resp.Message.ToolCalls),
		})

		if len(resp.Message.ToolCalls) == 0 {
			return subagents.Result{
				Output: strings.TrimSpace(resp.Message.Content),
				Metadata: map[string]any{
					"stop_reason": resp.StopReason,
					"usage":       resp.Usage,
				},
			}, nil
		}
		if err := rt.executeToolCalls(ctx, resp.Message.ToolCalls, toolExec, chain, &middleware.State{Values: map[string]any{}}, rt.opts.tracer, nil, Request{SessionID: sessionID}); err != nil {
			return subagents.Result{}, err
		}
	}
}

func (rt *Runtime) selectRuntimeSubagentModel(target, preferred string) (model.Model, string) {
	if rt == nil {
		return nil, ""
	}
	preferred = strings.TrimSpace(preferred)
	if preferred != "" {
		if _, ok := subagentModelAliasTier(preferred); !ok {
			return rt.opts.Model, preferred
		}
	}
	if tier, ok := rt.subagentMappingTier(target); ok {
		if mdl := rt.modelForTier(tier); mdl != nil {
			return mdl, ""
		}
	}
	if tier, ok := subagentModelAliasTier(preferred); ok {
		if mdl := rt.modelForTier(tier); mdl != nil {
			return mdl, ""
		}
	}
	return rt.opts.Model, normalizedSubagentModel(preferred)
}

func (rt *Runtime) subagentMappingTier(target string) (ModelTier, bool) {
	if rt == nil || len(rt.opts.SubagentModelMapping) == 0 {
		return "", false
	}
	key := strings.ToLower(strings.TrimSpace(target))
	if key == "" {
		return "", false
	}
	tier, ok := rt.opts.SubagentModelMapping[key]
	return tier, ok
}

func (rt *Runtime) modelForTier(tier ModelTier) model.Model {
	if rt == nil || tier == "" || len(rt.opts.ModelPool) == 0 {
		return nil
	}
	return rt.opts.ModelPool[tier]
}

func subagentModelAliasTier(value string) (ModelTier, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case subagents.ModelHaiku:
		return ModelTierLow, true
	case subagents.ModelSonnet:
		return ModelTierMid, true
	case "opus":
		return ModelTierHigh, true
	default:
		return "", false
	}
}

func subagentSystemPrompt(base, target string) string {
	extra := "You are running as a subagent. Complete the delegated task independently. Return only a concise final result for the parent agent."
	if strings.TrimSpace(target) == subagents.TypeSkillExecutor {
		extra = "You are running as the skill execution subagent. Execute the provided skill instructions for the user's request. Use tools when needed. Return only the final user-facing result or a concise summary."
	}
	base = strings.TrimSpace(base)
	if base == "" {
		return extra
	}
	return base + "\n\n" + extra
}

func normalizedSubagentModel(value string) string {
	value = strings.TrimSpace(value)
	switch value {
	case subagents.ModelSonnet, subagents.ModelHaiku:
		return ""
	default:
		return value
	}
}

func subagentToolAllow(rt *Runtime, whitelist []string, target string) map[string]struct{} {
	allow := toLowerSet(whitelist)
	if strings.TrimSpace(target) != subagents.TypeSkillExecutor {
		return allow
	}
	if len(allow) == 0 {
		allow = map[string]struct{}{}
		if rt != nil && rt.registry != nil {
			for _, impl := range rt.registry.List() {
				if impl == nil {
					continue
				}
				name := canonicalToolName(impl.Name())
				if name == "" || name == "skill" {
					continue
				}
				allow[name] = struct{}{}
			}
		}
		return allow
	}
	delete(allow, "skill")
	return allow
}

func convertModelToolCalls(calls []model.ToolCall) []message.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]message.ToolCall, len(calls))
	for i, call := range calls {
		out[i] = message.ToolCall{
			ID:        call.ID,
			Name:      call.Name,
			Arguments: cloneArguments(call.Arguments),
			Result:    call.Result,
		}
	}
	return out
}

func sessionIDFromMetadata(meta map[string]any) string {
	if len(meta) == 0 {
		return ""
	}
	if value, ok := meta["session_id"]; ok {
		return strings.TrimSpace(fmt.Sprint(value))
	}
	return ""
}
