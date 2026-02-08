package memory

import (
	"context"
	"maps"
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

	storer.SanitizeEdges(metadata)

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
		SpaceId:   spaceId,
		CreatedAt: now,
	}

	s.records[id] = rec

	return nil
}

func (s *memoryStorer) Search(ctx context.Context, spaceId string, vector []float32, limit int) ([]storer.Record, error) {
	if limit < 1 {
		return nil, nil
	}

	s.mtx.RLock()
	defer s.mtx.RUnlock()

	candidates := make([]storer.Record, 0, len(s.records))

	for _, rec := range s.records {
		if rec.SpaceId != spaceId {
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

func (s *memoryStorer) SearchNeighborhood(ctx context.Context, seedIds []string, hops int, limit int) ([]storer.Record, error) {
	if limit < 1 || len(seedIds) == 0 {
		return nil, nil
	}

	s.mtx.RLock()
	defer s.mtx.RUnlock()

	visited := map[string]struct{}{}
	var records []storer.Record

	for range hops {
		if len(seedIds) == 0 {
			break
		}

		fetchIds := make([]string, 0, len(seedIds))
		for _, id := range seedIds {
			if _, seen := visited[id]; !seen {
				fetchIds = append(fetchIds, id)
				visited[id] = struct{}{}
			}
		}

		if len(fetchIds) == 0 {
			break
		}

		batch := make([]storer.Record, 0, len(fetchIds))
		for _, id := range fetchIds {
			if rec, exists := s.records[id]; exists {
				batch = append(batch, rec)
			}
		}

		next := []string{}
		for _, rec := range batch {
			records = append(records, rec)
			if len(records) >= limit {
				return records, nil
			}
			if rec.Metadata != nil {
				metadataCopy := make(map[string]any, len(rec.Metadata))
				maps.Copy(metadataCopy, rec.Metadata)
				edges := storer.SanitizeEdges(metadataCopy)
				ids := make([]string, 0, len(edges))
				for _, edge := range edges {
					ids = append(ids, edge["target"])
				}
				next = append(next, ids...)
			}
		}

		seedIds = next
	}

	return records, nil
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
