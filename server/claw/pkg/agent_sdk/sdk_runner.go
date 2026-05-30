package agent_sdk

import (
	"context"
	"errors"

	"icoo_claw/common/agentproto/agentruntime"
	"icoo_claw/common/core/agent_sdk/api"
)

type SDKRunner struct {
	factory *RuntimeFactory
	history *HistoryAdapter
}

func NewSDKRunner(factory *RuntimeFactory, history *HistoryAdapter) *SDKRunner {
	return &SDKRunner{factory: factory, history: history}
}

func (r *SDKRunner) Run(ctx context.Context, req RunRequest) (*RunResponse, error) {
	if r == nil || r.factory == nil {
		return nil, errors.New("agent sdk runner is not configured")
	}

	rt, err := r.factory.New(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rt.Close() }()

	resp, err := rt.Run(ctx, api.Request{
		SessionID:     req.SessionID,
		RequestID:     req.RequestID,
		Prompt:        req.Prompt,
		ToolWhitelist: req.ToolWhitelist,
		ForceSkills:   req.ForceSkills,
		Metadata:      req.Metadata,
	})
	if err != nil {
		return nil, mapRuntimeError(err)
	}
	if err := r.saveSnapshot(ctx, rt, req.SessionID); err != nil {
		return nil, err
	}
	if resp == nil || resp.Result == nil {
		return nil, errors.New("agent runtime returned empty response")
	}

	return &RunResponse{
		SessionID:  req.SessionID,
		RequestID:  resp.RequestID,
		Output:     resp.Result.Output,
		StopReason: resp.Result.StopReason,
	}, nil
}

func (r *SDKRunner) RunStream(ctx context.Context, req RunRequest) (<-chan StreamEvent, error) {
	if r == nil || r.factory == nil {
		return nil, errors.New("agent sdk runner is not configured")
	}

	rt, err := r.factory.New(ctx, req)
	if err != nil {
		return nil, err
	}

	events, err := rt.RunStream(ctx, api.Request{
		SessionID:     req.SessionID,
		RequestID:     req.RequestID,
		Prompt:        req.Prompt,
		ToolWhitelist: req.ToolWhitelist,
		ForceSkills:   req.ForceSkills,
		Metadata:      req.Metadata,
	})
	if err != nil {
		_ = rt.Close()
		return nil, mapRuntimeError(err)
	}

	out := make(chan StreamEvent, 128)
	go func() {
		defer close(out)
		defer func() { _ = rt.Close() }()
		snapshotSaved := false
		defer func() {
			if !snapshotSaved {
				_ = r.saveSnapshot(context.Background(), rt, req.SessionID)
			}
		}()

		for event := range events {
			mapped := agentruntime.MapStreamEvent(event, req.SessionID, req.RequestID)
			if mapped.Type == StreamEventSessionCompleted {
				if err := r.saveSnapshot(context.Background(), rt, req.SessionID); err != nil {
					out <- StreamEvent{
						Type:      StreamEventSessionError,
						SessionID: req.SessionID,
						RequestID: req.RequestID,
						Error:     &StreamError{Code: "history_save_failed", Message: err.Error()},
					}
					return
				}
				snapshotSaved = true
			}
			out <- mapped
		}
	}()

	return out, nil
}

func (r *SDKRunner) saveSnapshot(ctx context.Context, rt *api.Runtime, sessionID string) error {
	if r.history == nil {
		return nil
	}
	snapshot, ok := rt.SessionHistory(sessionID)
	if !ok {
		return nil
	}
	return r.history.SaveSnapshot(ctx, sessionID, snapshot)
}

func normalizeStopReason(value string) string {
	switch value {
	case "", "stop_sequence":
		return "end_turn"
	default:
		return value
	}
}

func mapRuntimeError(err error) error {
	if errors.Is(err, api.ErrConcurrentExecution) {
		return ErrSessionBusy
	}
	return err
}

var ErrSessionBusy = errors.New("session busy")
