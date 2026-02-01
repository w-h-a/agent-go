package session

import (
	"context"

	"github.com/w-h-a/agent/retriever"
)

type Session struct {
	retriever retriever.Retriever
	id        string
}

func (s *Session) ID() string {
	return s.id
}

func (s *Session) Flush(ctx context.Context) error {
	return s.retriever.FlushToLongTerm(ctx, s.id)
}
