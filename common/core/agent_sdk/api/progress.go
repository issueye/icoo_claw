package api

import (
	"context"
	"encoding/json"
	"sync/atomic"

	"icoo_claw/common/core/agent_sdk/middleware"
	"icoo_claw/common/core/agent_sdk/model"
	"icoo_claw/common/core/agent_sdk/tool"
)

type streamEmitFunc func(context.Context, StreamEvent)

type agentEmitFunc func(context.Context, AgentEvent)

func newProgressMiddleware(events chan<- StreamEvent) *progressMiddleware {
	return newProgressMiddlewareWithProtocol(events, NewAnthropicSSEProtocol())
}

func newProgressMiddlewareWithProtocol(events chan<- StreamEvent, protocol StreamProtocol) *progressMiddleware {
	return &progressMiddleware{
		emitter:  progressEmitter{ch: events},
		protocol: protocol,
	}
}

type progressMiddleware struct {
	emitter           progressEmitter
	protocol          StreamProtocol
	modelTextStreamed atomic.Bool
	blockIndex        int
}

func (p *progressMiddleware) Name() string { return "progress" }

func (p *progressMiddleware) emit(ctx context.Context, evt StreamEvent) {
	p.emitter.emit(ctx, evt)
}

func (p *progressMiddleware) emitAgent(ctx context.Context, event AgentEvent) {
	encoded := p.protocol.Encode(event)
	if se, ok := encoded.(StreamEvent); ok {
		p.emitter.emit(ctx, se)
	}
}

func (p *progressMiddleware) streamEmit() streamEmitFunc {
	return func(ctx context.Context, evt StreamEvent) {
		if evt.Type == EventContentBlockDelta && evt.Delta != nil && evt.Delta.Type == "text_delta" && evt.Delta.Text != "" {
			p.modelTextStreamed.Store(true)
		}
		p.emit(ctx, evt)
	}
}

func (p *progressMiddleware) streamEmitAgent() agentEmitFunc {
	return func(ctx context.Context, event AgentEvent) {
		if event.Type == AETextDelta && event.Text != "" {
			p.modelTextStreamed.Store(true)
		}
		p.emitAgent(ctx, event)
	}
}

func (p *progressMiddleware) BeforeAgent(ctx context.Context, st *middleware.State) error {
	iter := 0
	if st != nil {
		iter = st.Iteration
	}
	if iter == 0 {
		p.emitAgent(ctx, AgentEvent{Type: AEAgentStart})
	}
	p.emitAgent(ctx, AgentEvent{Type: AEIterationStart, Iteration: iter})
	p.emitAgent(ctx, AgentEvent{Type: AEMessageEnvelopeStart})
	p.blockIndex = 0
	return nil
}

func (p *progressMiddleware) AfterAgent(ctx context.Context, st *middleware.State) error {
	resp, ok := st.ModelOutput.(*model.Response)
	if !ok || resp == nil {
		return nil
	}

	if !p.modelTextStreamed.Load() {
		p.emitTextBlock(ctx, resp.Message.Content)
	}
	if resp.Message.Content != "" {
		p.blockIndex++
	}

	for _, call := range resp.Message.ToolCalls {
		p.emitToolBlock(ctx, call)
		p.blockIndex++
	}

	p.emitAgent(ctx, AgentEvent{
		Type:                AEUsageUpdate,
		InputTokens:         resp.Usage.InputTokens,
		OutputTokens:        resp.Usage.OutputTokens,
		TotalTokens:         resp.Usage.TotalTokens,
		CacheReadTokens:     resp.Usage.CacheReadTokens,
		CacheCreationTokens: resp.Usage.CacheCreationTokens,
	})

	reason := "end_turn"
	if len(resp.Message.ToolCalls) > 0 {
		reason = "tool_use"
	}
	p.emitAgent(ctx, AgentEvent{Type: AEStopReason, StopReason: reason})
	p.emitAgent(ctx, AgentEvent{Type: AEMessageEnvelopeStop})
	p.emitAgent(ctx, AgentEvent{Type: AEIterationStop, Iteration: st.Iteration})
	if len(resp.Message.ToolCalls) == 0 {
		p.emitAgent(ctx, AgentEvent{Type: AEAgentStop})
	}
	return nil
}

func (p *progressMiddleware) BeforeTool(ctx context.Context, st *middleware.State) error {
	call, ok := st.ToolCall.(model.ToolCall)
	if !ok {
		return nil
	}
	p.emitAgent(ctx, AgentEvent{
		Type:      AEToolExecutionStart,
		ToolUseID: call.ID,
		ToolName:  call.Name,
		ToolInput: call.Arguments,
		Iteration: st.Iteration,
	})
	return nil
}

func (p *progressMiddleware) AfterTool(ctx context.Context, st *middleware.State) error {
	call, ok := st.ToolCall.(model.ToolCall)
	if !ok {
		return nil
	}
	cr, ok := st.ToolResult.(*tool.CallResult)
	if !ok || cr == nil {
		return nil
	}

	output := ""
	if cr.Result != nil {
		output = cr.Result.Output
	}
	if output != "" {
		p.emitAgent(ctx, AgentEvent{
			Type:       AEToolExecutionOutput,
			ToolUseID:  call.ID,
			ToolName:   call.Name,
			ToolOutput: output,
		})
	}

	payload := map[string]any{}
	if output != "" {
		payload["output"] = output
	}
	meta := map[string]any{}
	if cr.Err != nil {
		meta["error"] = cr.Err.Error()
	}
	if cr.Result != nil {
		if cr.Result.Data != nil {
			meta["data"] = cr.Result.Data
		}
		if cr.Result.OutputRef != nil {
			meta["output_ref"] = cr.Result.OutputRef
		}
	}
	if len(meta) > 0 {
		payload["metadata"] = meta
	}
	isError := cr.Err != nil
	p.emitAgent(ctx, AgentEvent{
		Type:        AEToolExecutionResult,
		ToolUseID:   call.ID,
		ToolName:    call.Name,
		ToolOutput:  payload,
		IsError:     isError,
	})
	return nil
}

func (p *progressMiddleware) emitTextBlock(ctx context.Context, content string) {
	if content == "" {
		return
	}
	p.emitAgent(ctx, AgentEvent{Type: AETextStart})
	p.emitAgent(ctx, AgentEvent{Type: AETextDelta, Text: content})
	p.emitAgent(ctx, AgentEvent{Type: AETextStop})
}

func (p *progressMiddleware) emitToolBlock(ctx context.Context, call model.ToolCall) {
	p.emitAgent(ctx, AgentEvent{
		Type:      AEToolCallStart,
		ToolUseID: call.ID,
		ToolName:  call.Name,
	})
	raw, err := json.Marshal(call.Arguments)
	if err != nil {
		raw = []byte("{}")
	}
	for _, chunk := range chunkString(string(raw), 10) {
		p.emitAgent(ctx, AgentEvent{
			Type:          AEToolCallInputDelta,
			ToolUseID:     call.ID,
			ToolOutputRaw: chunk,
		})
	}
	p.emitAgent(ctx, AgentEvent{
		Type:      AEToolCallInputStop,
		ToolUseID: call.ID,
	})
}

type progressEmitter struct {
	ch chan<- StreamEvent
}

func (e progressEmitter) emit(ctx context.Context, evt StreamEvent) {
	if e.ch == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return
	case e.ch <- evt:
	}
}

func chunkString(s string, size int) []string {
	if size <= 0 || s == "" {
		return nil
	}
	out := make([]string, 0, (len(s)+size-1)/size)
	for start := 0; start < len(s); start += size {
		end := start + size
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[start:end])
	}
	return out
}
