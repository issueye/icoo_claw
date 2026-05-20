package sessionstore

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
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (c *Client) CreateSession(ctx context.Context, req CreateSessionRequest) (*Session, error) {
	var out Session
	if err := c.doJSON(ctx, http.MethodPost, "/v1/sessions", req, http.StatusCreated, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListSessions(ctx context.Context, opts ListSessionsOptions) ([]Session, error) {
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
	var out SessionsResponse
	if err := c.doJSON(ctx, http.MethodGet, path, nil, http.StatusOK, &out); err != nil {
		return nil, err
	}
	return out.Sessions, nil
}

func (c *Client) ListMessages(ctx context.Context, sessionID string) ([]Message, error) {
	messages, _, err := c.ListMessagesWithRevision(ctx, sessionID)
	return messages, err
}

func (c *Client) ListMessagesWithRevision(ctx context.Context, sessionID string) ([]Message, int64, error) {
	var out MessagesResponse
	path := fmt.Sprintf("/v1/sessions/%s/messages?limit=0", sessionID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, http.StatusOK, &out); err != nil {
		return nil, 0, err
	}
	return out.Messages, out.Revision, nil
}

func (c *Client) AppendMessages(ctx context.Context, sessionID string, messages []Message) error {
	path := fmt.Sprintf("/v1/sessions/%s/messages", sessionID)
	return c.doJSON(ctx, http.MethodPost, path, MessagesRequest{Messages: messages}, http.StatusNoContent, nil)
}

func (c *Client) ReplaceMessages(ctx context.Context, sessionID string, messages []Message) error {
	return c.ReplaceMessagesWithRevision(ctx, sessionID, messages, nil)
}

func (c *Client) ReplaceMessagesWithRevision(ctx context.Context, sessionID string, messages []Message, expectedRevision *int64) error {
	path := fmt.Sprintf("/v1/sessions/%s/messages/snapshot", sessionID)
	return c.doJSON(ctx, http.MethodPut, path, MessagesRequest{Messages: messages, ExpectedRevision: expectedRevision}, http.StatusNoContent, nil)
}

func (c *Client) ListRuns(ctx context.Context, sessionID string) ([]Run, error) {
	var out RunsResponse
	path := fmt.Sprintf("/v1/sessions/%s/runs?limit=0", sessionID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, http.StatusOK, &out); err != nil {
		return nil, err
	}
	return out.Runs, nil
}

func (c *Client) AppendRuns(ctx context.Context, sessionID string, runs []Run) error {
	path := fmt.Sprintf("/v1/sessions/%s/runs", sessionID)
	return c.doJSON(ctx, http.MethodPost, path, RunsRequest{Runs: runs}, http.StatusNoContent, nil)
}

func (c *Client) ListRunEvents(ctx context.Context, sessionID string, runID string) ([]RunEvent, error) {
	var out RunEventsResponse
	path := fmt.Sprintf("/v1/sessions/%s/runs/%s/events?limit=0", sessionID, runID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, http.StatusOK, &out); err != nil {
		return nil, err
	}
	return out.Events, nil
}

func (c *Client) AppendRunEvents(ctx context.Context, sessionID string, runID string, events []RunEvent) error {
	path := fmt.Sprintf("/v1/sessions/%s/runs/%s/events", sessionID, runID)
	return c.doJSON(ctx, http.MethodPost, path, RunEventsRequest{Events: events}, http.StatusNoContent, nil)
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any, expected int, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
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
		return fmt.Errorf("session api %s %s: status %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
