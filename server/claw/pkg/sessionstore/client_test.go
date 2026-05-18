package sessionstore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientListAndReplaceMessages(t *testing.T) {
	var replaced MessagesRequest
	var appendedRuns RunsRequest
	var appendedEvents RunEventsRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/sessions":
			_ = json.NewEncoder(w).Encode(SessionsResponse{Sessions: []Session{{SessionID: "sess_1", Status: "active"}}})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/sessions/sess_1/messages":
			_ = json.NewEncoder(w).Encode(MessagesResponse{Messages: []Message{{ID: "msg_1", Role: "user", Content: "hello"}}, Revision: 7})
		case r.Method == http.MethodPut && r.URL.Path == "/v1/sessions/sess_1/messages/snapshot":
			w.WriteHeader(http.StatusNoContent)
			if err := json.NewDecoder(r.Body).Decode(&replaced); err != nil {
				t.Errorf("decode replace request: %v", err)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/v1/sessions/sess_1/runs":
			_ = json.NewEncoder(w).Encode(RunsResponse{Runs: []Run{{ID: "run_1", Status: "completed"}}})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/sessions/sess_1/runs":
			w.WriteHeader(http.StatusNoContent)
			if err := json.NewDecoder(r.Body).Decode(&appendedRuns); err != nil {
				t.Errorf("decode append runs request: %v", err)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/v1/sessions/sess_1/runs/run_1/events":
			_ = json.NewEncoder(w).Encode(RunEventsResponse{Events: []RunEvent{{ID: "evt_1", RunID: "run_1", Type: "delta"}}})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/sessions/sess_1/runs/run_1/events":
			w.WriteHeader(http.StatusNoContent)
			if err := json.NewDecoder(r.Body).Decode(&appendedEvents); err != nil {
				t.Errorf("decode append events request: %v", err)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, server.Client())
	sessions, err := client.ListSessions(context.Background(), ListSessionsOptions{UserID: "user_1", Limit: 10})
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].SessionID != "sess_1" {
		t.Fatalf("sessions = %+v", sessions)
	}

	messages, err := client.ListMessages(context.Background(), "sess_1")
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 1 || messages[0].Content != "hello" {
		t.Fatalf("messages = %+v", messages)
	}
	_, revision, err := client.ListMessagesWithRevision(context.Background(), "sess_1")
	if err != nil {
		t.Fatalf("list messages with revision: %v", err)
	}
	if revision != 7 {
		t.Fatalf("revision = %d, want 7", revision)
	}

	expectedRevision := int64(7)
	err = client.ReplaceMessagesWithRevision(context.Background(), "sess_1", []Message{{ID: "msg_2", Role: "assistant", Content: "hi"}}, &expectedRevision)
	if err != nil {
		t.Fatalf("replace messages: %v", err)
	}
	if len(replaced.Messages) != 1 || replaced.Messages[0].Content != "hi" {
		t.Fatalf("replace request = %+v", replaced)
	}
	if replaced.ExpectedRevision == nil || *replaced.ExpectedRevision != 7 {
		t.Fatalf("expected revision = %v, want 7", replaced.ExpectedRevision)
	}

	runs, err := client.ListRuns(context.Background(), "sess_1")
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 1 || runs[0].ID != "run_1" {
		t.Fatalf("runs = %+v", runs)
	}
	if err := client.AppendRuns(context.Background(), "sess_1", []Run{{ID: "run_2", Status: "completed"}}); err != nil {
		t.Fatalf("append runs: %v", err)
	}
	if len(appendedRuns.Runs) != 1 || appendedRuns.Runs[0].ID != "run_2" {
		t.Fatalf("append runs request = %+v", appendedRuns)
	}
	events, err := client.ListRunEvents(context.Background(), "sess_1", "run_1")
	if err != nil {
		t.Fatalf("list run events: %v", err)
	}
	if len(events) != 1 || events[0].ID != "evt_1" {
		t.Fatalf("events = %+v", events)
	}
	if err := client.AppendRunEvents(context.Background(), "sess_1", "run_1", []RunEvent{{ID: "evt_2", Type: "done"}}); err != nil {
		t.Fatalf("append run events: %v", err)
	}
	if len(appendedEvents.Events) != 1 || appendedEvents.Events[0].ID != "evt_2" {
		t.Fatalf("append events request = %+v", appendedEvents)
	}
}
