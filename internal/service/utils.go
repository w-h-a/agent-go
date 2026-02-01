package service

import (
	"encoding/json"
	"strings"
)

func splitCommand(payload string) (name string, args string) {
	parts := strings.Fields(payload)
	if len(parts) == 0 {
		return "", ""
	}

	name = parts[0]
	if len(payload) > len(name) {
		args = strings.TrimSpace(payload[len(name):])
	}

	return name, args
}

func parseToolArguments(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}
	var payload map[string]any
	if strings.HasPrefix(raw, "{") {
		if err := json.Unmarshal([]byte(raw), &payload); err == nil {
			return payload
		}
	}
	if strings.HasPrefix(raw, "[") {
		var arr []any
		if err := json.Unmarshal([]byte(raw), &arr); err == nil {
			return map[string]any{"items": arr}
		}
	}
	return map[string]any{"input": raw}
}
