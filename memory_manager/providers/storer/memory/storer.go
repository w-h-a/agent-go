package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	memorymanager "github.com/w-h-a/agent/memory_manager"
	"github.com/w-h-a/agent/memory_manager/providers/storer"
)

type memoryStorer struct {
	options storer.Options
	records map[string]storer.Record
	mtx     sync.RWMutex
}

func (s *memoryStorer) Store(ctx context.Context, spaceId string, sessionId string, content string, metadata map[string]any, vector []float32) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	id := uuid.New().String()

	now := time.Now().UTC()

	cpy := make([]float32, len(vector))
	copy(cpy, vector)

	rec := storer.Record{
		Id:        id,
		SessionId: sessionId,
		Content:   content,
		Metadata:  metadata,
		Embedding: cpy,
		Space:     spaceId,
		CreatedAt: now,
	}

	s.records[id] = rec

	return nil
}

func (s *memoryStorer) Search(ctx context.Context, spaceId string, vector []float32, limit int) ([]storer.Record, error) {
	if limit <= 0 {
		return nil, nil
	}

	s.mtx.RLock()
	defer s.mtx.RUnlock()

	candidates := make([]storer.Record, 0, len(s.records))

	for _, rec := range s.records {
		if rec.Space != spaceId {
			continue
		}
		score := memorymanager.CosineSimilarity(vector, rec.Embedding)
		rec.Score = float32(score)
		candidates = append(candidates, rec)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	return candidates, nil
}

func NewStorer(opts ...storer.Option) *memoryStorer {
	options := storer.NewOptions(opts...)

	s := &memoryStorer{
		options: options,
		records: map[string]storer.Record{},
		mtx:     sync.RWMutex{},
	}

	return s
}
