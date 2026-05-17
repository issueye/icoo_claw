package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"icoo_claw/server/session_store/internal/controller"
	"icoo_claw/server/session_store/internal/model"
	"icoo_claw/server/session_store/internal/service"
)

type fakeSessionRepo struct{}

func (f fakeSessionRepo) Create(context.Context, model.Session) error { return nil }
func (f fakeSessionRepo) Get(context.Context, string) (*model.Session, error) {
	now := time.Now().UTC()
	return &model.Session{SessionID: "sess_1", Status: "active", CreatedAt: now, UpdatedAt: now}, nil
}
func (f fakeSessionRepo) Update(_ context.Context, session model.Session) error { return nil }
func (f fakeSessionRepo) Delete(context.Context, string) error                  { return nil }
func (f fakeSessionRepo) ListMessages(context.Context, string, model.MessagePage) ([]model.Message, error) {
	return nil, nil
}
func (f fakeSessionRepo) AppendMessages(context.Context, string, []model.Message) error {
	return nil
}
func (f fakeSessionRepo) ReplaceMessages(context.Context, string, []model.Message) error {
	return nil
}

func TestHealthRoute(t *testing.T) {
	engine := New(Controllers{
		Health:  controller.NewHealthController(),
		Session: controller.NewSessionController(service.NewSessionService(fakeSessionRepo{})),
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
