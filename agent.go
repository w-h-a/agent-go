package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/w-h-a/agent/generator"
	"github.com/w-h-a/agent/retriever"
	toolprovider "github.com/w-h-a/agent/tool_provider"
)

const (
	defaultSystemPrompt = "You are the primary coordinator for an AI agent team. Provide concise, accurate answers and explain when you call tools or delegate work to specialist sub-agents"
)

type Agent struct {
	retriever     retriever.Retriever
	generator     generator.Generator
	toolProviders map[string]toolprovider.ToolProvider
	systemPrompt  string
	contextLimit  int
}

func (a *Agent) CreateSpace(ctx context.Context, name string) (string, error) {
	return a.retriever.CreateSpace(ctx, name)
}

func (a *Agent) CreateSession(ctx context.Context, spaceId string) (string, error) {
	return a.retriever.CreateSession(ctx, retriever.WithSpaceId(spaceId))
}

func (a *Agent) Respond(ctx context.Context, spaceId string, sessionId string, userInput string) (string, error) {
	if len(strings.TrimSpace(userInput)) == 0 {
		return "", errors.New("user input is required")
	}

	a.addShortTerm(ctx, sessionId, "user", userInput, nil)

	if handled, output, err := a.handleCommand(ctx, sessionId, userInput); handled {
		if err != nil {
			a.addShortTerm(ctx, sessionId, "assistant", fmt.Sprintf("tool error: %v", err), map[string]any{"source": "tool"})
			return "", err
		}
		a.addShortTerm(ctx, sessionId, "assistant", output, map[string]any{"source": "tool"})
		return output, nil
	}

	prompt, err := a.buildPrompt(ctx, spaceId, sessionId, userInput)
	if err != nil {
		return "", err
	}

	result, err := a.generator.Generate(ctx, prompt)
	if err != nil {
		return "", err
	}

	a.addShortTerm(ctx, sessionId, "assistant", result, nil)

	return result, nil
}

func (a *Agent) Flush(ctx context.Context, sessionId string) error {
	return a.retriever.FlushToLongTerm(ctx, sessionId)
}

func (a *Agent) addShortTerm(ctx context.Context, sessionId string, role string, input string, meta map[string]any) {
	parts := []retriever.Part{
		{Type: "text", Text: input, Meta: meta},
	}

	// TODO: files

	a.retriever.AddShortTerm(ctx, sessionId, role, parts)
}

func (a *Agent) handleCommand(ctx context.Context, sessionId string, input string) (bool, string, error) {
	trimmed := strings.TrimSpace(input)
	lower := strings.ToLower(trimmed)

	if !strings.HasPrefix(lower, "tool:") {
		return false, "", nil
	}

	payload := strings.TrimSpace(trimmed[len("tool:"):])
	if len(payload) == 0 {
		return true, "", errors.New("tool name is missing")
	}

	name, args := splitCommand(payload)

	if len(a.toolProviders) == 0 {
		return true, "", errors.New("no tools available")
	}

	tp, ok := a.toolProviders[name]
	if !ok {
		return true, "", fmt.Errorf("unknown tool: %s", name)
	}

	result, err := tp.Run(ctx, args)
	if err != nil {
		return true, "", err
	}

	a.addShortTerm(ctx, sessionId, "tool", fmt.Sprintf("%s => %s", tp.Name(), strings.TrimSpace(result)), map[string]any{"tool": tp.Name()})

	return true, result, nil
}

func (a *Agent) buildPrompt(ctx context.Context, spaceId string, sessionId string, input string) (string, error) {
	// 1. Fetch Short-Term (Messages + Tasks)
	shortTermMsgs, tasks, err := a.retriever.ListShortTerm(
		ctx,
		sessionId,
		retriever.WithShortTermLimit(a.contextLimit),
	)
	if err != nil {
		return "", fmt.Errorf("short-term error: %w", err)
	}

	// 2. Fetch Long-Term (Messages + Skills)
	longTermMsgs, skills, err := a.retriever.SearchLongTerm(
		ctx,
		input,
		retriever.WithSearchLongTermSpaceId(spaceId),
		retriever.WithSearchLongTermLimit(a.contextLimit),
	)
	if err != nil {
		return "", fmt.Errorf("long-term error: %w", err)
	}

	// 3. Deduplicate Messages (Favor Long-Term)
	isRelevant := make(map[string]bool)
	for _, msg := range longTermMsgs {
		isRelevant[msg.Id] = true
	}

	var uniqueShortTerm []retriever.Message
	for _, msg := range shortTermMsgs {
		if !isRelevant[msg.Id] {
			uniqueShortTerm = append(uniqueShortTerm, msg)
		}
	}

	// 4. Build Prompt
	var sb bytes.Buffer
	sb.WriteString(a.systemPrompt)

	if len(a.toolProviders) > 0 {
		sb.WriteString("\n\nAvailable tools:\n")
		for _, tool := range a.toolProviders {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))
		}
		sb.WriteString("Invoke a tool by replying with the exact format `tool:<name> <input>` when it improves the answer.\n")
	}

	if len(skills) > 0 {
		sb.WriteString("\nRelevant Skills (SOPs):\n")
		for i, skill := range skills {
			sb.WriteString(fmt.Sprintf("%d. TRIGGER: %s\n  SOP: %s\n", i+1, skill.Trigger, skill.SOP))
		}
	}

	if len(longTermMsgs) > 0 {
		sb.WriteString("\nRelevant Memories:\n")
		for i, msg := range longTermMsgs {
			if len(msg.Parts) > 0 {
				sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, msg.Role, msg.Parts[0].Text))
			}
		}
	}

	if len(tasks) > 0 {
		sb.WriteString("\nCurrent Tasks / To-Do:\n")
		for i, task := range tasks {
			sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, task.Status, string(task.Data)))
		}
	}

	if len(uniqueShortTerm) > 0 {
		sb.WriteString("\nConversation History:\n")
		for i := len(uniqueShortTerm) - 1; i >= 0; i-- {
			msg := uniqueShortTerm[i]
			if len(msg.Parts) > 0 {
				sb.WriteString(fmt.Sprintf("[%s]: %s\n", msg.Role, msg.Parts[0].Text))
			}
		}
	}

	sb.WriteString("\nCurrent user message:\n")
	sb.WriteString(strings.TrimSpace(input))
	sb.WriteString("\n\nCompose the best possible assistant reply.\n")

	return sb.String(), nil
}

func New(
	retriever retriever.Retriever,
	generator generator.Generator,
	toolProviders map[string]toolprovider.ToolProvider,
	systemPrompt string,
	contextLimit int,
) *Agent {
	if retriever == nil {
		panic("retriever is required")
	}

	if generator == nil {
		panic("generator is required")
	}

	if contextLimit <= 0 {
		contextLimit = 8
	}

	if len(strings.TrimSpace(systemPrompt)) == 0 {
		systemPrompt = defaultSystemPrompt
	}

	return &Agent{
		retriever:     retriever,
		generator:     generator,
		toolProviders: toolProviders,
		systemPrompt:  systemPrompt,
		contextLimit:  contextLimit,
	}
}
