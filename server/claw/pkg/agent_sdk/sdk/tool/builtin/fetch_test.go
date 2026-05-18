package toolbuiltin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"icoo_claw/server/claw/pkg/agent_sdk/sdk/sandbox"
)

func TestFetchToolGetsHTTPResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != "test-agent" {
			t.Fatalf("user-agent = %q, want test-agent", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	fetch := NewFetchToolWithNetworkPolicy(sandbox.NewDomainAllowList("127.0.0.1"))
	result, err := fetch.Execute(context.Background(), map[string]interface{}{
		"url":        server.URL,
		"user_agent": "test-agent",
	})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if !result.Success || !strings.Contains(result.Output, `{"ok":true}`) {
		t.Fatalf("result = %+v", result)
	}
}

func TestFetchToolRejectsDeniedHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	fetch := NewFetchToolWithNetworkPolicy(sandbox.NewDomainAllowList("example.com"))
	_, err := fetch.Execute(context.Background(), map[string]interface{}{"url": server.URL})
	if err == nil {
		t.Fatal("fetch denied host succeeded, want error")
	}
}

func TestFetchToolTruncatesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("abcdef"))
	}))
	defer server.Close()

	fetch := NewFetchToolWithNetworkPolicy(sandbox.NewDomainAllowList("127.0.0.1"))
	result, err := fetch.Execute(context.Background(), map[string]interface{}{
		"url":       server.URL,
		"max_bytes": 3,
	})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if !strings.Contains(result.Output, "abc") || !strings.Contains(result.Output, "truncated") {
		t.Fatalf("output = %q", result.Output)
	}
}
