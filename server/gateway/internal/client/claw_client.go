package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

type ClawClient struct {
	httpClient *http.Client
	token      string
}

func NewClawClient(httpClient *http.Client, token ...string) *ClawClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 120 * time.Second}
	}
	resolved := ""
	if len(token) > 0 {
		resolved = token[0]
	}
	return &ClawClient{httpClient: httpClient, token: resolved}
}

func (c *ClawClient) Run(ctx context.Context, baseURL string, req RunRequest) (*RunResponse, error) {
	var out RunResponse
	if err := c.doJSON(ctx, baseURL, "/internal/agent/run", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *ClawClient) Stream(ctx context.Context, baseURL string, req RunRequest) (<-chan StreamEvent, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/internal/agent/run/stream", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		httpReq.Header.Set("X-Internal-Token", c.token)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, newHTTPError("claw", http.MethodPost, "/internal/agent/run/stream", resp.StatusCode, body)
	}

	out := make(chan StreamEvent, 128)
	go func() {
		defer close(out)
		defer resp.Body.Close()
		reader := newSSELineReader(resp.Body)
		for {
			line, err := reader.readLine()
			if err != nil {
				if err != io.EOF {
					out <- StreamEvent{Type: "error", Output: err.Error()}
				}
				return
			}
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			raw := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if raw == "" {
				continue
			}
			var event StreamEvent
			if err := json.Unmarshal([]byte(raw), &event); err == nil {
				out <- event
			}
		}
	}()
	return out, nil
}

type sseLineReader struct {
	reader io.Reader
	buf    []byte
}

func newSSELineReader(reader io.Reader) *sseLineReader {
	return &sseLineReader{reader: reader, buf: make([]byte, 0, 4096)}
}

func (r *sseLineReader) readLine() (string, error) {
	tmp := make([]byte, 1024)
	for {
		for i, b := range r.buf {
			if b == '\n' {
				line := strings.TrimRight(string(r.buf[:i]), "\r")
				r.buf = append([]byte(nil), r.buf[i+1:]...)
				return line, nil
			}
		}
		n, err := r.reader.Read(tmp)
		if n > 0 {
			r.buf = append(r.buf, tmp[:n]...)
		}
		if err != nil {
			if err == io.EOF && len(r.buf) > 0 {
				line := strings.TrimRight(string(r.buf), "\r")
				r.buf = r.buf[:0]
				return line, nil
			}
			return "", err
		}
	}
}

func (c *ClawClient) doJSON(ctx context.Context, baseURL, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("X-Internal-Token", c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return newHTTPError("claw", http.MethodPost, path, resp.StatusCode, body)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type RunRequest struct {
	SessionID     string         `json:"session_id"`
	RequestID     string         `json:"request_id,omitempty"`
	Prompt        string         `json:"prompt"`
	Agent         map[string]any `json:"agent"`
	ToolWhitelist []string       `json:"tool_whitelist,omitempty"`
	ForceSkills   []string       `json:"force_skills,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type RunResponse struct {
	SessionID  string `json:"session_id"`
	RequestID  string `json:"request_id,omitempty"`
	Output     string `json:"output"`
	StopReason string `json:"stop_reason"`
}

type StreamEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	RequestID string `json:"request_id,omitempty"`
	Output    string `json:"output,omitempty"`
}
