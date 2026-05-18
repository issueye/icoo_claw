package toolbuiltin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"icoo_claw/server/claw/pkg/agent_sdk/sdk/sandbox"
	"icoo_claw/server/claw/pkg/agent_sdk/sdk/tool"
)

const (
	fetchDefaultTimeout = 15 * time.Second
	fetchMaxTimeout     = 60 * time.Second
	fetchDefaultMaxBody = 256 * 1024
	fetchMaxBody        = 1024 * 1024
	fetchToolDesc       = `Fetch an HTTP or HTTPS URL using GET or HEAD.
The target host is checked against the runtime network allowlist, redirects are rechecked, and response bodies are capped.`
)

var fetchSchema = &tool.JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"url": map[string]interface{}{
			"type":        "string",
			"description": "HTTP or HTTPS URL to fetch.",
		},
		"method": map[string]interface{}{
			"type":        "string",
			"description": "HTTP method: GET or HEAD. Defaults to GET.",
			"enum":        []interface{}{"GET", "HEAD"},
			"default":     "GET",
		},
		"timeout": map[string]interface{}{
			"type":        "number",
			"description": "Timeout in seconds. Defaults to 15, capped at 60.",
		},
		"max_bytes": map[string]interface{}{
			"type":        "integer",
			"description": "Maximum response body bytes to return. Defaults to 262144, capped at 1048576.",
		},
		"user_agent": map[string]interface{}{
			"type":        "string",
			"description": "Optional User-Agent header.",
		},
	},
	Required: []string{"url"},
}

// FetchTool performs sandboxed HTTP(S) reads.
type FetchTool struct {
	policy      sandbox.NetworkPolicy
	client      *http.Client
	timeout     time.Duration
	maxBodySize int
}

// NewFetchTool builds a FetchTool using the default local-only network allowlist.
func NewFetchTool() *FetchTool {
	return NewFetchToolWithNetworkPolicy(sandbox.NewDomainAllowList("localhost", "127.0.0.1", "::1"))
}

// NewFetchToolWithNetworkPolicy builds a FetchTool using a custom network policy.
// A nil policy disables network host checks and should only be used when sandboxing is disabled.
func NewFetchToolWithNetworkPolicy(policy sandbox.NetworkPolicy) *FetchTool {
	t := &FetchTool{
		policy:      policy,
		timeout:     fetchDefaultTimeout,
		maxBodySize: fetchDefaultMaxBody,
	}
	t.client = &http.Client{
		Timeout: t.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("stopped after 5 redirects")
			}
			return t.validateURL(req.URL)
		},
	}
	return t
}

func (f *FetchTool) Name() string { return "fetch" }

func (f *FetchTool) Description() string { return fetchToolDesc }

func (f *FetchTool) Schema() *tool.JSONSchema { return fetchSchema }

func (f *FetchTool) Metadata() tool.Metadata {
	return tool.Metadata{IsReadOnly: true, IsConcurrencySafe: true}
}

func (f *FetchTool) Execute(ctx context.Context, params map[string]interface{}) (*tool.ToolResult, error) {
	if ctx == nil {
		return nil, errors.New("context is nil")
	}
	if f == nil {
		return nil, errors.New("fetch tool is not initialised")
	}
	target, err := parseFetchURL(params)
	if err != nil {
		return nil, err
	}
	if err := f.validateURL(target); err != nil {
		return nil, err
	}
	method, err := parseFetchMethod(params)
	if err != nil {
		return nil, err
	}
	timeout, err := parseFetchTimeout(params, f.timeout)
	if err != nil {
		return nil, err
	}
	maxBytes, err := parseFetchMaxBytes(params, f.maxBodySize)
	if err != nil {
		return nil, err
	}

	callCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(callCtx, method, target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "text/plain, text/html, application/json, application/xml;q=0.9, */*;q=0.8")
	if ua := parseFetchUserAgent(params); ua != "" {
		req.Header.Set("User-Agent", ua)
	} else {
		req.Header.Set("User-Agent", "icoo-claw-agent/1.0")
	}

	client := http.Client{
		Timeout: timeout,
	}
	if f.client != nil {
		client.Transport = f.client.Transport
		client.Jar = f.client.Jar
		client.CheckRedirect = f.client.CheckRedirect
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", target.Redacted(), err)
	}
	defer resp.Body.Close()

	limit := int64(maxBytes) + 1
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	truncated := len(body) > maxBytes
	if truncated {
		body = body[:maxBytes]
	}
	text := string(body)
	if !utf8.Valid(body) {
		text = strings.ToValidUTF8(text, "\uFFFD")
	}

	output := formatFetchOutput(resp, text, truncated)
	return &tool.ToolResult{
		Success: resp.StatusCode >= 200 && resp.StatusCode < 400,
		Output:  output,
		Data: map[string]interface{}{
			"url":            target.String(),
			"final_url":      resp.Request.URL.String(),
			"status_code":    resp.StatusCode,
			"status":         resp.Status,
			"content_type":   resp.Header.Get("Content-Type"),
			"content_length": resp.ContentLength,
			"bytes":          len(body),
			"truncated":      truncated,
		},
	}, nil
}

func (f *FetchTool) validateURL(u *url.URL) error {
	if u == nil {
		return errors.New("url is nil")
	}
	scheme := strings.ToLower(strings.TrimSpace(u.Scheme))
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
	if strings.TrimSpace(u.Host) == "" {
		return errors.New("url host is required")
	}
	if f.policy != nil {
		if err := f.policy.Validate(u.Host); err != nil {
			return fmt.Errorf("network host denied: %w", err)
		}
	}
	return nil
}

func parseFetchURL(params map[string]interface{}) (*url.URL, error) {
	if params == nil {
		return nil, errors.New("params is nil")
	}
	raw, ok := params["url"]
	if !ok {
		return nil, errors.New("url is required")
	}
	value, err := coerceString(raw)
	if err != nil {
		return nil, fmt.Errorf("url must be string: %w", err)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("url cannot be empty")
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	return parsed, nil
}

func parseFetchMethod(params map[string]interface{}) (string, error) {
	if params == nil {
		return http.MethodGet, nil
	}
	raw, ok := params["method"]
	if !ok || raw == nil {
		return http.MethodGet, nil
	}
	value, err := coerceString(raw)
	if err != nil {
		return "", fmt.Errorf("method must be string: %w", err)
	}
	method := strings.ToUpper(strings.TrimSpace(value))
	switch method {
	case "", http.MethodGet:
		return http.MethodGet, nil
	case http.MethodHead:
		return http.MethodHead, nil
	default:
		return "", errors.New("method must be GET or HEAD")
	}
}

func parseFetchTimeout(params map[string]interface{}, fallback time.Duration) (time.Duration, error) {
	if fallback <= 0 {
		fallback = fetchDefaultTimeout
	}
	if params == nil {
		return fallback, nil
	}
	raw, ok := params["timeout"]
	if !ok || raw == nil {
		return fallback, nil
	}
	seconds, err := numberFromParam(raw)
	if err != nil {
		return 0, fmt.Errorf("timeout must be number: %w", err)
	}
	if seconds <= 0 {
		return 0, errors.New("timeout must be > 0")
	}
	timeout := time.Duration(seconds * float64(time.Second))
	if timeout > fetchMaxTimeout {
		return fetchMaxTimeout, nil
	}
	return timeout, nil
}

func parseFetchMaxBytes(params map[string]interface{}, fallback int) (int, error) {
	if fallback <= 0 {
		fallback = fetchDefaultMaxBody
	}
	if params == nil {
		return fallback, nil
	}
	raw, ok := params["max_bytes"]
	if !ok || raw == nil {
		return fallback, nil
	}
	value, err := intFromParam(raw)
	if err != nil {
		return 0, fmt.Errorf("max_bytes must be integer: %w", err)
	}
	if value <= 0 {
		return 0, errors.New("max_bytes must be > 0")
	}
	if value > fetchMaxBody {
		return fetchMaxBody, nil
	}
	return value, nil
}

func parseFetchUserAgent(params map[string]interface{}) string {
	if params == nil {
		return ""
	}
	raw, ok := params["user_agent"]
	if !ok || raw == nil {
		return ""
	}
	value, err := coerceString(raw)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func numberFromParam(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		value := strings.TrimSpace(v)
		if value == "" {
			return 0, errors.New("empty string")
		}
		return strconv.ParseFloat(value, 64)
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}

func formatFetchOutput(resp *http.Response, body string, truncated bool) string {
	if resp == nil {
		return body
	}
	var b strings.Builder
	fmt.Fprintf(&b, "HTTP %s\n", resp.Status)
	if contentType := strings.TrimSpace(resp.Header.Get("Content-Type")); contentType != "" {
		fmt.Fprintf(&b, "Content-Type: %s\n", contentType)
	}
	if finalURL := resp.Request.URL.String(); finalURL != "" {
		fmt.Fprintf(&b, "URL: %s\n", finalURL)
	}
	b.WriteByte('\n')
	b.WriteString(body)
	if truncated {
		b.WriteString("\n... truncated")
	}
	return b.String()
}
