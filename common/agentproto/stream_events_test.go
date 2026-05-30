package agentproto

import "testing"

func TestDispatchStreamEventDoesNotCallUnhandledForHandledEvents(t *testing.T) {
	handled := 0
	unhandled := 0

	if err := DispatchStreamEvent(StreamEvent{Type: StreamEventSessionUpdate}, StreamEventHandlerFunc{
		OnUpdate: func(StreamEvent) error {
			handled++
			return nil
		},
		OnUnhandled: func(StreamEvent) error {
			unhandled++
			return nil
		},
	}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	if handled != 1 {
		t.Fatalf("handled = %d, want 1", handled)
	}
	if unhandled != 0 {
		t.Fatalf("unhandled = %d, want 0", unhandled)
	}
}

func TestCollectTextStream(t *testing.T) {
	events := make(chan StreamEvent, 3)
	events <- StreamEvent{Type: StreamEventSessionUpdate, SessionID: "sess_1", RequestID: "req_1", Update: &SessionUpdate{SessionUpdate: "agent_message_chunk", Content: &ContentBlock{Type: "text", Text: "he"}}}
	events <- StreamEvent{Type: StreamEventSessionUpdate, SessionID: "sess_1", RequestID: "req_1", Update: &SessionUpdate{SessionUpdate: "agent_message_chunk", Content: &ContentBlock{Type: "text", Text: "llo"}}}
	events <- StreamEvent{Type: StreamEventSessionCompleted, SessionID: "sess_1", RequestID: "req_1", StopReason: "end_turn"}
	close(events)

	got, err := CollectTextStream(events, "fallback_session", "fallback_request")
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if got.Output != "hello" || got.SessionID != "sess_1" || got.RequestID != "req_1" || got.StopReason != "end_turn" {
		t.Fatalf("collected = %+v", got)
	}
}

func TestCollectTextStreamErrorsWhenClosedBeforeCompletion(t *testing.T) {
	events := make(chan StreamEvent, 1)
	events <- StreamEvent{Type: StreamEventSessionUpdate, SessionID: "sess_1", RequestID: "req_1", Update: &SessionUpdate{SessionUpdate: "agent_message_chunk", Content: &ContentBlock{Type: "text", Text: "partial"}}}
	close(events)

	got, err := CollectTextStream(events, "fallback_session", "fallback_request")
	if err == nil {
		t.Fatal("expected stream close error")
	}
	if got.SessionID != "sess_1" || got.RequestID != "req_1" || got.Output != "partial" {
		t.Fatalf("collected = %+v", got)
	}
	if err.Error() != "agent stream closed before completion" {
		t.Fatalf("error = %q", err.Error())
	}
}
