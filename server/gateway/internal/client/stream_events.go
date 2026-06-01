package client

import "icoo_claw/common/agentproto"

type StreamEventHandlerFunc = agentproto.StreamEventHandlerFunc
type PlanEntry = agentproto.PlanEntry
type SessionConfigOption = agentproto.SessionConfigOption
type SessionConfigSelectOption = agentproto.SessionConfigSelectOption
type SessionConfigSelectGroup = agentproto.SessionConfigSelectGroup

var DispatchStreamEvents = agentproto.DispatchStreamEvents
var DispatchStreamEvent = agentproto.DispatchStreamEvent
