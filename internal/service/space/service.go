package space

import (
	"context"
	"fmt"
	"strings"
	"sync"

	memorymanager "github.com/w-h-a/agent/memory_manager"
)

type Service struct {
	memory memorymanager.MemoryManager
	spaces map[string]*Space
	mtx    sync.RWMutex
}

func (s *Service) CreateSpace(ctx context.Context, name string, id string) (*Space, error) {
	if len(strings.TrimSpace(id)) == 0 {
		var err error
		id, err = s.memory.CreateSpace(ctx, name)
		if err != nil {
			return nil, err
		}
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	if space, ok := s.spaces[id]; ok {
		return space, nil
	}

	space := &Space{
		id:   id,
		name: name,
	}

	s.spaces[id] = space

	return space, nil
}

func (s *Service) ListSpaceIds(ctx context.Context) ([]string, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	ids := make([]string, 0, len(s.spaces))
	for id := range s.spaces {
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *Service) GetSpace(ctx context.Context, id string) (*Space, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	space, ok := s.spaces[id]
	if !ok {
		return nil, fmt.Errorf("space %s not found", id)
	}
	return space, nil
}

func (s *Service) DeleteSpace(ctx context.Context, id string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	delete(s.spaces, id)
}

func New(
	memory memorymanager.MemoryManager,
) *Service {
	return &Service{
		memory: memory,
		spaces: map[string]*Space{},
		mtx:    sync.RWMutex{},
	}
}
