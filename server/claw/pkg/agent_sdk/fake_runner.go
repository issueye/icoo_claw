package agent_sdk

import (
	"context"
	"fmt"
)

type FakeRunner struct{}

func NewFakeRunner() *FakeRunner {
	return &FakeRunner{}
}

func (r *FakeRunner) Run(ctx context.Context, req RunRequest) (*RunResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &RunResponse{
		SessionID:  req.SessionID,
		RequestID:  req.RequestID,
		Output:     fmt.Sprintf("fake agent response: %s", req.Prompt),
		StopReason: "end_turn",
	}, nil
}

func (r *FakeRunner) RunStream(ctx context.Context, req RunRequest) (<-chan StreamEvent, error) {
	out := make(chan StreamEvent, 3)
	go func() {
		defer close(out)
		events := []StreamEvent{
			{Type: "agent_start", SessionID: req.SessionID, RequestID: req.RequestID},
			{Type: "content_block_delta", SessionID: req.SessionID, RequestID: req.RequestID, Output: "fake agent response: " + req.Prompt},
			{Type: "message_stop", SessionID: req.SessionID, RequestID: req.RequestID},
		}
		for _, event := range events {
			select {
			case <-ctx.Done():
				return
			case out <- event:
			}
		}
	}()
	return out, nil
}
