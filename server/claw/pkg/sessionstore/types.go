package sessionstore

import "icoo_claw/common/sessionproto"

// These aliases preserve the historical Claw package path. New shared callers
// should prefer common/sessionproto directly.
type Session = sessionproto.Session
type CreateSessionRequest = sessionproto.CreateSessionRequest
type ListSessionsOptions = sessionproto.ListSessionsOptions
type Message = sessionproto.Message
type MessagesRequest = sessionproto.MessagesRequest
type MessagesResponse = sessionproto.MessagesResponse
type SessionsResponse = sessionproto.SessionsResponse
type Run = sessionproto.Run
type RunsRequest = sessionproto.RunsRequest
type RunsResponse = sessionproto.RunsResponse
type RunEvent = sessionproto.RunEvent
type RunEventsRequest = sessionproto.RunEventsRequest
type RunEventsResponse = sessionproto.RunEventsResponse
