package storer

import "time"

type Record struct {
	Id        string         `json:"id"`
	SessionId string         `json:"session_id"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata"`
	Embedding []float32      `json:"embedding,omitempty"`
	Score     float32        `json:"score,omitempty"`
	Space     string         `json:"space,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at,omitempty"`
}
