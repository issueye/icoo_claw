package toolbuiltin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"icoo_claw/common/core/agent_sdk/sandbox"
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

func TestFetchToolUsesConfiguredHTTPProxy(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("target should not be reached directly")
	}))
	defer target.Close()

	proxyHit := false
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHit = true
		if strings.TrimRight(r.URL.String(), "/") != strings.TrimRight(target.URL, "/") {
			t.Fatalf("proxy request URL = %q, want %q", r.URL.String(), target.URL)
		}
		_, _ = w.Write([]byte("via proxy"))
	}))
	defer proxy.Close()

	fetch := NewFetchToolWithNetworkPolicyAndOptions(
		sandbox.NewDomainAllowList("127.0.0.1"),
		NetworkOptions{HTTPProxy: proxy.URL},
	)
	result, err := fetch.Execute(context.Background(), map[string]interface{}{"url": target.URL})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if !proxyHit || !strings.Contains(result.Output, "via proxy") {
		t.Fatalf("proxyHit=%v output=%q", proxyHit, result.Output)
	}
}

func TestFetchToolNoProxyBypassesConfiguredProxy(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("direct"))
	}))
	defer target.Close()

	proxyHit := false
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHit = true
		_, _ = w.Write([]byte("proxy"))
	}))
	defer proxy.Close()

	fetch := NewFetchToolWithNetworkPolicyAndOptions(
		sandbox.NewDomainAllowList("127.0.0.1"),
		NetworkOptions{HTTPProxy: proxy.URL, NoProxy: "127.0.0.1"},
	)
	result, err := fetch.Execute(context.Background(), map[string]interface{}{"url": target.URL})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if proxyHit || !strings.Contains(result.Output, "direct") {
		t.Fatalf("proxyHit=%v output=%q", proxyHit, result.Output)
	}
}
