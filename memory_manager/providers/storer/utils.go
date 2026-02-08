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
		if len(strings.TrimSpace(edge["target"])) == 0 || len(strings.TrimSpace(edge["type"])) == 0 {
			continue
		}

		key := edge["target"] + "|" + edge["type"]
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}

		valid = append(valid, edge)
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
