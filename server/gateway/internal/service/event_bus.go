package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"icoo_claw/common/id"
	"icoo_claw/server/gateway/internal/dto"
)

type EventPublisher interface {
	Publish(ctx context.Context, event dto.EventBusEvent) error
}

type EventBusFilter struct {
	Protocol string
}

type EventBus struct {
	mu          sync.RWMutex
	nextID      atomic.Uint64
	maxHistory  int
	history     []dto.EventBusEvent
	subscribers map[uint64]eventSubscriber
}

type eventSubscriber struct {
	filter EventBusFilter
	ch     chan dto.EventBusEvent
}

func NewEventBus(maxHistory int) *EventBus {
	if maxHistory <= 0 {
		maxHistory = 500
	}
	return &EventBus{
		maxHistory:  maxHistory,
		subscribers: make(map[uint64]eventSubscriber),
	}
}

func (b *EventBus) Publish(_ context.Context, event dto.EventBusEvent) error {
	if b == nil {
		return nil
	}
	event = normalizeEventBusEvent(event)

	b.mu.Lock()
	b.history = append(b.history, event)
	if len(b.history) > b.maxHistory {
		b.history = append([]dto.EventBusEvent(nil), b.history[len(b.history)-b.maxHistory:]...)
	}
	subscribers := make([]eventSubscriber, 0, len(b.subscribers))
	for _, subscriber := range b.subscribers {
		if eventMatchesFilter(event, subscriber.filter) {
			subscribers = append(subscribers, subscriber)
		}
	}
	b.mu.Unlock()

	for _, subscriber := range subscribers {
		select {
		case subscriber.ch <- event:
		default:
		}
	}
	return nil
}

func (b *EventBus) Subscribe(ctx context.Context, filter EventBusFilter) (<-chan dto.EventBusEvent, func()) {
	ch := make(chan dto.EventBusEvent, max(256, b.maxHistory))
	if b == nil {
		close(ch)
		return ch, func() {}
	}

	subscriptionID := b.nextID.Add(1)

	b.mu.Lock()
	for _, event := range b.history {
		if eventMatchesFilter(event, filter) {
			ch <- event
		}
	}
	b.subscribers[subscriptionID] = eventSubscriber{filter: filter, ch: ch}
	b.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subscribers, subscriptionID)
			b.mu.Unlock()
			close(ch)
		})
	}

	if ctx != nil {
		go func() {
			<-ctx.Done()
			cancel()
		}()
	}

	return ch, cancel
}

func (b *EventBus) History(filter EventBusFilter) []dto.EventBusEvent {
	if b == nil {
		return nil
	}
	b.mu.RLock()
	defer b.mu.RUnlock()

	events := make([]dto.EventBusEvent, 0, len(b.history))
	for _, event := range b.history {
		if eventMatchesFilter(event, filter) {
			events = append(events, event)
		}
	}
	return events
}

func normalizeEventBusEvent(event dto.EventBusEvent) dto.EventBusEvent {
	if event.ID == "" {
		event.ID = "evt_" + id.Random()
	}
	if event.Time.IsZero() {
		event.Time = time.Now().UTC()
	}
	if event.Source == "" {
		event.Source = "gateway"
	}
	if event.Protocol == "" {
		event.Protocol = "event"
	}
	if event.Type == "" {
		event.Type = "unknown"
	}
	return event
}

func eventMatchesFilter(event dto.EventBusEvent, filter EventBusFilter) bool {
	return filter.Protocol == "" || event.Protocol == filter.Protocol
}
