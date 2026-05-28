package toolbuiltin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"icoo_claw/common/core/agent_sdk/sandbox"
	"icoo_claw/common/core/agent_sdk/tool"

	"golang.org/x/net/html"
)

const (
	webSearchDefaultTimeout = 15 * time.Second
	webSearchMaxTimeout     = 60 * time.Second
	webSearchDefaultLimit   = 5
	webSearchMaxLimit       = 10
	webSearchMaxBody        = 1024 * 1024
	webSearchToolDesc       = `Search the web using DuckDuckGo HTML search.
Returns result titles, URLs, and snippets. The DuckDuckGo host must be allowed by the runtime network allowlist.`
)

var webSearchSchema = &tool.JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"query": map[string]interface{}{
			"type":        "string",
			"description": "Search query.",
		},
		"max_results": map[string]interface{}{
			"type":        "integer",
			"description": "Maximum number of search results to return. Defaults to 5, capped at 10.",
		},
		"region": map[string]interface{}{
			"type":        "string",
			"description": "Optional DuckDuckGo region code, for example us-en, cn-zh, wt-wt.",
		},
		"safe_search": map[string]interface{}{
			"type":        "string",
			"description": "Safe search level: strict, moderate, or off. Defaults to moderate.",
			"enum":        []interface{}{"strict", "moderate", "off"},
		},
		"timeout": map[string]interface{}{
			"type":        "number",
			"description": "Timeout in seconds. Defaults to 15, capped at 60.",
		},
	},
	Required: []string{"query"},
}

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet,omitempty"`
}

// WebSearchTool searches DuckDuckGo through its no-JavaScript HTML endpoint.
type WebSearchTool struct {
	policy    sandbox.NetworkPolicy
	client    *http.Client
	endpoint  string
	timeout   time.Duration
	maxResult int
}

// NewWebSearchTool builds a WebSearchTool using the default DuckDuckGo network allowlist.
func NewWebSearchTool() *WebSearchTool {
	return NewWebSearchToolWithNetworkPolicy(sandbox.NewDomainAllowList("duckduckgo.com", "html.duckduckgo.com"))
}

// NewWebSearchToolWithNetworkPolicy builds a WebSearchTool using a custom network policy.
// A nil policy disables network host checks and should only be used when sandboxing is disabled.
func NewWebSearchToolWithNetworkPolicy(policy sandbox.NetworkPolicy) *WebSearchTool {
	return NewWebSearchToolWithEndpoint(policy, "https://html.duckduckgo.com/html/")
}

// NewWebSearchToolWithEndpoint builds a WebSearchTool against a custom endpoint, primarily for tests.
func NewWebSearchToolWithEndpoint(policy sandbox.NetworkPolicy, endpoint string) *WebSearchTool {
	return &WebSearchTool{
		policy:    policy,
		client:    &http.Client{Timeout: webSearchDefaultTimeout},
		endpoint:  strings.TrimSpace(endpoint),
		timeout:   webSearchDefaultTimeout,
		maxResult: webSearchDefaultLimit,
	}
}

func (w *WebSearchTool) Name() string { return "web_search" }

func (w *WebSearchTool) Description() string { return webSearchToolDesc }

func (w *WebSearchTool) Schema() *tool.JSONSchema { return webSearchSchema }

func (w *WebSearchTool) Metadata() tool.Metadata {
	return tool.Metadata{IsReadOnly: true, IsConcurrencySafe: true}
}

func (w *WebSearchTool) Execute(ctx context.Context, params map[string]interface{}) (*tool.ToolResult, error) {
	if ctx == nil {
		return nil, errors.New("context is nil")
	}
	if w == nil {
		return nil, errors.New("web_search tool is not initialised")
	}
	query, err := parseWebSearchQuery(params)
	if err != nil {
		return nil, err
	}
	maxResults, err := parseWebSearchLimit(params, w.maxResult)
	if err != nil {
		return nil, err
	}
	timeout, err := parseWebSearchTimeout(params, w.timeout)
	if err != nil {
		return nil, err
	}
	searchURL, err := w.buildSearchURL(params, query)
	if err != nil {
		return nil, err
	}
	if err := w.validateURL(searchURL); err != nil {
		return nil, err
	}

	callCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(callCtx, http.MethodGet, searchURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create search request: %w", err)
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("User-Agent", "icoo-claw-agent/1.0")

	client := http.Client{Timeout: timeout}
	if w.client != nil {
		client.Transport = w.client.Transport
		client.Jar = w.client.Jar
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("stopped after 5 redirects")
			}
			return w.validateURL(req.URL)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("duckduckgo search: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, webSearchMaxBody+1))
	if err != nil {
		return nil, fmt.Errorf("read search response: %w", err)
	}
	truncatedBody := len(body) > webSearchMaxBody
	if truncatedBody {
		body = body[:webSearchMaxBody]
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return &tool.ToolResult{
			Success: false,
			Output:  fmt.Sprintf("DuckDuckGo returned HTTP %s", resp.Status),
			Data: map[string]interface{}{
				"query":       query,
				"status":      resp.Status,
				"status_code": resp.StatusCode,
			},
		}, nil
	}

	results, err := parseDuckDuckGoResults(body, maxResults)
	if err != nil {
		return nil, err
	}
	return &tool.ToolResult{
		Success: true,
		Output:  formatSearchResults(results, truncatedBody),
		Data: map[string]interface{}{
			"query":          query,
			"engine":         "duckduckgo",
			"results":        results,
			"count":          len(results),
			"max_results":    maxResults,
			"body_truncated": truncatedBody,
		},
	}, nil
}

func (w *WebSearchTool) buildSearchURL(params map[string]interface{}, query string) (*url.URL, error) {
	endpoint := strings.TrimSpace(w.endpoint)
	if endpoint == "" {
		endpoint = "https://html.duckduckgo.com/html/"
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse search endpoint: %w", err)
	}
	values := u.Query()
	values.Set("q", query)
	if region := parseWebSearchString(params, "region"); region != "" {
		values.Set("kl", region)
	}
	switch parseWebSearchSafeSearch(params) {
	case "strict":
		values.Set("kp", "1")
	case "off":
		values.Set("kp", "-2")
	default:
		values.Set("kp", "-1")
	}
	u.RawQuery = values.Encode()
	return u, nil
}

func (w *WebSearchTool) validateURL(u *url.URL) error {
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
	if w.policy != nil {
		if err := w.policy.Validate(u.Host); err != nil {
			return fmt.Errorf("network host denied: %w", err)
		}
	}
	return nil
}

func parseWebSearchQuery(params map[string]interface{}) (string, error) {
	if params == nil {
		return "", errors.New("params is nil")
	}
	raw, ok := params["query"]
	if !ok {
		return "", errors.New("query is required")
	}
	value, err := coerceString(raw)
	if err != nil {
		return "", fmt.Errorf("query must be string: %w", err)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("query cannot be empty")
	}
	return value, nil
}

func parseWebSearchLimit(params map[string]interface{}, fallback int) (int, error) {
	if fallback <= 0 {
		fallback = webSearchDefaultLimit
	}
	if params == nil {
		return fallback, nil
	}
	raw, ok := params["max_results"]
	if !ok || raw == nil {
		return fallback, nil
	}
	value, err := intFromParam(raw)
	if err != nil {
		return 0, fmt.Errorf("max_results must be integer: %w", err)
	}
	if value <= 0 {
		return 0, errors.New("max_results must be > 0")
	}
	if value > webSearchMaxLimit {
		return webSearchMaxLimit, nil
	}
	return value, nil
}

func parseWebSearchTimeout(params map[string]interface{}, fallback time.Duration) (time.Duration, error) {
	if fallback <= 0 {
		fallback = webSearchDefaultTimeout
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
	if timeout > webSearchMaxTimeout {
		return webSearchMaxTimeout, nil
	}
	return timeout, nil
}

func parseWebSearchString(params map[string]interface{}, key string) string {
	if params == nil {
		return ""
	}
	raw, ok := params[key]
	if !ok || raw == nil {
		return ""
	}
	value, err := coerceString(raw)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func parseWebSearchSafeSearch(params map[string]interface{}) string {
	value := strings.ToLower(parseWebSearchString(params, "safe_search"))
	switch value {
	case "strict", "off":
		return value
	default:
		return "moderate"
	}
}

func parseDuckDuckGoResults(body []byte, maxResults int) ([]SearchResult, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse duckduckgo html: %w", err)
	}
	var results []SearchResult
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n == nil || len(results) >= maxResults {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" && hasHTMLClass(n, "result__a") {
			title := strings.TrimSpace(nodeText(n))
			href := attrValue(n, "href")
			link := normalizeDuckDuckGoURL(href)
			if title != "" && link != "" {
				results = append(results, SearchResult{Title: title, URL: link})
			}
		}
		for child := n.FirstChild; child != nil && len(results) < maxResults; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	assignDuckDuckGoSnippets(doc, results)
	return results, nil
}

func assignDuckDuckGoSnippets(doc *html.Node, results []SearchResult) {
	if doc == nil || len(results) == 0 {
		return
	}
	idx := 0
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n == nil || idx >= len(results) {
			return
		}
		if n.Type == html.ElementNode && hasHTMLClass(n, "result__snippet") {
			results[idx].Snippet = strings.TrimSpace(nodeText(n))
			idx++
			return
		}
		for child := n.FirstChild; child != nil && idx < len(results); child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
}

func normalizeDuckDuckGoURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if encoded := u.Query().Get("uddg"); encoded != "" {
		if decoded, err := url.QueryUnescape(encoded); err == nil && strings.TrimSpace(decoded) != "" {
			return decoded
		}
		return encoded
	}
	return raw
}

func hasHTMLClass(n *html.Node, class string) bool {
	classes := strings.Fields(attrValue(n, "class"))
	for _, value := range classes {
		if value == class {
			return true
		}
	}
	return false
}

func attrValue(n *html.Node, key string) string {
	if n == nil {
		return ""
	}
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func nodeText(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}
	var parts []string
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if text := strings.TrimSpace(nodeText(child)); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " ")
}

func formatSearchResults(results []SearchResult, bodyTruncated bool) string {
	if len(results) == 0 {
		if bodyTruncated {
			return "no results parsed; response body was truncated"
		}
		return "no results"
	}
	var b strings.Builder
	for i, result := range results {
		fmt.Fprintf(&b, "%d. %s\n%s", i+1, result.Title, result.URL)
		if result.Snippet != "" {
			fmt.Fprintf(&b, "\n%s", result.Snippet)
		}
		if i < len(results)-1 {
			b.WriteString("\n\n")
		}
	}
	if bodyTruncated {
		b.WriteString("\n\n... response body truncated")
	}
	return b.String()
}
