package qdrant

import (
	"encoding/json"
	"strings"
)

type qdrantEnvelope[T any] struct {
	Status qdrantStatus `json:"status"`
	Result T            `json:"result"`
}

type qdrantStatus struct {
	State string `json:"status"`
	Error string `json:"error,omitempty"`
}

func (s *qdrantStatus) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		s.State = strings.ToLower(v)
		return nil
	}

	var obj struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return err
	}
	if obj.Error != "" {
		s.State = "error"
		s.Error = obj.Error
	}
	return nil
}

type qdrantPointResult struct {
	Id      string         `json:"id"`
	Score   float64        `json:"score"`
	Payload map[string]any `json:"payload"`
	Vector  []float32      `json:"vector"`
}
