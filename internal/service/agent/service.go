package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/w-h-a/agent/generator"
	"github.com/w-h-a/agent/retriever"
	toolhandler "github.com/w-h-a/agent/tool_handler"
)

const (
	defaultSystemPrompt = "You are the primary coordinator for an AI agent team. Provide concise, accurate answers and explain when you call tools or delegate work to specialist sub-agents"
)

type Service struct {
	retriever    retriever.Retriever
	generator    generator.Generator
	catalog      *Catalog
	contextLimit int
	systemPrompt string
}

func (s *Service) CreateSpace(ctx context.Context, name string) (string, error) {
	return s.retriever.CreateSpace(ctx, name)
}

func (s *Service) CreateSession(ctx context.Context, spaceId string) (string, error) {
	return s.retriever.CreateSession(ctx, retriever.WithSpaceId(spaceId))
}

func (s *Service) Respond(ctx context.Context, spaceId string, sessionId string, userInput string) (string, error) {
	if len(strings.TrimSpace(userInput)) == 0 {
		return "", errors.New("user input is required")
	}

	s.addShortTerm(ctx, sessionId, "user", userInput, nil)

	if handled, output, metadata, err := s.handleCommand(ctx, sessionId, userInput); handled {
		extra := map[string]any{"source": "tool"}
		if err != nil {
			s.addShortTerm(ctx, sessionId, "assistant", fmt.Sprintf("tool error: %v", err), extra)
			return "", err
		}
		for k, v := range metadata {
			if len(strings.TrimSpace(k)) == 0 {
				continue
			}
			extra[k] = v
		}
		s.addShortTerm(ctx, sessionId, "assistant", output, extra)
		return output, nil
	}

	prompt, err := s.buildPrompt(ctx, spaceId, sessionId, userInput)
	if err != nil {
		return "", err
	}

	result, err := s.generator.Generate(ctx, prompt)
	if err != nil {
		return "", err
	}

	s.addShortTerm(ctx, sessionId, "assistant", result, nil)

	return result, nil
}

func (s *Service) Flush(ctx context.Context, sessionId string) error {
	return s.retriever.FlushToLongTerm(ctx, sessionId)
}

func (s *Service) addShortTerm(ctx context.Context, sessionId string, role string, input string, meta map[string]any) {
	parts := []retriever.Part{
		{Type: "text", Text: input, Meta: meta},
	}

	// TODO: files

	s.retriever.AddShortTerm(ctx, sessionId, role, parts)
}

func (s *Service) buildPrompt(ctx context.Context, spaceId string, sessionId string, input string) (string, error) {
	// 1. Fetch Short-Term (Messages + Tasks)
	shortTermMsgs, tasks, err := s.retriever.ListShortTerm(
		ctx,
		sessionId,
		retriever.WithShortTermLimit(s.contextLimit),
	)
	if err != nil {
		return "", fmt.Errorf("short-term error: %w", err)
	}

	// 2. Fetch Long-Term (Messages + Skills)
	var longTermMsgs []retriever.Message
	var skills []retriever.Skill
	if len(spaceId) > 0 {
		var err error
		longTermMsgs, skills, err = s.retriever.SearchLongTerm(
			ctx,
			input,
			retriever.WithSearchLongTermSpaceId(spaceId),
			retriever.WithSearchLongTermLimit(s.contextLimit),
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
	sb.WriteString(s.systemPrompt)

	if specs := s.catalog.ListSpecs(); len(specs) > 0 {
		sb.WriteString("\n\nAvailable tools:\n")
		for _, spec := range specs {
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

func (s *Service) handleCommand(ctx context.Context, sessionId string, input string) (bool, string, map[string]any, error) {
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

	tp, spec, ok := s.catalog.Get(name)
	if !ok {
		return true, "", nil, fmt.Errorf("unknown tool: %s", name)
	}

	parsed := parseToolArguments(args)

	result, err := tp.Invoke(ctx, toolhandler.ToolRequest{
		SessionId: sessionId,
		Arguments: parsed,
	})
	if err != nil {
		return true, "", nil, err
	}

	metadata := map[string]any{"tool": spec.Name}
	for k, v := range result.Metadata {
		if len(strings.TrimSpace(k)) == 0 {
			continue
		}
		metadata[k] = v
	}

	s.addShortTerm(ctx, sessionId, "tool", fmt.Sprintf("%s => %s", spec.Name, strings.TrimSpace(result.Content)), metadata)

	return true, result.Content, metadata, nil
}

func New(
	retriever retriever.Retriever,
	generator generator.Generator,
	toolHandlers []toolhandler.ToolHandler,
	contextLimit int,
	systemPrompt string,
) *Service {
	catalog := &Catalog{
		tools: map[string]toolhandler.ToolHandler{},
		specs: map[string]toolhandler.ToolSpec{},
		order: []string{},
		mtx:   sync.RWMutex{},
	}

	for _, th := range toolHandlers {
		if th == nil {
			continue
		}
		if err := catalog.Register(th); err != nil {
			continue
		}
	}

	if contextLimit <= 0 {
		contextLimit = 8
	}

	if len(strings.TrimSpace(systemPrompt)) == 0 {
		systemPrompt = defaultSystemPrompt
	}

	return &Service{
		retriever:    retriever,
		generator:    generator,
		catalog:      catalog,
		contextLimit: contextLimit,
		systemPrompt: systemPrompt,
	}
}
