package agent_sdk

import "icoo_claw/common/agentproto"

type StreamEventHandlerFunc = agentproto.StreamEventHandlerFunc

var DispatchStreamEvents = agentproto.DispatchStreamEvents
var DispatchStreamEvent = agentproto.DispatchStreamEvent
