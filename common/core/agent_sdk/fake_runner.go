package agent_sdk

import (
	"context"
	"fmt"

	sdkmessage "icoo_claw/common/core/agent_sdk/sdk/message"
)

type FakeRunner struct {
	history *HistoryAdapter
}

func NewFakeRunner(history ...*HistoryAdapter) *FakeRunner {
	var adapter *HistoryAdapter
	if len(history) > 0 {
		adapter = history[0]
	}
	return &FakeRunner{history: adapter}
}

func (r *FakeRunner) Run(ctx context.Context, req RunRequest) (*RunResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	output := fmt.Sprintf("fake agent response: %s", req.Prompt)
	if err := r.save(ctx, req.SessionID, req.Prompt, output); err != nil {
		return nil, err
	}
	return &RunResponse{
		SessionID:  req.SessionID,
		RequestID:  req.RequestID,
		Output:     output,
		StopReason: "end_turn",
	}, nil
}

func (r *FakeRunner) RunStream(ctx context.Context, req RunRequest) (<-chan StreamEvent, error) {
	out := make(chan StreamEvent, 3)
	go func() {
		defer close(out)
		output := "fake agent response: " + req.Prompt
		events := []StreamEvent{
			{
				Type:      StreamEventSessionUpdate,
				SessionID: req.SessionID,
				RequestID: req.RequestID,
				Update:    &SessionUpdate{SessionUpdate: "agent_message_chunk", Content: &ContentBlock{Type: "text", Text: output}},
			},
			{Type: StreamEventSessionCompleted, SessionID: req.SessionID, RequestID: req.RequestID, StopReason: "end_turn"},
		}
		for _, event := range events {
			select {
			case <-ctx.Done():
				return
			case out <- event:
			}
		}
		_ = r.save(context.Background(), req.SessionID, req.Prompt, output)
	}()
	return out, nil
}

func (r *FakeRunner) save(ctx context.Context, sessionID string, prompt string, output string) error {
	if r == nil || r.history == nil {
		return nil
	}
	messages, err := r.history.Load(ctx, sessionID)
	if err != nil {
		return err
	}
	messages = append(messages,
		sdkmessage.Message{Role: "user", Content: prompt},
		sdkmessage.Message{Role: "assistant", Content: output},
	)
	return r.history.SaveSnapshot(ctx, sessionID, messages)
}
