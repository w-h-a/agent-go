package memorymanager

import "context"

type MemoryManager interface {
	CreateSpace(ctx context.Context, name string) (string, error)
	CreateSession(ctx context.Context, opts ...CreateSessionOption) (string, error)
	AddShortTerm(ctx context.Context, sessionId string, role string, parts []Part, opts ...AddToShortTermOption) error
	ListShortTerm(ctx context.Context, sessionId string, opts ...ListShortTermOption) ([]Message, []Task, error)
	FlushToLongTerm(ctx context.Context, sessionId string) error
	SearchLongTerm(ctx context.Context, query string, opts ...SearchLongTermOption) ([]Message, []Skill, error)
}
