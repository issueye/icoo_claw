package toolbuiltin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"icoo_claw/common/core/agent_sdk/sdk/runtime/skills"
	"icoo_claw/common/core/agent_sdk/sdk/runtime/subagents"
	"icoo_claw/common/core/agent_sdk/sdk/tool"
)

const skillToolDescriptionHeader = `Execute a skill.

<skills_instructions>
Call this tool with {"command":"<skill-name>"} (no arguments).
Only use skills listed in <available_skills>. Do not invoke a skill that is already running.
</skills_instructions>

<available_skills>
`

var skillSchema = &tool.JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"command": map[string]interface{}{
			"type":        "string",
			"description": "The skill name (no arguments). E.g., \"pdf\" or \"xlsx\"",
		},
	},
	Required: []string{"command"},
}

// ActivationContextProvider resolves the activation context for manual skill calls.
type ActivationContextProvider func(context.Context) skills.ActivationContext

// SkillSubagentDispatcher runs a prepared skill instruction in a subagent.
type SkillSubagentDispatcher interface {
	Dispatch(context.Context, subagents.Request) (subagents.Result, error)
}

// SkillTool adapts the runtime skills registry into a tool.
type SkillTool struct {
	registry *skills.Registry
	provider ActivationContextProvider
	dispatch SkillSubagentDispatcher
}

// NewSkillTool wires the registry with an optional activation provider.
func NewSkillTool(reg *skills.Registry, provider ActivationContextProvider) *SkillTool {
	return NewSkillToolWithSubagent(reg, provider, nil)
}

// NewSkillToolWithSubagent wires the registry to subagent-backed execution.
func NewSkillToolWithSubagent(reg *skills.Registry, provider ActivationContextProvider, dispatcher SkillSubagentDispatcher) *SkillTool {
	if provider == nil {
		provider = defaultActivationProvider
	}
	return &SkillTool{registry: reg, provider: provider, dispatch: dispatcher}
}

func (s *SkillTool) Name() string { return "skill" }

func (s *SkillTool) Description() string {
	var defs []skills.Definition
	if s != nil && s.registry != nil {
		defs = s.registry.List()
	}
	return buildSkillDescription(defs)
}

func (s *SkillTool) Schema() *tool.JSONSchema { return skillSchema }

func buildSkillDescription(defs []skills.Definition) string {
	var b strings.Builder
	b.WriteString(skillToolDescriptionHeader)
	if len(defs) == 0 {
		b.WriteString("</available_skills>\n")
		return b.String()
	}
	for i, def := range defs {
		writeSkillDefinition(&b, def)
		if i < len(defs)-1 {
			b.WriteByte('\n')
		}
	}
	b.WriteString("</available_skills>\n")
	return b.String()
}

func writeSkillDefinition(b *strings.Builder, def skills.Definition) {
	name := strings.TrimSpace(def.Name)
	if name == "" {
		name = "unknown"
	}
	description := strings.TrimSpace(def.Description)
	if description == "" {
		description = "No description provided."
	}
	location := strings.TrimSpace(skillLocation(def))
	if location == "" {
		location = "unspecified"
	}

	fmt.Fprintf(b, `<skill>
<name>
%s
</name>
<description>
%s
</description>
<location>
%s
</location>
</skill>
`, escapeXML(name), escapeXML(description), escapeXML(location))
}

func (s *SkillTool) Execute(ctx context.Context, params map[string]interface{}) (*tool.ToolResult, error) {
	if ctx == nil {
		return nil, errors.New("context is nil")
	}
	if s == nil || s.registry == nil {
		return nil, errors.New("skill registry is not initialised")
	}
	name, err := parseSkillName(params)
	if err != nil {
		return nil, err
	}
	act := s.provider(ctx)
	result, err := s.registry.Execute(ctx, name, act)
	if err != nil {
		return nil, err
	}
	if s.dispatch == nil {
		return nil, errors.New("skill execution requires a subagent dispatcher")
	}
	request := SkillSubagentRequest(result, act)
	if strings.TrimSpace(request.Instruction) == "" {
		return nil, errors.New("skill instruction is empty")
	}
	subResult, err := s.dispatch.Dispatch(ctx, request)
	if err != nil {
		return nil, err
	}
	output := FormatSubagentOutput(subResult)
	data := map[string]interface{}{
		"skill":             result.Skill,
		"subagent":          subResult.Subagent,
		"summary":           output,
		"skill_metadata":    result.Metadata,
		"subagent_metadata": subResult.Metadata,
	}
	return &tool.ToolResult{
		Success: true,
		Output:  output,
		Data:    data,
	}, nil
}

func parseSkillName(params map[string]interface{}) (string, error) {
	if params == nil {
		return "", errors.New("params is nil")
	}
	raw, ok := params["command"]
	if !ok {
		return "", errors.New("command is required")
	}
	name, err := coerceString(raw)
	if err != nil {
		return "", fmt.Errorf("command must be string: %w", err)
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return "", errors.New("command cannot be empty")
	}
	return name, nil
}

func formatSkillOutput(result skills.Result) string {
	switch v := result.Output.(type) {
	case string:
		if strings.TrimSpace(v) != "" {
			return v
		}
	case fmt.Stringer:
		if text := strings.TrimSpace(v.String()); text != "" {
			return text
		}
	case nil:
	default:
		if data, err := json.Marshal(v); err == nil {
			text := strings.TrimSpace(string(data))
			if text != "" && text != "null" {
				return text
			}
		}
	}
	if result.Skill == "" {
		return "skill executed"
	}
	return fmt.Sprintf("skill %s executed", result.Skill)
}

// SkillSubagentRequest builds the delegated execution request for a loaded skill.
func SkillSubagentRequest(result skills.Result, act skills.ActivationContext) subagents.Request {
	return subagents.Request{
		Target:        subagents.TypeSkillExecutor,
		Instruction:   BuildSkillSubagentInstruction(result, act),
		Activation:    act,
		ToolWhitelist: SkillToolWhitelist(result),
		Metadata: map[string]any{
			"skill":          result.Skill,
			"skill_metadata": result.Metadata,
			"source":         result.Metadata["source"],
		},
	}
}

// BuildSkillSubagentInstruction converts loaded skill content into an executable subagent task.
func BuildSkillSubagentInstruction(result skills.Result, act skills.ActivationContext) string {
	body, support := skillBodyAndSupport(result.Output)
	source := strings.TrimSpace(fmt.Sprint(result.Metadata["source"]))
	var b strings.Builder
	b.WriteString("Execute the loaded skill for the user's request. Follow the skill instructions exactly, use available tools when needed, and return only a concise answer or summary for the main agent.\n\n")
	if prompt := strings.TrimSpace(act.Prompt); prompt != "" {
		b.WriteString("<user_request>\n")
		b.WriteString(prompt)
		b.WriteString("\n</user_request>\n\n")
	}
	b.WriteString("<skill>\n")
	if name := strings.TrimSpace(result.Skill); name != "" {
		b.WriteString("name: ")
		b.WriteString(name)
		b.WriteByte('\n')
	}
	if source != "" {
		b.WriteString("source: ")
		b.WriteString(source)
		b.WriteByte('\n')
	}
	b.WriteString("\n<instructions>\n")
	b.WriteString(strings.TrimSpace(body))
	b.WriteString("\n</instructions>\n")
	if listing := formatSupportFiles(source, support); listing != "" {
		b.WriteString("\n<support_files>\n")
		b.WriteString(listing)
		b.WriteString("\n</support_files>\n")
	}
	b.WriteString("</skill>\n\n")
	b.WriteString("Return only the final result for the user. Do not repeat the skill instructions.")
	return b.String()
}

func skillBodyAndSupport(output any) (string, map[string][]string) {
	switch v := output.(type) {
	case map[string]any:
		body, _ := v["body"].(string)
		return body, coerceSupportFiles(v["support_files"])
	case string:
		return v, nil
	default:
		return formatSkillOutput(skills.Result{Output: output}), nil
	}
}

func coerceSupportFiles(value any) map[string][]string {
	switch v := value.(type) {
	case map[string][]string:
		out := make(map[string][]string, len(v))
		for k, files := range v {
			out[k] = append([]string(nil), files...)
		}
		return out
	case map[string]any:
		out := map[string][]string{}
		for k, raw := range v {
			if files := stringList(raw); len(files) > 0 {
				out[k] = files
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}

func formatSupportFiles(source string, support map[string][]string) string {
	if len(support) == 0 {
		return ""
	}
	base := ""
	if source != "" {
		base = strings.TrimSuffix(strings.ReplaceAll(source, "\\", "/"), "/SKILL.md")
	}
	keys := make([]string, 0, len(support))
	for key := range support {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		files := append([]string(nil), support[key]...)
		sort.Strings(files)
		for _, file := range files {
			file = strings.TrimSpace(file)
			if file == "" {
				continue
			}
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString("- ")
			if base != "" {
				b.WriteString(base)
				b.WriteByte('/')
			}
			b.WriteString(strings.Trim(strings.ReplaceAll(key, "\\", "/"), "/"))
			b.WriteByte('/')
			b.WriteString(strings.TrimLeft(strings.ReplaceAll(file, "\\", "/"), "/"))
		}
	}
	return b.String()
}

// SkillToolWhitelist returns the tools a skill subagent may use.
func SkillToolWhitelist(result skills.Result) []string {
	tools := stringList(result.Metadata["allowed-tools"])
	if len(tools) == 0 {
		return nil
	}
	out := tools[:0]
	for _, name := range tools {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" || name == "skill" {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func stringList(value any) []string {
	switch v := value.(type) {
	case []string:
		out := append([]string(nil), v...)
		sort.Strings(out)
		return out
	case []any:
		var out []string
		for _, item := range v {
			if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
				out = append(out, text)
			}
		}
		sort.Strings(out)
		return out
	case string:
		var out []string
		for _, part := range strings.Split(v, ",") {
			if text := strings.TrimSpace(part); text != "" {
				out = append(out, text)
			}
		}
		sort.Strings(out)
		return out
	default:
		return nil
	}
}

// FormatSubagentOutput converts subagent output into the tool-facing summary string.
func FormatSubagentOutput(result subagents.Result) string {
	switch v := result.Output.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case nil:
		return ""
	default:
		if data, err := json.Marshal(v); err == nil {
			return strings.TrimSpace(string(data))
		}
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

type activationContextKey struct{}

// WithSkillActivationContext attaches a skills.ActivationContext to the context.
func WithSkillActivationContext(ctx context.Context, ac skills.ActivationContext) context.Context {
	return context.WithValue(ctx, activationContextKey{}, ac.Clone())
}

// SkillActivationContextFromContext extracts an activation context if present.
func SkillActivationContextFromContext(ctx context.Context) (skills.ActivationContext, bool) {
	if ctx == nil {
		return skills.ActivationContext{}, false
	}
	ac, ok := ctx.Value(activationContextKey{}).(skills.ActivationContext)
	if !ok {
		return skills.ActivationContext{}, false
	}
	return ac, true
}

func defaultActivationProvider(ctx context.Context) skills.ActivationContext {
	if ac, ok := SkillActivationContextFromContext(ctx); ok {
		return ac
	}
	return skills.ActivationContext{}
}

func skillLocation(def skills.Definition) string {
	if len(def.Metadata) == 0 {
		return ""
	}
	for _, key := range []string{"location", "source", "origin"} {
		if value := strings.TrimSpace(def.Metadata[key]); value != "" {
			return value
		}
	}
	return ""
}

var skillDescriptionEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
	"'", "&apos;",
)

func escapeXML(value string) string {
	if value == "" {
		return ""
	}
	return skillDescriptionEscaper.Replace(value)
}
