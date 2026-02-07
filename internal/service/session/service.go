package session

import (
	"context"
	"fmt"
	"strings"
	"sync"

	memorymanager "github.com/w-h-a/agent/memory_manager"
)

type Service struct {
	memory   memorymanager.MemoryManager
	sessions map[string]*Session
	mtx      sync.RWMutex
}

func (s *Service) CreateSession(ctx context.Context, id string, spaceId string) (*Session, error) {
	if len(strings.TrimSpace(id)) == 0 {
		var err error
		opts := []memorymanager.CreateSessionOption{}
		if len(spaceId) > 0 {
			opts = append(opts, memorymanager.WithSpaceId(spaceId))
		}
		id, err = s.memory.CreateSession(ctx, opts...)
		if err != nil {
			return nil, err
		}
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	if session, ok := s.sessions[id]; ok {
		return session, nil
	}

	session := &Session{
		id:      id,
		spaceId: spaceId,
	}

	s.sessions[id] = session

	return session, nil
}

func (s *Service) ListSessionIds(ctx context.Context) ([]string, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	ids := make([]string, 0, len(s.sessions))
	for id := range s.sessions {
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *Service) GetSession(ctx context.Context, id string) (*Session, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session %s not found", id)
	}
	return session, nil
}

func (s *Service) DeleteSession(ctx context.Context, id string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	delete(s.sessions, id)
}

func New(
	memory memorymanager.MemoryManager,
) *Service {
	return &Service{
		memory:   memory,
		sessions: map[string]*Session{},
		mtx:      sync.RWMutex{},
	}
}
