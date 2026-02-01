package session

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/w-h-a/agent/retriever"
)

type Service struct {
	retriever retriever.Retriever
	sessions  map[string]*Session
	mtx       sync.RWMutex
}

func (s *Service) CreateSession(ctx context.Context, id string) (*Session, error) {
	if len(strings.TrimSpace(id)) == 0 {
		var err error
		id, err = s.retriever.CreateSession(ctx)
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
		retriever: s.retriever,
		id:        id,
	}

	s.sessions[id] = session

	return session, nil
}

func (s *Service) ListSessionIds() []string {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	ids := make([]string, 0, len(s.sessions))
	for id := range s.sessions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (s *Service) GetSession(id string) (*Session, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session %s not found", id)
	}
	return session, nil
}

func (s *Service) DeleteSession(id string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	delete(s.sessions, id)
}

func New(
	retriever retriever.Retriever,
) *Service {
	return &Service{
		retriever: retriever,
		sessions:  map[string]*Session{},
		mtx:       sync.RWMutex{},
	}
}
