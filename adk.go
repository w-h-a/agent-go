package agent

import (
	"context"

	"github.com/w-h-a/agent/generator"
	"github.com/w-h-a/agent/internal/service/agent"
	"github.com/w-h-a/agent/internal/service/session"
	"github.com/w-h-a/agent/retriever"
	toolprovider "github.com/w-h-a/agent/tool_provider"
)

type ADK struct {
	agent   *agent.Service
	session *session.Service
}

// TODO: Space

func (a *ADK) NewSession(ctx context.Context, sessionId string) (*session.Session, error) {
	return a.session.CreateSession(ctx, sessionId)
}

func (a *ADK) ListSessionIds() []string {
	return a.session.ListSessionIds()
}

func (a *ADK) GetSession(id string) (*session.Session, error) {
	return a.session.GetSession(id)
}

func (a *ADK) DeleteSession(id string) {
	a.session.DeleteSession(id)
}

func (a *ADK) Generate(ctx context.Context, spaceId string, sessionId string, userInput string) (string, error) {
	return a.agent.Respond(ctx, spaceId, sessionId, userInput)
}

func (a *ADK) Close() error {
	// TODO: implement
	return nil
}

func New(
	retriever retriever.Retriever,
	generator generator.Generator,
	toolProviders []toolprovider.ToolProvider,
	context int,
	systemPrompt string,
) *ADK {
	agent := agent.New(
		retriever,
		generator,
		toolProviders,
		context,
		systemPrompt,
	)

	session := session.New(
		retriever,
	)

	adk := &ADK{
		agent:   agent,
		session: session,
	}

	return adk
}
