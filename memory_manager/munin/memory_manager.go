package munin

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	memorymanager "github.com/w-h-a/agent/memory_manager"
)

type muninMemoryManager struct {
	options   memorymanager.Options
	counter   atomic.Uint64
	shortTerm map[string][]memorymanager.Message
	mtx       sync.RWMutex
}

func (m *muninMemoryManager) CreateSpace(ctx context.Context, name string) (string, error) {
	return "default", nil
}

func (m *muninMemoryManager) CreateSession(ctx context.Context, opts ...memorymanager.CreateSessionOption) (string, error) {
	options := memorymanager.NewCreateSessionOptions(opts...)

	id := fmt.Sprintf("session-%s-%d", options.SpaceId, m.counter.Add(1))

	m.mtx.Lock()
	defer m.mtx.Unlock()

	m.shortTerm[id] = []memorymanager.Message{}

	return id, nil
}

func (m *muninMemoryManager) AddShortTerm(ctx context.Context, sessionId string, role string, parts []memorymanager.Part, opts ...memorymanager.AddToShortTermOption) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if _, exists := m.shortTerm[sessionId]; !exists {
		return fmt.Errorf("session %s not found", sessionId)
	}

	m.shortTerm[sessionId] = append(m.shortTerm[sessionId], memorymanager.Message{
		SessionId: sessionId, Role: role, Parts: parts,
	})

	if len(m.shortTerm[sessionId]) > m.options.SessionWindowSize {
		m.shortTerm[sessionId] = m.shortTerm[sessionId][len(m.shortTerm[sessionId])-m.options.SessionWindowSize:]
	}

	return nil
}

func (m *muninMemoryManager) ListShortTerm(ctx context.Context, sessionId string, opts ...memorymanager.ListShortTermOption) ([]memorymanager.Message, []memorymanager.Task, error) {
	options := memorymanager.NewListShortTermOptions(opts...)

	m.mtx.RLock()
	defer m.mtx.RUnlock()

	history, exists := m.shortTerm[sessionId]
	if !exists {
		return nil, nil, fmt.Errorf("session %s not found", sessionId)
	}

	copied := make([]memorymanager.Message, len(history))
	copy(copied, history)

	if len(copied) > options.Limit {
		copied = copied[len(copied)-options.Limit:]
	}

	return copied, nil, nil
}

func (m *muninMemoryManager) FlushToLongTerm(ctx context.Context, sessionId string) error {
	m.mtx.RLock()
	history, exists := m.shortTerm[sessionId]
	m.mtx.RUnlock()

	if !exists {
		return fmt.Errorf("session %s not found", sessionId)
	}

	if len(history) == 0 {
		return nil
	}

	for _, msg := range history {
		var sb strings.Builder
		for _, p := range msg.Parts {
			sb.WriteString(p.Text)
		}
		raw := sb.String()
		if len(strings.TrimSpace(raw)) == 0 {
			continue
		}

		content := fmt.Sprintf("%s: %s", msg.Role, raw)

		vec, err := m.options.Embedder.Embed(ctx, content)
		if err != nil {
			return err
		}

		// no matter what the similarity score is from storer
		// check cosinesimilarity and skip if we already have good matches
		// unless the current best match is old
		candidates, _ := m.options.Storer.Search(ctx, vec, 1)
		shouldSave := true

		if len(candidates) > 0 {
			existing := candidates[0]
			sim := memorymanager.CosineSimilarity(vec, candidates[0].Embedding)
			if sim >= m.options.Thresholds.RejectionSimilarity {
				age := time.Now().UTC().Sub(existing.CreatedAt)
				if age < m.options.Thresholds.HalfLife {
					shouldSave = false
				}
			}
		}

		if !shouldSave {
			continue
		}

		meta := map[string]any{
			"source": msg.Role,
		}

		if err := m.options.Storer.Upsert(ctx, sessionId, content, meta, vec); err != nil {
			return err
		}
	}

	return nil
}

func (m *muninMemoryManager) SearchLongTerm(ctx context.Context, query string, opts ...memorymanager.SearchLongTermOption) ([]memorymanager.Message, []memorymanager.Skill, error) {
	options := memorymanager.NewSearchOptions(opts...)

	vec, err := m.options.Embedder.Embed(ctx, query)
	if err != nil {
		return nil, nil, err
	}

	candidates, err := m.options.Storer.Search(ctx, vec, options.Limit*4)
	if err != nil {
		return nil, nil, err
	}

	sim, rec := memorymanager.NormalizeWeights(m.options.Weights)
	now := time.Now().UTC()

	for i := range candidates {
		record := &candidates[i]

		score := float64(record.Score)

		age := now.Sub(record.CreatedAt)
		recency := math.Pow(0.5, age.Hours()/m.options.Thresholds.HalfLife.Hours())

		weighted := (sim * score) + (rec * recency)
		record.Score = float32(weighted)
	}

	selected := memorymanager.Select(candidates, vec, options.Limit, m.options.Thresholds.Relevance)

	sort.Slice(selected, func(i, j int) bool {
		return selected[i].Score > selected[j].Score
	})

	var messages []memorymanager.Message
	for _, rec := range selected {
		role := "default"
		if v, ok := rec.Metadata["source"]; ok {
			if s, ok := v.(string); ok {
				role = s
			}
		}
		msg := memorymanager.Message{
			SessionId: rec.SessionId,
			Role:      role,
			Parts: []memorymanager.Part{
				{
					Type: "text",
					Text: rec.Content,
					Meta: rec.Metadata,
				},
			},
		}
		messages = append(messages, msg)
	}

	return messages, nil, nil
}

func NewMemoryManager(opts ...memorymanager.Option) memorymanager.MemoryManager {
	options := memorymanager.NewOptions(opts...)

	m := &muninMemoryManager{
		options:   options,
		shortTerm: map[string][]memorymanager.Message{},
		mtx:       sync.RWMutex{},
	}

	return m
}
