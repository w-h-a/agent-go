package storer

import "context"

type Storer interface {
	Upsert(ctx context.Context, sessionId string, content string, metadata map[string]any, vector []float32) error
	Search(ctx context.Context, vector []float32, limit int) ([]Record, error)
}
