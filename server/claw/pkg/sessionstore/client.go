package sessionstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func (c *Client) ListMessages(ctx context.Context, sessionID string) ([]Message, error) {
	var out MessagesResponse
	path := fmt.Sprintf("/v1/sessions/%s/messages?limit=0", sessionID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, http.StatusOK, &out); err != nil {
		return nil, err
	}
	return out.Messages, nil
}

func (c *Client) AppendMessages(ctx context.Context, sessionID string, messages []Message) error {
	path := fmt.Sprintf("/v1/sessions/%s/messages", sessionID)
	return c.doJSON(ctx, http.MethodPost, path, MessagesRequest{Messages: messages}, http.StatusNoContent, nil)
}

func (c *Client) ReplaceMessages(ctx context.Context, sessionID string, messages []Message) error {
	path := fmt.Sprintf("/v1/sessions/%s/messages/snapshot", sessionID)
	return c.doJSON(ctx, http.MethodPut, path, MessagesRequest{Messages: messages}, http.StatusNoContent, nil)
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
		return fmt.Errorf("session store %s %s: status %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
