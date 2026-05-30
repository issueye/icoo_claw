package jsonutil

import (
	"encoding/json"
	"strings"
)

func MarshalStringSlice(values []string) string {
	if values == nil {
		values = []string{}
	}
	payload, _ := json.Marshal(values)
	return string(payload)
}

func UnmarshalStringSlice(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []string{}
	}
	return out
}

func CleanStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
