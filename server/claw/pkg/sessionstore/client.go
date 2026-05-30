package sessionstore

import (
	"net/http"

	"icoo_claw/common/sessionproto"
)

// Client is a compatibility alias. New shared callers should prefer
// sessionproto.Client directly.
type Client = sessionproto.Client

func NewClient(baseURL string, httpClient *http.Client) *Client {
	return sessionproto.NewClient(baseURL, httpClient)
}
