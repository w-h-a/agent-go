package session

import (
	"context"

	memorymanager "github.com/w-h-a/agent/memory_manager"
)

type Session struct {
	memory memorymanager.MemoryManager
	id     string
}

func (s *Session) ID() string {
	return s.id
}

func (s *Session) Flush(ctx context.Context) error {
	return s.memory.FlushToLongTerm(ctx, s.id)
}
