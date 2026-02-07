package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/gorilla/mux"
	"github.com/w-h-a/agent"
	"github.com/w-h-a/agent/cmd/demo/handler/tool"
	"github.com/w-h-a/agent/cmd/demo/tool/calculate"
	"github.com/w-h-a/agent/cmd/demo/tool/research"
	"github.com/w-h-a/agent/generator"
	openaigenerator "github.com/w-h-a/agent/generator/openai"
	memorymanager "github.com/w-h-a/agent/memory_manager"
	"github.com/w-h-a/agent/memory_manager/gomento"
	"github.com/w-h-a/agent/server"
	httpserver "github.com/w-h-a/agent/server/http"
	toolhandler "github.com/w-h-a/agent/tool_handler"
	toolprovider "github.com/w-h-a/agent/tool_provider"
	"github.com/w-h-a/agent/tool_provider/utcp"
)

var (
	cfg struct {
		// Memory config
		MemoryLocation string `help:"Address of memory store for memory manager" default:"http://localhost:4000"`
		// MemoryLocation string `help:"Address of memory store for memory manager" default:"postgres://user:password@localhost:5432/memory?sslmode=disable"`
		// MemoryLocation string `help:"Address of memory store for memory manager" default:"http://localhost:6333"`
		Window      int    `help:"Short-term memory window size per session" default:"8"`
		EmbedderKey string `help:"API Key for the embedder" default:""`
		Embedder    string `help:"Model identifier for embedder" default:"text-embedding-3-small"`

		// Generator config
		GeneratorKey string `help:"API Key for the generator" default:""`
		Generator    string `help:"Model identifier for generator" default:"gpt-3.5-turbo"`

		// Tool Provider config
		ToolProviderClientAddrs []string `help:"List of addresses of servers with exposed tool handlers" default:"http://localhost:8080/tools"`

		// Agent config
		Context      int    `help:"Number of conversation turns to send to the model" default:"6"`
		SystemPrompt string `help:"System prompt for the agent" default:"You orchestrate tooling and specialists to help the user build AI agents."`

		// Space config
		Space string `help:"Option space identifier" default:""`

		// Session config
		Session string `help:"Optional fixed session identifier" default:""`
	}
)

func main() {
	// Parse inputs
	_ = kong.Parse(&cfg)
	ctx := context.Background()

	// Start server for dynamic tool handling
	stop := make(chan struct{})
	srv, err := initHttpServer(ctx, ":8080")
	if err != nil {
		log.Fatalf("❌ failed to init http server: %v", err)
	}
	var wg sync.WaitGroup
	errCh := make(chan error, 1)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	wg.Add(1)
	go func() {
		defer wg.Done()
		errCh <- srv.Run(stop)
	}()
	time.Sleep(200 * time.Millisecond)

	// Create memory manager
	re := gomento.NewMemoryManager(
		memorymanager.WithLocation(cfg.MemoryLocation),
	)

	// re := munin.NewMemoryManager(
	// 	memorymanager.WithStorer(
	// 		postgres.NewStorer(
	// 			storer.WithLocation(cfg.MemoryLocation),
	// 			storer.WithCollection("agent_memory"),
	// 			storer.WithVectorSize(1536),
	// 		),
	// 	),
	// 	memorymanager.WithEmbedder(
	// 		openaiembedder.NewEmbedder(
	// 			embedder.WithApiKey(cfg.EmbedderKey),
	// 			embedder.WithModel(cfg.Embedder),
	// 		),
	// 	),
	// )

	// Create primary agent's model
	primaryModel := openaigenerator.NewGenerator(
		generator.WithApiKey(cfg.GeneratorKey),
		generator.WithModel(cfg.Generator),
		generator.WithPromptPrefix("Coordinator response:"),
	)

	// Load dynamic tooling
	var dynamicToolHandlers []toolhandler.ToolHandler
	if len(cfg.ToolProviderClientAddrs) > 0 {
		provider := utcp.NewToolProvider(toolprovider.WithAddrs(cfg.ToolProviderClientAddrs...))

		loaded, err := provider.Load(ctx, "", 100)
		if err != nil {
			log.Fatalf("❌ failed to load tools: %v", err)
		}
		dynamicToolHandlers = append(dynamicToolHandlers, loaded...)
	}

	// Create custom tooling
	calculate := calculate.NewToolHandler()

	researchModel := openaigenerator.NewGenerator(
		generator.WithApiKey(cfg.GeneratorKey),
		generator.WithModel(cfg.Generator),
		generator.WithPromptPrefix("Researcher response:"),
	)
	research := research.NewToolHandler(
		toolhandler.WithGenerator(researchModel),
	)

	allToolHandlers := []toolhandler.ToolHandler{
		calculate,
		research,
	}

	allToolHandlers = append(allToolHandlers, dynamicToolHandlers...)

	// Create ADK
	adk := agent.New(
		re,
		primaryModel,
		allToolHandlers,
		cfg.Context,
		cfg.SystemPrompt,
	)
	defer adk.Close()

	fmt.Println("--- Agent Development Kit Demo ---")

	// 1. Create space
	spaceId, err := adk.CreateSpace(ctx, "agent-learning-space", cfg.Space)
	if err != nil {
		log.Fatalf("❌ failed to create space: %v", err)
	}
	fmt.Printf("✅ Connected to Space: %s\n", spaceId)

	// 2. Start session
	sessionId, err := adk.CreateSession(ctx, cfg.Session, spaceId)
	if err != nil {
		log.Fatalf("❌ failed to start session: %v", err)
	}
	defer adk.FlushSession(ctx, sessionId)
	fmt.Printf("✅ Started Session: %s\n", sessionId)

	// 3. Simulate Conversation
	prompts := []string{
		"Summarize what I asked in our previous session.",
		"I want to design an AI agent with memory. What’s the first step?",
		`tool:calculate {"expression":"21 / 3"}`,
		`tool:research {"query":"Briefly explain pgvector and its benefits for retrieval."}`,
		`tool:localhost.echo {"message":"Hello from the Dynamic UTCP Server!"}`,
		`tool:localhost.timestamp {"format":"rfc3339"}`,
	}

	for _, prompt := range prompts {
		start := time.Now()
		reply, err := adk.Generate(ctx, sessionId, prompt)
		duration := time.Since(start)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}
		fmt.Printf("User: %s\nAgent: %s\n(%.2fs)\n\n", prompt, reply, duration.Seconds())
	}
	fmt.Println("✅ Populated conversation history.")

	// 4. TODO: Override Session Space

	select {
	case err := <-errCh:
		if err != nil {
			return
		}
	case <-sigChan:
		close(stop)
	}

	wg.Wait()
	close(errCh)

	fmt.Println("💾 All interactions flushed to long-term memory.")
}

func initHttpServer(ctx context.Context, addr string) (server.Server, error) {
	srv := httpserver.NewServer(
		server.WithAddress(addr),
		// add middleware
	)

	router, err := registerHttpHandlers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to register handlers: %w", err)
	}

	// otel

	if err := srv.Handle(router); err != nil {
		return nil, fmt.Errorf("failed to attach router: %w", err)
	}

	return srv, nil
}

func registerHttpHandlers(_ context.Context) (http.Handler, error) {
	router := mux.NewRouter()

	toolHandler := tool.NewHandler()
	router.Methods("POST").Path("/tools").HandlerFunc(toolHandler.Handle)

	return router, nil
}
