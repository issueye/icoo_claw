package api

import (
	"context"
	"errors"
	"testing"

	"icoo_claw/common/core/agent_sdk/config"
	"icoo_claw/common/core/agent_sdk/model"
	"icoo_claw/common/core/agent_sdk/tool"
)

func TestPermissionEvaluatorDenyOverridesAllow(t *testing.T) {
	evaluator := newTestPermissionEvaluator(t, &config.PermissionsConfig{
		DefaultMode: "bypassPermissions",
		Allow:       []string{"bash(*)"},
		Deny:        []string{"bash(rm*)"},
	}, nil)

	decision, err := evaluator.Evaluate(context.Background(), model.ToolCall{
		Name:      "bash",
		Arguments: map[string]any{"command": "rm -rf tmp"},
	})
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("Evaluate err = %v, want permission denied", err)
	}
	if decision.Allowed || decision.Mode != "deny" {
		t.Fatalf("decision = %#v, want deny", decision)
	}
}

func TestPermissionEvaluatorAllowRuleBypassesAskDefault(t *testing.T) {
	evaluator := newTestPermissionEvaluator(t, &config.PermissionsConfig{
		DefaultMode: "askBeforeRunningTools",
		Allow:       []string{"read(*)"},
	}, nil)

	decision, err := evaluator.Evaluate(context.Background(), model.ToolCall{
		Name:      "read",
		Arguments: map[string]any{"file_path": "README.md"},
	})
	if err != nil {
		t.Fatalf("Evaluate err = %v, want allow", err)
	}
	if !decision.Allowed || decision.Mode != "allow" {
		t.Fatalf("decision = %#v, want allow", decision)
	}
}

func TestPermissionEvaluatorAskRequiresPrompter(t *testing.T) {
	evaluator := newTestPermissionEvaluator(t, &config.PermissionsConfig{
		DefaultMode: "askBeforeRunningTools",
	}, nil)

	decision, err := evaluator.Evaluate(context.Background(), model.ToolCall{Name: "bash"})
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("Evaluate err = %v, want permission denied", err)
	}
	if decision.Allowed || decision.Reason == "" {
		t.Fatalf("decision = %#v, want denied with reason", decision)
	}
}

func TestPermissionEvaluatorAskPrompterApproves(t *testing.T) {
	var seen PermissionRequest
	evaluator := newTestPermissionEvaluator(t, &config.PermissionsConfig{
		DefaultMode: "askBeforeRunningTools",
	}, PermissionPrompterFunc(func(_ context.Context, req PermissionRequest) (bool, error) {
		seen = req
		return true, nil
	}))

	decision, err := evaluator.Evaluate(context.Background(), model.ToolCall{
		Name:      "bash",
		Arguments: map[string]any{"command": "go test ./..."},
	})
	if err != nil {
		t.Fatalf("Evaluate err = %v, want allow", err)
	}
	if !decision.Allowed || decision.Reason != "permission prompt approved" {
		t.Fatalf("decision = %#v, want approved", decision)
	}
	if seen.Target != "go test ./..." || seen.Mode != "askBeforeRunningTools" {
		t.Fatalf("prompt request = %#v", seen)
	}
}

func TestPermissionEvaluatorAcceptReadOnly(t *testing.T) {
	evaluator := newTestPermissionEvaluator(t, &config.PermissionsConfig{
		DefaultMode: "acceptReadOnly",
	}, nil)

	decision, err := evaluator.Evaluate(context.Background(), model.ToolCall{Name: "read"})
	if err != nil {
		t.Fatalf("read err = %v, want allow", err)
	}
	if !decision.Allowed || decision.Mode != "acceptReadOnly" {
		t.Fatalf("decision = %#v, want acceptReadOnly", decision)
	}
	decision, err = evaluator.Evaluate(context.Background(), model.ToolCall{Name: "bash"})
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("bash err = %v, want permission denied", err)
	}
	if decision.Allowed {
		t.Fatalf("decision = %#v, want denied", decision)
	}
}

func newTestPermissionEvaluator(t *testing.T, permissions *config.PermissionsConfig, prompter PermissionPrompter) *permissionEvaluator {
	t.Helper()
	registry := tool.NewRegistry()
	for _, impl := range []tool.Tool{
		permissionTestTool{name: "read", meta: tool.Metadata{IsReadOnly: true}},
		permissionTestTool{name: "bash", meta: tool.Metadata{IsDestructive: true}},
	} {
		if err := registry.Register(impl); err != nil {
			t.Fatalf("register %s: %v", impl.Name(), err)
		}
	}
	return newPermissionEvaluator(&config.Settings{Permissions: permissions}, registry, prompter)
}

type permissionTestTool struct {
	name string
	meta tool.Metadata
}

func (t permissionTestTool) Name() string { return t.name }

func (t permissionTestTool) Description() string { return t.name }

func (t permissionTestTool) Schema() *tool.JSONSchema { return &tool.JSONSchema{Type: "object"} }

func (t permissionTestTool) Metadata() tool.Metadata { return t.meta }

func (t permissionTestTool) Execute(context.Context, map[string]interface{}) (*tool.ToolResult, error) {
	return &tool.ToolResult{Success: true}, nil
}
