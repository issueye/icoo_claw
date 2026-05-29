package model

import "strings"

// CacheHint marks whether a conversation item should be considered for
// provider-side prompt caching when an encoder supports it.
type CacheHint string

const (
	CacheHintNone      CacheHint = ""
	CacheHintEphemeral CacheHint = "ephemeral"
)

// Conversation is a provider-neutral representation of model history. It keeps
// the runtime Message shape intact while separating tool results from assistant
// tool calls so each provider encoder can make its own wire-format choices.
type Conversation struct {
	Messages []ConversationMessage
}

// ConversationMessage is one normalized turn in a Conversation.
type ConversationMessage struct {
	Role             string
	Text             string
	ContentBlocks    []ContentBlock
	ToolCalls        []ConversationToolCall
	ToolResults      []ConversationToolResult
	ReasoningContent string
	CacheHint        CacheHint
}

// ConversationToolCall is a structured assistant tool invocation.
type ConversationToolCall struct {
	ID        string
	Name      string
	Arguments map[string]any
}

// ConversationToolResult is a structured result for a prior tool invocation.
type ConversationToolResult struct {
	ID      string
	Name    string
	Content string
}

// BuildConversation converts runtime model messages into provider-neutral IR.
func BuildConversation(messages []Message) Conversation {
	items := make([]ConversationMessage, 0, len(messages))
	for _, msg := range messages {
		item := ConversationMessage{
			Role:             normalizeRole(msg.Role),
			Text:             msg.Content,
			ContentBlocks:    cloneContentBlocksIR(msg.ContentBlocks),
			ReasoningContent: msg.ReasoningContent,
		}

		if item.Role == "tool" {
			item.ToolResults = toolResultsFromMessage(msg)
		} else {
			item.ToolCalls = toolCallsFromMessage(msg)
		}

		items = append(items, item.Normalize())
	}
	return Conversation{Messages: items}.Normalize()
}

// Clone returns a deep copy of the conversation.
func (c Conversation) Clone() Conversation {
	if len(c.Messages) == 0 {
		return Conversation{}
	}
	out := Conversation{Messages: make([]ConversationMessage, len(c.Messages))}
	for i, msg := range c.Messages {
		out.Messages[i] = msg.Clone()
	}
	return out
}

// Normalize returns a deep-copied conversation with canonical roles and no nil
// mutable nested collections.
func (c Conversation) Normalize() Conversation {
	clone := c.Clone()
	for i := range clone.Messages {
		clone.Messages[i] = clone.Messages[i].Normalize()
	}
	return clone
}

// Clone returns a deep copy of one conversation message.
func (m ConversationMessage) Clone() ConversationMessage {
	out := ConversationMessage{
		Role:             m.Role,
		Text:             m.Text,
		ReasoningContent: m.ReasoningContent,
		CacheHint:        m.CacheHint,
		ContentBlocks:    cloneContentBlocksIR(m.ContentBlocks),
		ToolCalls:        cloneConversationToolCalls(m.ToolCalls),
		ToolResults:      cloneConversationToolResults(m.ToolResults),
	}
	return out
}

// Normalize returns a deep-copied message with canonical role and collection
// fields normalized to empty slices when appropriate.
func (m ConversationMessage) Normalize() ConversationMessage {
	out := m.Clone()
	out.Role = normalizeRole(out.Role)
	if out.CacheHint != CacheHintEphemeral {
		out.CacheHint = CacheHintNone
	}
	if out.ContentBlocks == nil {
		out.ContentBlocks = []ContentBlock{}
	}
	if out.ToolCalls == nil {
		out.ToolCalls = []ConversationToolCall{}
	}
	if out.ToolResults == nil {
		out.ToolResults = []ConversationToolResult{}
	}
	return out
}

func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "assistant":
		return "assistant"
	case "system":
		return "system"
	case "tool":
		return "tool"
	case "developer":
		return "developer"
	default:
		return "user"
	}
}

func toolCallsFromMessage(msg Message) []ConversationToolCall {
	if len(msg.ToolCalls) == 0 {
		return []ConversationToolCall{}
	}
	out := make([]ConversationToolCall, 0, len(msg.ToolCalls))
	for _, call := range msg.ToolCalls {
		out = append(out, ConversationToolCall{
			ID:        call.ID,
			Name:      call.Name,
			Arguments: cloneMapIR(call.Arguments),
		})
	}
	return out
}

func toolResultsFromMessage(msg Message) []ConversationToolResult {
	if len(msg.ToolCalls) == 0 {
		return []ConversationToolResult{{
			Content: msg.Content,
		}}
	}

	out := make([]ConversationToolResult, 0, len(msg.ToolCalls))
	for _, call := range msg.ToolCalls {
		content := call.Result
		if strings.TrimSpace(content) == "" {
			content = msg.Content
		}
		out = append(out, ConversationToolResult{
			ID:      call.ID,
			Name:    call.Name,
			Content: content,
		})
	}
	return out
}

func cloneConversationToolCalls(calls []ConversationToolCall) []ConversationToolCall {
	if len(calls) == 0 {
		return []ConversationToolCall{}
	}
	out := make([]ConversationToolCall, len(calls))
	for i, call := range calls {
		out[i] = ConversationToolCall{
			ID:        call.ID,
			Name:      call.Name,
			Arguments: cloneMapIR(call.Arguments),
		}
	}
	return out
}

func cloneConversationToolResults(results []ConversationToolResult) []ConversationToolResult {
	if len(results) == 0 {
		return []ConversationToolResult{}
	}
	out := make([]ConversationToolResult, len(results))
	copy(out, results)
	return out
}

func cloneContentBlocksIR(blocks []ContentBlock) []ContentBlock {
	if len(blocks) == 0 {
		return []ContentBlock{}
	}
	out := make([]ContentBlock, len(blocks))
	copy(out, blocks)
	return out
}

func cloneMapIR(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	out := make(map[string]any, len(input))
	for k, v := range input {
		out[k] = cloneValueIR(v)
	}
	return out
}

func cloneValueIR(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return cloneMapIR(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = cloneValueIR(item)
		}
		return out
	default:
		return v
	}
}
