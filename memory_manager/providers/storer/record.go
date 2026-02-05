package storer

import "time"

type Record struct {
	Id        string
	SessionId string
	Content   string
	Metadata  map[string]any
	Embedding []float32
	Score     float32
	CreatedAt time.Time
	UpdatedAt time.Time
}
