package getsafe

import "time"

func String(payload map[string]any, key string) string {
	if v, ok := payload[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func Time(payload map[string]any, key string) time.Time {
	if v, ok := payload[key]; ok {
		if t, ok := v.(time.Time); ok {
			return t
		}
		if s, ok := v.(string); ok {
			if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

func Metadata(payload map[string]any, key string) map[string]any {
	if v, ok := payload[key]; ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return nil
}
