package agent

import (
	"context"

	"github.com/w-h-a/agent/generator"
	"github.com/w-h-a/agent/internal/service/agent"
	"github.com/w-h-a/agent/internal/service/session"
	"github.com/w-h-a/agent/internal/service/space"
	memorymanager "github.com/w-h-a/agent/memory_manager"
	toolhandler "github.com/w-h-a/agent/tool_handler"
)

type ADK struct {
	agent   *agent.Service
	space   *space.Service
	session *session.Service
}

func (a *ADK) CreateSpace(ctx context.Context, name string, id string) (string, error) {
	space, err := a.space.CreateSpace(ctx, name, id)
	if err != nil {
		return "", err
	}
	return space.ID(), nil
}

func (a *ADK) ListSpaceIds(ctx context.Context) ([]string, error) {
	return a.space.ListSpaceIds(ctx)
}

func (a *ADK) GetSpaceName(ctx context.Context, id string) (string, error) {
	space, err := a.space.GetSpace(ctx, id)
	if err != nil {
		return "", err
	}
	return space.Name(), nil
}

func (a *ADK) DeleteSpace(ctx context.Context, id string) {
	a.space.DeleteSpace(ctx, id)
}

func (a *ADK) CreateSession(ctx context.Context, sessionId string, spaceId string) (string, error) {
	session, err := a.session.CreateSession(ctx, sessionId, spaceId)
	if err != nil {
		return "", err
	}
	return session.ID(), nil
}

func (a *ADK) ListSessionIds(ctx context.Context) ([]string, error) {
	return a.session.ListSessionIds(ctx)
}

func (a *ADK) GetSessionSpaceId(ctx context.Context, id string) (string, error) {
	session, err := a.session.GetSession(ctx, id)
	if err != nil {
		return "", err
	}
	return session.SpaceId(), nil
}

func (a *ADK) DeleteSession(ctx context.Context, id string) {
	a.session.DeleteSession(ctx, id)
}

func (a *ADK) Generate(ctx context.Context, sessionId string, userInput string) (string, error) {
	return a.agent.Respond(ctx, sessionId, userInput)
}

func (a *ADK) FlushSession(ctx context.Context, sessionId string) error {
	return a.agent.Flush(ctx, sessionId)
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

	space := space.New(
		memory,
	)

	session := session.New(
		memory,
	)

	adk := &ADK{
		agent:   agent,
		space:   space,
		session: session,
	}

	return adk
}
