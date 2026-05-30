package client

import (
	"net/http"

	"icoo_claw/common/agentproto"
)

// ClawClient is a compatibility alias. New shared callers should prefer
// agentproto.HTTPClient directly.
type ClawClient = agentproto.HTTPClient

func NewClawClient(httpClient *http.Client, token ...string) *ClawClient {
	return agentproto.NewHTTPClient(httpClient, token...)
}

type RunRequest = agentproto.RunRequest
type RunResponse = agentproto.RunResponse
type StreamEvent = agentproto.StreamEvent
type SessionUpdate = agentproto.SessionUpdate
type ContentBlock = agentproto.ContentBlock
type ToolCallLocation = agentproto.ToolCallLocation
type UsageUpdate = agentproto.UsageUpdate
type StreamError = agentproto.StreamError
