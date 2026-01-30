package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	gotime "time"

	"github.com/w-h-a/agent/pkg/generator"
	"github.com/w-h-a/agent/pkg/generator/anthropic"
	"github.com/w-h-a/agent/pkg/generator/google"
	"github.com/w-h-a/agent/pkg/generator/openai"
	"github.com/w-h-a/agent/pkg/retriever"
	"github.com/w-h-a/agent/pkg/retriever/gomento"
	"github.com/w-h-a/agent/pkg/retriever/postgres"
	toolprovider "github.com/w-h-a/agent/pkg/tool_provider"
	"github.com/w-h-a/agent/pkg/tool_provider/calculator"
	"github.com/w-h-a/agent/pkg/tool_provider/echo"
	"github.com/w-h-a/agent/pkg/tool_provider/time"
)

const (
	gomentoURL = "http://localhost:4000"
	memoryPG   = "postgres://user:password@localhost:5432/memory?sslmode=disable"
)

func main() {
	ctx := context.Background()

	// 1. Initialize the Retriever
	r := initRetriever("postgres")

	// 2. Initialize a Generator (Model) for the Agent
	_ = initPrimaryModel("openai")

	// 3. Initialize a Generator (Model) for the Sub-Agent(s?)
	_ = initSubModels("openai")

	// 4. Initialize the Tool Providers (revisit)
	_ = []toolprovider.ToolProvider{
		echo.NewToolProvider(),
		calculator.NewToolProvider(),
		time.NewToolProvider(),
	}

	// 5. Create Agent and Sub-Agents

	// 6. Initialize The (Optional) Space
	spaceID, err := r.CreateSpace(ctx, "agent-learning-space")
	if err != nil {
		log.Fatalf("❌ failed to create space: %v", err)
	}
	fmt.Printf("✅ Connected to Space: %s\n", spaceID)

	// 7. Initialize The Session
	sessionID, err := r.CreateSession(ctx, retriever.WithSpaceId(spaceID))
	if err != nil {
		log.Fatalf("❌ failed to create session: %v", err)
	}
	fmt.Printf("✅ Started Session: %s\n", sessionID)

	// 8. Simulate Conversation
	inputs := []string{
		"User: I want to build a Go agent that uses pgvector.",
		"Agent: That sounds great. You should use the pgvector-go library.",
		"User: How do I optimize the search query?",
		"Agent: You should use an HNSW index for performance.",
	}

	for _, text := range inputs {
		role := "user"
		if len(text) > 5 && text[:5] == "Agent" {
			role = "assistant"
		}
		parts := []retriever.Part{
			{Type: "text", Text: text},
		}

		if err := r.AddShortTerm(ctx, sessionID, role, parts); err != nil {
			log.Printf("⚠️ failed to add memory: %v", err)
		}
	}
	fmt.Println("✅ Populated conversation history.")

	// 9. Consolidate Memory
	fmt.Println("⏳ Flushing to Long-Term...")
	if err := r.FlushToLongTerm(ctx, sessionID); err != nil {
		log.Printf("⚠️ failed to flush: %v", err)
	}

	gotime.Sleep(10 * gotime.Second)

	// 10. Retrieve Context & Build Prompt
	userQuery := "What index did you recommend for vector search?"

	prompt, err := retrieveContextAndBuildPrompt(ctx, r, spaceID, sessionID, userQuery)
	if err != nil {
		log.Fatalf("❌ failed to build prompt: %v", err)
	}

	fmt.Println("\n----- Final LLM Prompt -----")
	fmt.Println(prompt)
	fmt.Println("----------------------------")
}

func retrieveContextAndBuildPrompt(ctx context.Context, r retriever.Retriever, spaceID, sessionID, query string) (string, error) {
	// 1. Fetch Short-Term (Messages + Tasks)
	shortTermMsgs, tasks, err := r.ListShortTerm(ctx, sessionID, retriever.WithShortTermLimit(5))
	if err != nil {
		return "", fmt.Errorf("short-term error: %w", err)
	}

	// 2. Fetch Long-Term (Messages + Skills)
	longTermMsgs, skills, err := r.SearchLongTerm(ctx, query, retriever.WithSearchLongTermSpaceId(spaceID), retriever.WithSearchLongTermLimit(5))
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
	sb.WriteString("System: You are a helpful assistant.\n")

	if len(skills) > 0 {
		sb.WriteString("\nAcquired Skills (SOPs):\n")
		for _, skill := range skills {
			sb.WriteString(fmt.Sprintf("- TRIGGER: %s\n  SOP: %s\n", skill.Trigger, skill.SOP))
		}
	}

	if len(longTermMsgs) > 0 {
		sb.WriteString("\nRelevant Memories:\n")
		for _, msg := range longTermMsgs {
			if len(msg.Parts) > 0 {
				sb.WriteString(fmt.Sprintf("- %s\n", msg.Parts[0].Text))
			}
		}
	}

	if len(tasks) > 0 {
		sb.WriteString("\nCurrent Tasks / To-Do:\n")
		for _, task := range tasks {
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", task.Status, string(task.Data)))
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

	sb.WriteString(fmt.Sprintf("\nUser: %s", query))
	return sb.String(), nil
}

func initRetriever(choice string) retriever.Retriever {
	switch choice {
	case "gomento":
		return gomento.NewRetriever(
			retriever.WithLocation(gomentoURL),
		)
	case "postgres":
		return postgres.NewRetriever(
			retriever.WithLocation(memoryPG),
			retriever.WithApiKey(""),
			retriever.WithModel("text-embedding-3-small"),
			retriever.WithShortTermMemorySize(5),
		)
	default:
		panic("unknown retriever choice")
	}
}

func initPrimaryModel(choice string) generator.Generator {
	switch choice {
	case "openai":
		return openai.NewGenerator(
			generator.WithApiKey(""),
			generator.WithModel("gpt-3.5-turbo"),
			generator.WithPromptPrefix(""),
		)
	case "google":
		return google.NewGenerator(
			generator.WithApiKey(""),
			generator.WithModel(""),
			generator.WithPromptPrefix(""),
		)
	case "anthropic":
		return anthropic.NewGenerator(
			generator.WithApiKey(""),
			generator.WithModel(""),
			generator.WithPromptPrefix(""),
		)
	default:
		panic("unknown model choice")
	}
}

func initSubModels(choice string) []generator.Generator {
	switch choice {
	case "openai":
		return []generator.Generator{openai.NewGenerator(
			generator.WithApiKey(""),
			generator.WithModel("gpt-3.5-turbo"),
			generator.WithPromptPrefix(""),
		)}
	case "google":
		return []generator.Generator{google.NewGenerator(
			generator.WithApiKey(""),
			generator.WithModel(""),
			generator.WithPromptPrefix(""),
		)}
	case "anthropic":
		return []generator.Generator{anthropic.NewGenerator(
			generator.WithApiKey(""),
			generator.WithModel(""),
			generator.WithPromptPrefix(""),
		)}
	default:
		panic("unknown model choice")
	}
}
