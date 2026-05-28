package agent_sdk

type StreamEventHandlerFunc struct {
	OnUpdate     func(StreamEvent) error
	OnCompleted  func(StreamEvent) error
	OnError      func(StreamEvent) error
	OnUnhandled  func(StreamEvent) error
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
	case StreamEventSessionUpdate:
		if handler.OnUpdate != nil {
			return handler.OnUpdate(event)
		}
	case StreamEventSessionCompleted:
		if handler.OnCompleted != nil {
			return handler.OnCompleted(event)
		}
	case StreamEventSessionError:
		if handler.OnError != nil {
			return handler.OnError(event)
		}
	}
	if handler.OnUnhandled != nil {
		return handler.OnUnhandled(event)
	}
	return nil
}
