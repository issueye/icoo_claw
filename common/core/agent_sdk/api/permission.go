package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"

	"icoo_claw/common/core/agent_sdk/config"
	"icoo_claw/common/core/agent_sdk/model"
	"icoo_claw/common/core/agent_sdk/tool"
)

var ErrPermissionDenied = errors.New("api: tool use denied by permission")

type PermissionRequest struct {
	ToolCallID string
	ToolName  string
	Arguments map[string]any
	Target    string
	Rule      string
	Mode      string
}

type PermissionDecision struct {
	ToolName string `json:"tool_name,omitempty"`
	Target   string `json:"target,omitempty"`
	Rule     string `json:"rule,omitempty"`
	Mode     string `json:"mode,omitempty"`
	Allowed  bool   `json:"allowed"`
	Reason   string `json:"reason,omitempty"`
}

type PermissionPrompter interface {
	PromptPermission(context.Context, PermissionRequest) (bool, error)
}

type PermissionPrompterFunc func(context.Context, PermissionRequest) (bool, error)

func (fn PermissionPrompterFunc) PromptPermission(ctx context.Context, req PermissionRequest) (bool, error) {
	if fn == nil {
		return false, nil
	}
	return fn(ctx, req)
}

type permissionEvaluator struct {
	cfg      *config.PermissionsConfig
	registry *tool.Registry
	prompter PermissionPrompter

	allow []permissionRule
	ask   []permissionRule
	deny  []permissionRule
}

type permissionRule struct {
	raw       string
	tool      string
	target    string
	hasTarget bool
}

func newPermissionEvaluator(settings *config.Settings, registry *tool.Registry, prompter PermissionPrompter) *permissionEvaluator {
	if settings == nil || settings.Permissions == nil {
		return nil
	}
	cfg := settings.Permissions
	return &permissionEvaluator{
		cfg:      cfg,
		registry: registry,
		prompter: prompter,
		allow:    parsePermissionRules(cfg.Allow),
		ask:      parsePermissionRules(cfg.Ask),
		deny:     parsePermissionRules(cfg.Deny),
	}
}

func (e *permissionEvaluator) Evaluate(ctx context.Context, call model.ToolCall) (PermissionDecision, error) {
	if e == nil || e.cfg == nil {
		return PermissionDecision{ToolName: strings.TrimSpace(call.Name), Allowed: true, Reason: "permissions disabled"}, nil
	}
	req := PermissionRequest{
		ToolCallID: strings.TrimSpace(call.ID),
		ToolName:  strings.TrimSpace(call.Name),
		Arguments: cloneArguments(call.Arguments),
		Target:    permissionTarget(call),
	}
	if rule, ok := firstMatchingPermissionRule(e.deny, call, req.Target); ok {
		req.Rule = rule.raw
		req.Mode = "deny"
		decision := permissionDecision(req, false, "matched deny rule")
		return decision, permissionDenied(req, decision.Reason)
	}
	if rule, ok := firstMatchingPermissionRule(e.ask, call, req.Target); ok {
		req.Rule = rule.raw
		req.Mode = "ask"
		return e.askPermission(ctx, req)
	}
	if rule, ok := firstMatchingPermissionRule(e.allow, call, req.Target); ok {
		req.Rule = rule.raw
		req.Mode = "allow"
		return permissionDecision(req, true, "matched allow rule"), nil
	}
	return e.evaluateDefaultMode(ctx, call, req)
}

func (e *permissionEvaluator) evaluateDefaultMode(ctx context.Context, call model.ToolCall, req PermissionRequest) (PermissionDecision, error) {
	mode := strings.TrimSpace(e.cfg.DefaultMode)
	if mode == "" {
		mode = "askBeforeRunningTools"
	}
	req.Mode = mode
	switch mode {
	case "bypassPermissions":
		if strings.EqualFold(strings.TrimSpace(e.cfg.DisableBypassPermissionsMode), "disable") {
			req.Mode = "askBeforeRunningTools"
			return e.askPermission(ctx, req)
		}
		return permissionDecision(req, true, "bypassPermissions"), nil
	case "acceptReadOnly":
		if e.toolMetadata(call.Name).IsReadOnly {
			return permissionDecision(req, true, "read-only tool accepted"), nil
		}
		return e.askPermission(ctx, req)
	case "acceptEdits":
		meta := e.toolMetadata(call.Name)
		if meta.IsReadOnly || !meta.IsDestructive {
			return permissionDecision(req, true, "non-destructive tool accepted"), nil
		}
		return e.askPermission(ctx, req)
	default:
		return e.askPermission(ctx, req)
	}
}

func (e *permissionEvaluator) askPermission(ctx context.Context, req PermissionRequest) (PermissionDecision, error) {
	if e.prompter == nil {
		decision := permissionDecision(req, false, "permission prompt required")
		return decision, permissionDenied(req, decision.Reason)
	}
	allowed, err := e.prompter.PromptPermission(ctx, req)
	if err != nil {
		decision := permissionDecision(req, false, "permission prompt failed")
		return decision, fmt.Errorf("%w: %s: prompt failed: %v", ErrPermissionDenied, req.ToolName, err)
	}
	if !allowed {
		decision := permissionDecision(req, false, "permission prompt rejected")
		return decision, permissionDenied(req, decision.Reason)
	}
	return permissionDecision(req, true, "permission prompt approved"), nil
}

func (e *permissionEvaluator) toolMetadata(name string) tool.Metadata {
	if e == nil || e.registry == nil {
		return tool.Metadata{}
	}
	impl, err := e.registry.Get(name)
	if err != nil {
		return tool.Metadata{}
	}
	return tool.MetadataOf(impl)
}

func permissionDenied(req PermissionRequest, reason string) error {
	if req.Rule != "" {
		return fmt.Errorf("%w: %s: %s (%s)", ErrPermissionDenied, req.ToolName, reason, req.Rule)
	}
	return fmt.Errorf("%w: %s: %s", ErrPermissionDenied, req.ToolName, reason)
}

func permissionDecision(req PermissionRequest, allowed bool, reason string) PermissionDecision {
	return PermissionDecision{
		ToolName: req.ToolName,
		Target:   req.Target,
		Rule:     req.Rule,
		Mode:     req.Mode,
		Allowed:  allowed,
		Reason:   reason,
	}
}

func parsePermissionRules(raw []string) []permissionRule {
	if len(raw) == 0 {
		return nil
	}
	rules := make([]permissionRule, 0, len(raw))
	for _, entry := range raw {
		rule := parsePermissionRule(entry)
		if rule.raw != "" {
			rules = append(rules, rule)
		}
	}
	return rules
}

func parsePermissionRule(raw string) permissionRule {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return permissionRule{}
	}
	open := strings.IndexRune(raw, '(')
	if open < 0 || !strings.HasSuffix(raw, ")") {
		return permissionRule{raw: raw, tool: canonicalToolName(raw)}
	}
	return permissionRule{
		raw:       raw,
		tool:      canonicalToolName(raw[:open]),
		target:    strings.TrimSpace(raw[open+1 : len(raw)-1]),
		hasTarget: true,
	}
}

func firstMatchingPermissionRule(rules []permissionRule, call model.ToolCall, target string) (permissionRule, bool) {
	for _, rule := range rules {
		if rule.matches(call, target) {
			return rule, true
		}
	}
	return permissionRule{}, false
}

func (r permissionRule) matches(call model.ToolCall, target string) bool {
	toolName := canonicalToolName(call.Name)
	if r.tool != "*" && r.tool != toolName {
		return false
	}
	if !r.hasTarget {
		return true
	}
	return wildcardMatch(r.target, target)
}

func permissionTarget(call model.ToolCall) string {
	if len(call.Arguments) == 0 {
		return ""
	}
	for _, key := range []string{"command", "file_path", "path", "pattern", "query", "url"} {
		if value, ok := call.Arguments[key]; ok {
			return strings.TrimSpace(fmt.Sprint(value))
		}
	}
	data, err := json.Marshal(call.Arguments)
	if err != nil {
		return fmt.Sprint(call.Arguments)
	}
	return string(data)
}

func wildcardMatch(pattern, value string) bool {
	pattern = strings.TrimSpace(pattern)
	value = strings.TrimSpace(value)
	if pattern == "" {
		return value == ""
	}
	if pattern == "*" {
		return true
	}
	if ok, err := path.Match(pattern, value); err == nil && ok {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == value
	}
	parts := strings.Split(pattern, "*")
	pos := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		idx := strings.Index(value[pos:], part)
		if idx < 0 {
			return false
		}
		if i == 0 && !strings.HasPrefix(pattern, "*") && idx != 0 {
			return false
		}
		pos += idx + len(part)
	}
	last := parts[len(parts)-1]
	return last == "" || strings.HasSuffix(value, last)
}
