package api

import (
	"context"
	"strings"
	"testing"

	"icoo_claw/common/core/agent_sdk/model"
	"icoo_claw/common/core/agent_sdk/runtime/skills"
)

func TestSkillContextDoesNotPolluteMainHistory(t *testing.T) {
	mdl := &captureMainPromptModel{}
	rt, err := New(context.Background(), Options{
		ProjectRoot:   t.TempDir(),
		Model:         mdl,
		MaxIterations: 3,
		Skills: []SkillRegistration{{
			Definition: skills.Definition{Name: "weather-zh", Description: "天气"},
			Handler: skills.HandlerFunc(func(context.Context, skills.ActivationContext) (skills.Result, error) {
				return skills.Result{
					Skill:  "weather-zh",
					Output: map[string]any{"body": "查询天气后返回摘要"},
				}, nil
			}),
		}},
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close() }()

	resp, err := rt.Run(context.Background(), Request{
		SessionID: "sess_1",
		Prompt:    "今天成都的天气",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if resp == nil || len(resp.SkillResults) != 1 {
		t.Fatalf("skill results = %#v", resp)
	}
	if !strings.Contains(mdl.mainPrompt, "skill-context: 今天成都晴") {
		t.Fatalf("main model prompt missing skill result: %q", mdl.mainPrompt)
	}

	history, ok := rt.SessionHistory("sess_1")
	if !ok {
		t.Fatalf("history missing")
	}
	if len(history) == 0 || history[0].Content != "今天成都的天气" {
		t.Fatalf("history first message = %#v, want original user prompt", history)
	}
	if strings.Contains(history[0].Content, "skill-context") {
		t.Fatalf("skill result leaked into main history: %#v", history[0])
	}
}

type captureMainPromptModel struct {
	mainPrompt string
}

func (m *captureMainPromptModel) Complete(context.Context, model.Request) (*model.Response, error) {
	return nil, nil
}

func (m *captureMainPromptModel) CompleteStream(_ context.Context, req model.Request, cb model.StreamHandler) error {
	if len(req.Messages) > 0 && strings.Contains(req.Messages[0].Content, "Execute the loaded skill") {
		return cb(model.StreamResult{
			Final: true,
			Response: &model.Response{
				Message:    model.Message{Role: "assistant", Content: "skill-context: 今天成都晴"},
				StopReason: "end_turn",
			},
		})
	}
	if len(req.Messages) > 0 {
		m.mainPrompt = req.Messages[len(req.Messages)-1].Content
	}
	return cb(model.StreamResult{
		Final: true,
		Response: &model.Response{
			Message:    model.Message{Role: "assistant", Content: "最终回答"},
			StopReason: "end_turn",
		},
	})
}
