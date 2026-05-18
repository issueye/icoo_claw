package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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

func (c *SessionStoreClient) ListSessions(ctx context.Context, opts ListSessionsOptions) ([]SessionStoreSession, error) {
	values := url.Values{}
	if opts.UserID != "" {
		values.Set("user_id", opts.UserID)
	}
	if opts.AgentID != "" {
		values.Set("agent_id", opts.AgentID)
	}
	if opts.Status != "" {
		values.Set("status", opts.Status)
	}
	if opts.Offset > 0 {
		values.Set("offset", strconv.Itoa(opts.Offset))
	}
	if opts.Limit > 0 {
		values.Set("limit", strconv.Itoa(opts.Limit))
	}
	path := "/v1/sessions"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var out struct {
		Sessions []SessionStoreSession `json:"sessions"`
	}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, http.StatusOK, &out); err != nil {
		return nil, err
	}
	return out.Sessions, nil
}

func (c *SessionStoreClient) ListMessages(ctx context.Context, sessionID string) ([]dto.SessionMessage, error) {
	var out struct {
		Messages []dto.SessionMessage `json:"messages"`
		Revision int64                `json:"revision"`
	}
	path := fmt.Sprintf("/v1/sessions/%s/messages?limit=0", sessionID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, http.StatusOK, &out); err != nil {
		return nil, err
	}
	return out.Messages, nil
}

func (c *SessionStoreClient) ListRunEvents(ctx context.Context, sessionID string, runID string) ([]SessionRunEvent, error) {
	var out struct {
		Events []SessionRunEvent `json:"events"`
	}
	path := fmt.Sprintf("/v1/sessions/%s/runs/%s/events?limit=0", sessionID, runID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, http.StatusOK, &out); err != nil {
		return nil, err
	}
	return out.Events, nil
}

func (c *SessionStoreClient) AppendRunEvents(ctx context.Context, sessionID string, runID string, events []SessionRunEvent) error {
	path := fmt.Sprintf("/v1/sessions/%s/runs/%s/events", sessionID, runID)
	return c.doJSON(ctx, http.MethodPost, path, struct {
		Events []SessionRunEvent `json:"events"`
	}{Events: events}, http.StatusNoContent, nil)
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
		return newHTTPError("session_store", method, path, resp.StatusCode, payload)
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

type ListSessionsOptions struct {
	UserID  string
	AgentID string
	Status  string
	Offset  int
	Limit   int
}

type SessionStoreSession struct {
	SessionID string         `json:"session_id"`
	UserID    string         `json:"user_id,omitempty"`
	AgentID   string         `json:"agent_id,omitempty"`
	Title     string         `json:"title,omitempty"`
	Status    string         `json:"status"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Revision  int64          `json:"revision"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type SessionRunEvent struct {
	ID        string         `json:"id"`
	RunID     string         `json:"run_id,omitempty"`
	Type      string         `json:"type"`
	Sequence  int64          `json:"sequence"`
	Payload   map[string]any `json:"payload,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}
