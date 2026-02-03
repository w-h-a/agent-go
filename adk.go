package agent

import (
	"context"

	"github.com/w-h-a/agent/generator"
	"github.com/w-h-a/agent/internal/service/agent"
	"github.com/w-h-a/agent/internal/service/session"
	memorymanager "github.com/w-h-a/agent/memory_manager"
	toolhandler "github.com/w-h-a/agent/tool_handler"
)

type ADK struct {
	agent   *agent.Service
	session *session.Service
}

// TODO: Space

func (a *ADK) NewSession(ctx context.Context, sessionId string) (string, func(context.Context) error, error) {
	session, err := a.session.CreateSession(ctx, sessionId)
	if err != nil {
		return "", nil, err
	}

	return session.ID(), session.Flush, nil
}

func (a *ADK) ListSessionIds(ctx context.Context) []string {
	return a.session.ListSessionIds(ctx)
}

func (a *ADK) DeleteSession(ctx context.Context, id string) {
	a.session.DeleteSession(ctx, id)
}

func (a *ADK) FlushSession(ctx context.Context, id string) error {
	session, err := a.session.GetSession(ctx, id)
	if err != nil {
		return err
	}
	return session.Flush(ctx)
}

func (a *ADK) Generate(ctx context.Context, spaceId string, sessionId string, userInput string) (string, error) {
	return a.agent.Respond(ctx, spaceId, sessionId, userInput)
}

func (a *ADK) Close() error {
	// TODO: implement
	return nil
}

func New(
	memory memorymanager.MemoryManager,
	generator generator.Generator,
	toolHandlers []toolhandler.ToolHandler,
	context int,
	systemPrompt string,
) *ADK {
	agent := agent.New(
		memory,
		generator,
		toolHandlers,
		context,
		systemPrompt,
	)

	session := session.New(
		memory,
	)

	adk := &ADK{
		agent:   agent,
		session: session,
	}

	return adk
}
