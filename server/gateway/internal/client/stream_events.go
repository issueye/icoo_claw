package client

type StreamEventHandlerFunc struct {
	OnUpdate    func(StreamEvent) error
	OnCompleted func(StreamEvent) error
	OnError     func(StreamEvent) error
	OnUnhandled func(StreamEvent) error
}

func DispatchStreamEvents(events <-chan StreamEvent, handler StreamEventHandlerFunc) error {
	for event := range events {
		if err := DispatchStreamEvent(event, handler); err != nil {
			return err
		}
	}
	return nil
}

func DispatchStreamEvent(event StreamEvent, handler StreamEventHandlerFunc) error {
	switch event.Type {
	case "session/update":
		if handler.OnUpdate != nil {
			return handler.OnUpdate(event)
		}
	case "session/completed":
		if handler.OnCompleted != nil {
			return handler.OnCompleted(event)
		}
	case "session/error":
		if handler.OnError != nil {
			return handler.OnError(event)
		}
	}
	if handler.OnUnhandled != nil {
		return handler.OnUnhandled(event)
	}
	return nil
}
