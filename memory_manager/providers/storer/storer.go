package storer

import "context"

type Storer interface {
	Store(ctx context.Context, spaceId string, sessionId string, content string, metadata map[string]any, vector []float32) error
	Search(ctx context.Context, spaceId string, vector []float32, limit int) ([]Record, error)
}
