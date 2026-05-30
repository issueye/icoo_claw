package agentproto

import "context"

type Runner interface {
	Run(ctx context.Context, req RunRequest) (*RunResponse, error)
	RunStream(ctx context.Context, req RunRequest) (<-chan StreamEvent, error)
}
