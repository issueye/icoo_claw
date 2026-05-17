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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/sessions/sess_1/messages":
			_ = json.NewEncoder(w).Encode(MessagesResponse{Messages: []Message{{ID: "msg_1", Role: "user", Content: "hello"}}})
		case r.Method == http.MethodPut && r.URL.Path == "/v1/sessions/sess_1/messages/snapshot":
			w.WriteHeader(http.StatusNoContent)
			if err := json.NewDecoder(r.Body).Decode(&replaced); err != nil {
				t.Errorf("decode replace request: %v", err)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, server.Client())
	messages, err := client.ListMessages(context.Background(), "sess_1")
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 1 || messages[0].Content != "hello" {
		t.Fatalf("messages = %+v", messages)
	}

	err = client.ReplaceMessages(context.Background(), "sess_1", []Message{{ID: "msg_2", Role: "assistant", Content: "hi"}})
	if err != nil {
		t.Fatalf("replace messages: %v", err)
	}
	if len(replaced.Messages) != 1 || replaced.Messages[0].Content != "hi" {
		t.Fatalf("replace request = %+v", replaced)
	}
}
