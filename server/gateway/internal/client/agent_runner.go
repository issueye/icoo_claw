package client

import "context"

type AgentRunner struct {
	http *ClawClient
	acp  *ACPRegistry
}

func NewAgentRunner(http *ClawClient, acp *ACPRegistry) *AgentRunner {
	return &AgentRunner{http: http, acp: acp}
}

func (r *AgentRunner) Run(ctx context.Context, baseURL string, req RunRequest) (*RunResponse, error) {
	if IsACPBaseURL(baseURL) {
		return r.acp.Run(ctx, baseURL, req)
	}
	return r.http.Run(ctx, baseURL, req)
}

func (r *AgentRunner) Stream(ctx context.Context, baseURL string, req RunRequest) (<-chan StreamEvent, error) {
	if IsACPBaseURL(baseURL) {
		return r.acp.Stream(ctx, baseURL, req)
	}
	return r.http.Stream(ctx, baseURL, req)
}
