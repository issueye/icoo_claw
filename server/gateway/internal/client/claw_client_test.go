package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClawClientStreamHandlesLargeSSELine(t *testing.T) {
	largeOutput := strings.Repeat("x", 128*1024)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/agent/run/stream" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		payload, _ := json.Marshal(StreamEvent{
			Type: "session/update",
			Update: &SessionUpdate{
				SessionUpdate: "agent_message_chunk",
				Content:       &ContentBlock{Type: "text", Text: largeOutput},
			},
		})
		_, _ = w.Write([]byte("data: " + string(payload) + "\n\n"))
	}))
	t.Cleanup(server.Close)

	client := NewClawClient(server.Client())
	events, err := client.Stream(context.Background(), server.URL, RunRequest{SessionID: "sess_1", Prompt: "hello"})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}

	event, ok := <-events
	if !ok {
		t.Fatal("expected stream event")
	}
	if event.Type != "session/update" || event.Update == nil || event.Update.Content == nil || event.Update.Content.Text != largeOutput {
		t.Fatalf("event = %+v", event)
	}
}

func TestClawClientPreservesErrorCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"code":"session_busy","error":"session is already running"}`))
	}))
	t.Cleanup(server.Close)

	client := NewClawClient(server.Client())
	_, err := client.Run(context.Background(), server.URL, RunRequest{SessionID: "sess_1", Prompt: "hello"})
	if err == nil {
		t.Fatal("expected error")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("error = %T, want HTTPError", err)
	}
	if httpErr.StatusCode != http.StatusConflict || httpErr.Code != "session_busy" {
		t.Fatalf("http error = %+v", httpErr)
	}
}
