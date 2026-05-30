package agent_sdk

import (
	"context"

	"icoo_claw/common/agentproto"
)

type Runner interface {
	Run(ctx context.Context, req RunRequest) (*RunResponse, error)
	RunStream(ctx context.Context, req RunRequest) (<-chan StreamEvent, error)
}

type RunRequest = agentproto.RunRequest
type RunResponse = agentproto.RunResponse
type StreamEvent = agentproto.StreamEvent

const (
	StreamEventSessionUpdate    = agentproto.StreamEventSessionUpdate
	StreamEventSessionCompleted = agentproto.StreamEventSessionCompleted
	StreamEventSessionError     = agentproto.StreamEventSessionError
)

type SessionUpdate = agentproto.SessionUpdate
type ContentBlock = agentproto.ContentBlock
type ToolCallLocation = agentproto.ToolCallLocation
type UsageUpdate = agentproto.UsageUpdate
type StreamError = agentproto.StreamError
