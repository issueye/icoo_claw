package middleware

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

const (
	defaultRedactionReplacement = "[REDACTED]"
	defaultMaxStringLength      = 4096
	truncatedSuffix             = "...[TRUNCATED]"
)

var defaultSensitiveKeyFragments = []string{
	"api_key",
	"token",
	"secret",
	"password",
	"authorization",
	"header",
}

// ReplayArtifact is the portable payload used to replay or debug a traced run.
type ReplayArtifact struct {
	RunID     string         `json:"run_id,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	Events    []TraceEvent   `json:"events"`
	Settings  map[string]any `json:"settings,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// ReplayArtifactOptions configures BuildReplayArtifact.
type ReplayArtifactOptions struct {
	RunID     string
	SessionID string
	RequestID string
	CreatedAt time.Time
	Events    []TraceEvent
	Settings  map[string]any
	Metadata  map[string]any
	Redactor  *Redactor
}

// Redactor produces replay-safe copies of trace payloads.
type Redactor struct {
	Replacement           string
	MaxStringLength       int
	SensitiveKeyFragments []string
}

// DefaultRedactor returns the default replay redaction policy.
func DefaultRedactor() Redactor {
	return Redactor{
		Replacement:           defaultRedactionReplacement,
		MaxStringLength:       defaultMaxStringLength,
		SensitiveKeyFragments: append([]string(nil), defaultSensitiveKeyFragments...),
	}
}

// BuildReplayArtifact creates a replay-safe artifact without mutating inputs.
func BuildReplayArtifact(opts ReplayArtifactOptions) ReplayArtifact {
	redactor := DefaultRedactor()
	if opts.Redactor != nil {
		redactor = opts.Redactor.normalized()
	}
	createdAt := opts.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	return ReplayArtifact{
		RunID:     opts.RunID,
		SessionID: opts.SessionID,
		RequestID: opts.RequestID,
		CreatedAt: createdAt,
		Events:    redactor.RedactTraceEvents(opts.Events),
		Settings:  redactor.redactMap(opts.Settings),
		Metadata:  redactor.redactMap(opts.Metadata),
	}
}

// Redact returns a replay-safe copy of v.
func (r Redactor) Redact(v any) any {
	normalized := r.normalized()
	return normalized.redactValue(v, "")
}

// RedactTraceEvents returns replay-safe copies of trace events.
func (r Redactor) RedactTraceEvents(events []TraceEvent) []TraceEvent {
	if len(events) == 0 {
		return nil
	}
	normalized := r.normalized()
	out := make([]TraceEvent, len(events))
	for i, evt := range events {
		out[i] = normalized.redactTraceEvent(evt)
	}
	return out
}

func (r Redactor) normalized() Redactor {
	if r.Replacement == "" {
		r.Replacement = defaultRedactionReplacement
	}
	if r.MaxStringLength <= 0 {
		r.MaxStringLength = defaultMaxStringLength
	}
	if len(r.SensitiveKeyFragments) == 0 {
		r.SensitiveKeyFragments = defaultSensitiveKeyFragments
	}
	return r
}

func (r Redactor) redactTraceEvent(evt TraceEvent) TraceEvent {
	evt.Input = r.redactValue(evt.Input, "")
	evt.Output = r.redactValue(evt.Output, "")
	evt.ModelRequest = r.redactMap(evt.ModelRequest)
	evt.ModelResponse = r.redactMap(evt.ModelResponse)
	evt.ToolCall = r.redactMap(evt.ToolCall)
	evt.ToolResult = r.redactMap(evt.ToolResult)
	evt.Error = r.truncateString(evt.Error)
	return evt
}

func (r Redactor) redactMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]any, len(src))
	for key, value := range src {
		out[key] = r.redactValue(value, key)
	}
	return out
}

func (r Redactor) redactValue(v any, key string) any {
	if r.isSensitiveKey(key) {
		return r.Replacement
	}
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case string:
		return r.truncateString(val)
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = r.redactValue(item, "")
		}
		return out
	case []string:
		out := make([]string, len(val))
		for i, item := range val {
			out[i] = r.truncateString(item)
		}
		return out
	case map[string]any:
		return r.redactMap(val)
	case map[string]string:
		out := make(map[string]string, len(val))
		for nestedKey, nestedValue := range val {
			if r.isSensitiveKey(nestedKey) {
				out[nestedKey] = r.Replacement
				continue
			}
			out[nestedKey] = r.truncateString(nestedValue)
		}
		return out
	}
	return r.redactReflect(v)
}

func (r Redactor) redactReflect(v any) any {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil
	}
	switch rv.Kind() {
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return v
		}
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			key := iter.Key().String()
			out[key] = r.redactValue(iter.Value().Interface(), key)
		}
		return out
	case reflect.Slice, reflect.Array:
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = r.redactValue(rv.Index(i).Interface(), "")
		}
		return out
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return nil
		}
		return r.redactValue(rv.Elem().Interface(), "")
	case reflect.String:
		return r.truncateString(fmt.Sprint(v))
	default:
		return v
	}
}

func (r Redactor) isSensitiveKey(key string) bool {
	if key == "" {
		return false
	}
	normalized := strings.ToLower(key)
	for _, fragment := range r.SensitiveKeyFragments {
		if fragment == "" {
			continue
		}
		if strings.Contains(normalized, strings.ToLower(fragment)) {
			return true
		}
	}
	return false
}

func (r Redactor) truncateString(value string) string {
	if len(value) <= r.MaxStringLength {
		return value
	}
	limit := r.MaxStringLength - len(truncatedSuffix)
	if limit < 0 {
		limit = 0
	}
	return value[:limit] + truncatedSuffix
}
