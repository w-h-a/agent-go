package storer

import (
	"encoding/json"
	"strings"
)

func SanitizeEdges(metadata map[string]any) []map[string]string {
	var edges []map[string]string
	if raw, ok := metadata["edges"]; ok {
		valid := ValidateEdges(raw)
		if len(valid) > 0 {
			metadata["edges"] = valid
			edges = valid
		} else {
			delete(metadata, "edges")
		}
	}
	return edges
}

func ValidateEdges(raw any) []map[string]string {
	candidates := ExtractEdges(raw)
	if len(candidates) == 0 {
		return nil
	}

	valid := []map[string]string{}
	seen := map[string]struct{}{}

	for _, edge := range candidates {
		target := SanitizeTarget(edge["target"])
		t := SanitizeType(edge["type"])
		if len(target) == 0 || len(t) == 0 {
			continue
		}

		key := target + "|" + t
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}

		valid = append(valid, map[string]string{
			"target": target,
			"type":   t,
		})
	}

	return valid
}

func ExtractEdges(raw any) []map[string]string {
	if raw == nil {
		return nil
	}

	bytes, err := json.Marshal(raw)
	if err != nil {
		return nil
	}

	var edges []map[string]string
	if err := json.Unmarshal(bytes, &edges); err != nil {
		return nil
	}

	return edges
}

func SanitizeTarget(t string) string {
	return strings.TrimSpace(t)
}

func SanitizeType(t string) string {
	t = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(t), " ", "_"))
	if len(t) == 0 {
		return "RELATED"
	}
	return t
}
