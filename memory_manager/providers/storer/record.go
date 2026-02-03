package storer

import "time"

type Record struct {
	Id         string
	SessionId  string
	Content    string
	Importance float64
	Metadata   map[string]any
	Embedding  []float32
	Score      float32
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
