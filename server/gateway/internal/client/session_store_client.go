package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"icoo_claw/server/gateway/internal/dto"
)

type SessionStoreClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewSessionStoreClient(baseURL string, httpClient *http.Client) *SessionStoreClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &SessionStoreClient{baseURL: strings.TrimRight(baseURL, "/"), httpClient: httpClient}
}

func (c *SessionStoreClient) CreateSession(ctx context.Context, req CreateSessionRequest) error {
	return c.doJSON(ctx, http.MethodPost, "/v1/sessions", req, http.StatusCreated, nil)
}

func (c *SessionStoreClient) ListMessages(ctx context.Context, sessionID string) ([]dto.SessionMessage, error) {
	var out struct {
		Messages []dto.SessionMessage `json:"messages"`
	}
	path := fmt.Sprintf("/v1/sessions/%s/messages?limit=0", sessionID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, http.StatusOK, &out); err != nil {
		return nil, err
	}
	return out.Messages, nil
}

func (c *SessionStoreClient) doJSON(ctx context.Context, method, path string, body any, expected int, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != expected {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("session store %s %s: status %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type CreateSessionRequest struct {
	SessionID string         `json:"session_id"`
	UserID    string         `json:"user_id,omitempty"`
	AgentID   string         `json:"agent_id,omitempty"`
	Title     string         `json:"title,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}
