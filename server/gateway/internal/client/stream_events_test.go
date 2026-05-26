package client

import "testing"

func TestDispatchStreamEventDoesNotCallUnhandledForHandledEvents(t *testing.T) {
	handled := 0
	unhandled := 0

	if err := DispatchStreamEvent(StreamEvent{Type: "session/update"}, StreamEventHandlerFunc{
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

