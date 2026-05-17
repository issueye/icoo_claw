package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"icoo_claw/server/claw/internal/controller"
	"icoo_claw/server/claw/internal/service"
	"icoo_claw/server/claw/pkg/agent_sdk"
)

func TestHealthRoute(t *testing.T) {
	engine := New(Controllers{
		Health: controller.NewHealthController(),
		Agent:  controller.NewAgentController(service.NewAgentService(agent_sdk.NewFakeRunner())),
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRunRouteUsesFakeRunner(t *testing.T) {
	engine := New(Controllers{
		Health: controller.NewHealthController(),
		Agent:  controller.NewAgentController(service.NewAgentService(agent_sdk.NewFakeRunner())),
	})

	body := `{"session_id":"sess_1","prompt":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/internal/agent/run", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "fake agent response") {
		t.Fatalf("response body = %s", rec.Body.String())
	}
}
