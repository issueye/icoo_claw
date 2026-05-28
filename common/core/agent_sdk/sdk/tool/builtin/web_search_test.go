package toolbuiltin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"icoo_claw/common/core/agent_sdk/sdk/sandbox"
)

func TestWebSearchToolParsesDuckDuckGoHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "agent tools" {
			t.Fatalf("query = %q, want agent tools", r.URL.Query().Get("q"))
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`
			<html><body>
				<div class="result">
					<a class="result__a" href="/l/?uddg=https%3A%2F%2Fexample.com%2Ftools">Agent Tools</a>
					<a class="result__snippet">A useful result about tools.</a>
				</div>
				<div class="result">
					<a class="result__a" href="https://example.org/agents">Agents</a>
					<a class="result__snippet">Another result.</a>
				</div>
			</body></html>`))
	}))
	defer server.Close()

	search := NewWebSearchToolWithEndpoint(sandbox.NewDomainAllowList("127.0.0.1"), server.URL)
	result, err := search.Execute(context.Background(), map[string]interface{}{
		"query":       "agent tools",
		"max_results": 1,
	})
	if err != nil {
		t.Fatalf("web_search: %v", err)
	}
	if !result.Success || !strings.Contains(result.Output, "Agent Tools") || !strings.Contains(result.Output, "https://example.com/tools") {
		t.Fatalf("result = %+v", result)
	}
	data := result.Data.(map[string]interface{})
	if data["count"] != 1 {
		t.Fatalf("count = %v, want 1", data["count"])
	}
}

func TestWebSearchToolRejectsDeniedDuckDuckGoHost(t *testing.T) {
	search := NewWebSearchToolWithNetworkPolicy(sandbox.NewDomainAllowList("example.com"))
	_, err := search.Execute(context.Background(), map[string]interface{}{"query": "agent tools"})
	if err == nil {
		t.Fatal("web_search denied host succeeded, want error")
	}
}
