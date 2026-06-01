package service

import (
	"context"
	"testing"
	"time"

	"icoo_claw/server/gateway/internal/dto"
)

func TestEventBusReplaysHistoryAndPublishesLiveEvents(t *testing.T) {
	bus := NewEventBus(4)
	if err := bus.Publish(context.Background(), dto.EventBusEvent{Protocol: "acp", Type: "session/update"}); err != nil {
		t.Fatalf("publish history: %v", err)
	}
	if err := bus.Publish(context.Background(), dto.EventBusEvent{Protocol: "other", Type: "ignored"}); err != nil {
		t.Fatalf("publish other: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events, unsubscribe := bus.Subscribe(ctx, EventBusFilter{Protocol: "acp"})
	defer unsubscribe()

	replayed := mustReceiveEvent(t, events)
	if replayed.Type != "session/update" {
		t.Fatalf("replayed = %+v", replayed)
	}

	if err := bus.Publish(context.Background(), dto.EventBusEvent{Protocol: "acp", Type: "session/completed"}); err != nil {
		t.Fatalf("publish live: %v", err)
	}
	live := mustReceiveEvent(t, events)
	if live.Type != "session/completed" || live.ID == "" || live.Time.IsZero() {
		t.Fatalf("live = %+v", live)
	}
}

func TestEventBusCapsHistory(t *testing.T) {
	bus := NewEventBus(2)
	_ = bus.Publish(context.Background(), dto.EventBusEvent{Protocol: "acp", Type: "one"})
	_ = bus.Publish(context.Background(), dto.EventBusEvent{Protocol: "acp", Type: "two"})
	_ = bus.Publish(context.Background(), dto.EventBusEvent{Protocol: "acp", Type: "three"})

	history := bus.History(EventBusFilter{Protocol: "acp"})
	if len(history) != 2 || history[0].Type != "two" || history[1].Type != "three" {
		t.Fatalf("history = %+v", history)
	}
}

func mustReceiveEvent(t *testing.T, events <-chan dto.EventBusEvent) dto.EventBusEvent {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
		return dto.EventBusEvent{}
	}
}
