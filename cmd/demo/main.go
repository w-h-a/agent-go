package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/w-h-a/agent"
	"github.com/w-h-a/agent/cmd/demo/tools/calculate"
	"github.com/w-h-a/agent/cmd/demo/tools/research"
	"github.com/w-h-a/agent/generator"
	"github.com/w-h-a/agent/generator/openai"
	"github.com/w-h-a/agent/retriever"
	"github.com/w-h-a/agent/retriever/gomento"
	toolprovider "github.com/w-h-a/agent/tool_provider"
)

var (
	cfg struct {
		// Retriever config
		RetrieverLocation string `help:"Address of memory store for retriever client" default:"http://localhost:4000"`
		// RetrieverLocation string `help:"Address of memory store for retriever client" default:"postgres://user:password@localhost:5432/memory?sslmode=disable"`
		Window   int    `help:"Short-term memory window size per session" default:"8"`
		Embedder string `help:"Model identifier for vector embeddings" default:"text-embedding-3-small"`

		// Generator config
		APIKey string `help:"API Key for the model" default:""`
		Model  string `help:"Model identifier for primary" default:"gpt-3.5-turbo"`

		// Agent config
		Context      int    `help:"Number of conversation turns to send to the model" default:"6"`
		SystemPrompt string `help:"System prompt for the agent" default:"You orchestrate tooling and specialists to help the user build AI agents."`

		// Session config
		Session string `help:"Optional fixed session identifier" default:""`
	}
)

func main() {
	// Parse inputs
	_ = kong.Parse(&cfg)
	ctx := context.Background()

	// Create retriever
	re := gomento.NewRetriever(
		retriever.WithLocation(cfg.RetrieverLocation),
	)

	// re := postgres.NewRetriever(
	// 	retriever.WithLocation(cfg.RetrieverLocation),
	// 	retriever.WithApiKey(cfg.APIKey),
	// 	retriever.WithModel(cfg.Embedder),
	// 	retriever.WithShortTermMemorySize(cfg.Window),
	// )

	// Create primary agent's model
	primaryModel := openai.NewGenerator(
		generator.WithApiKey(cfg.APIKey),
		generator.WithModel(cfg.Model),
		generator.WithPromptPrefix("Coordinator response:"),
	)

	// Create custom tooling
	calculate := calculate.NewToolProvider()

	researchModel := openai.NewGenerator(
		generator.WithApiKey(cfg.APIKey),
		generator.WithModel(cfg.Model),
		generator.WithPromptPrefix("Researcher response:"),
	)
	research := research.NewToolProvider(
		toolprovider.WithGenerator(researchModel),
	)

	// Create ADK
	adk := agent.New(
		ctx,
		re,
		primaryModel,
		map[string]toolprovider.ToolProvider{
			"calculate": calculate,
			"research":  research,
		},
		cfg.Context,
		cfg.SystemPrompt,
	)
	defer adk.Close()

	fmt.Println("--- Agent Development Kit Demo ---")

	// TODO: support spaces
	// spaceId, err := a.CreateSpace(ctx, "agent-learning-space")
	// if err != nil {
	// 	log.Fatalf("❌ failed to create space: %v", err)
	// }
	// fmt.Printf("✅ Connected to Space: %s\n", spaceId)

	// 1. Start session
	session, err := adk.NewSession(ctx, cfg.Session)
	if err != nil {
		log.Fatalf("❌ failed to start session: %v", err)
	}
	defer session.Flush(ctx)
	sessionId := session.ID()
	fmt.Printf("✅ Started Session: %s\n", sessionId)

	// 2. Simulate Conversation
	prompts := []string{
		"Summarize what I asked in our previous session.",
		"I want to design an AI agent with memory. What’s the first step?",
		"tool:calculate 21 / 3",
		"subagent:researcher Briefly explain pgvector and its benefits for retrieval.",
	}

	for _, prompt := range prompts {
		start := time.Now()
		reply, err := session.Ask(ctx, prompt)
		duration := time.Since(start)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}
		fmt.Printf("User: %s\nAgent: %s\n(%.2fs)\n\n", prompt, reply, duration.Seconds())
	}
	fmt.Println("✅ Populated conversation history.")

	fmt.Println("💾 All interactions flushed to long-term memory.")
}
