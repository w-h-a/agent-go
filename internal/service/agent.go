package service

import (
	"bytes"
	"context"
	"encoding/json"
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
	toolSpecs     map[string]toolprovider.ToolSpec
	contextLimit  int
	systemPrompt  string
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

	if handled, output, metadata, err := a.handleCommand(ctx, sessionId, userInput); handled {
		extra := map[string]any{"source": "tool"}
		if err != nil {
			a.addShortTerm(ctx, sessionId, "assistant", fmt.Sprintf("tool error: %v", err), extra)
			return "", err
		}
		for k, v := range metadata {
			if len(strings.TrimSpace(k)) == 0 {
				continue
			}
			extra[k] = v
		}
		a.addShortTerm(ctx, sessionId, "assistant", output, extra)
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

func (a *Agent) handleCommand(ctx context.Context, sessionId string, input string) (bool, string, map[string]any, error) {
	trimmed := strings.TrimSpace(input)
	lower := strings.ToLower(trimmed)

	if !strings.HasPrefix(lower, "tool:") {
		return false, "", nil, nil
	}

	payload := strings.TrimSpace(trimmed[len("tool:"):])
	if len(payload) == 0 {
		return true, "", nil, errors.New("tool name is missing")
	}

	name, args := splitCommand(payload)

	if len(a.toolProviders) == 0 {
		return true, "", nil, errors.New("no tools available")
	}

	tp, ok := a.toolProviders[strings.ToLower(name)]
	if !ok {
		return true, "", nil, fmt.Errorf("unknown tool: %s", name)
	}

	parsed := parseToolArguments(args)

	result, err := tp.Invoke(ctx, toolprovider.ToolRequest{
		SessionId: sessionId,
		Arguments: parsed,
	})
	if err != nil {
		return true, "", nil, err
	}

	spec := a.toolSpecs[strings.ToLower(name)]
	metadata := map[string]any{"tool": spec.Name}
	for k, v := range result.Metadata {
		if len(strings.TrimSpace(k)) == 0 {
			continue
		}
		metadata[k] = v
	}

	a.addShortTerm(ctx, sessionId, "tool", fmt.Sprintf("%s => %s", spec.Name, strings.TrimSpace(result.Content)), metadata)

	return true, result.Content, metadata, nil
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
	var longTermMsgs []retriever.Message
	var skills []retriever.Skill
	if len(spaceId) > 0 {
		var err error
		longTermMsgs, skills, err = a.retriever.SearchLongTerm(
			ctx,
			input,
			retriever.WithSearchLongTermSpaceId(spaceId),
			retriever.WithSearchLongTermLimit(a.contextLimit),
		)
		if err != nil {
			return "", fmt.Errorf("long-term error: %w", err)
		}
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
		for k := range a.toolProviders {
			spec := a.toolSpecs[k]
			sb.WriteString(fmt.Sprintf("- %s: %s\n", spec.Name, spec.Description))
			if len(spec.InputSchema) > 0 {
				schemaJSON, _ := json.MarshalIndent(spec.InputSchema, "  ", "  ")
				sb.WriteString("  Input schema: ")
				sb.Write(schemaJSON)
				sb.WriteString("\n")
			}
			if len(spec.Examples) > 0 {
				sb.WriteString("  Examples:\n")
				for _, ex := range spec.Examples {
					exJSON, _ := json.MarshalIndent(ex, "    ", "  ")
					sb.Write(exJSON)
					sb.WriteString("\n")
				}
			}
		}
		sb.WriteString("Invoke a tool by replying with the format `tool:<name> <json arguments>` when it improves the answer.\n")
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

func NewAgent(
	retriever retriever.Retriever,
	generator generator.Generator,
	toolProviders map[string]toolprovider.ToolProvider,
	contextLimit int,
	systemPrompt string,
) *Agent {
	if retriever == nil {
		panic("retriever is required")
	}

	if generator == nil {
		panic("generator is required")
	}

	tps := make(map[string]toolprovider.ToolProvider, len(toolProviders))
	tss := make(map[string]toolprovider.ToolSpec, len(toolProviders))
	for _, tp := range toolProviders {
		spec := tp.Spec()
		key := strings.ToLower(strings.TrimSpace(spec.Name))
		if len(key) == 0 {
			continue
		}
		tps[key] = tp
		tss[key] = spec
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
		toolProviders: tps,
		toolSpecs:     tss,
		contextLimit:  contextLimit,
		systemPrompt:  systemPrompt,
	}
}
