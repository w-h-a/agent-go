package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/w-h-a/agent"
	"github.com/w-h-a/agent/cmd/quickstart/tool/echo"
	"github.com/w-h-a/agent/generator"
	openaigenerator "github.com/w-h-a/agent/generator/openai"
	memorymanager "github.com/w-h-a/agent/memory_manager"
	"github.com/w-h-a/agent/memory_manager/munin"
	"github.com/w-h-a/agent/memory_manager/providers/embedder"
	openaiembedder "github.com/w-h-a/agent/memory_manager/providers/embedder/openai"
	"github.com/w-h-a/agent/memory_manager/providers/storer"
	"github.com/w-h-a/agent/memory_manager/providers/storer/qdrant"
	toolhandler "github.com/w-h-a/agent/tool_handler"
)

var (
	cfg struct {
		// Memory config
		MemoryLocation string `help:"Address of memory store for memory manager" default:"http://localhost:6333"`
		Window         int    `help:"Short-term memory window size per session" default:"8"`
		EmbedderKey    string `help:"API Key for the embedder" default:""`
		Embedder       string `help:"Model identifier for embedder" default:"text-embedding-3-small"`

		// Generator config
		GeneratorKey string `help:"API Key for the generator" default:""`
		Generator    string `help:"Model identifier for generator" default:"gpt-3.5-turbo"`

		// Tool Provider config
		ToolProviderClientAddrs []string `help:"List of addresses of servers with exposed tool handlers" default:"http://localhost:8080/tools"`

		// Agent config
		Context      int    `help:"Number of conversation turns to send to the model" default:"6"`
		Hops         int    `help:"Number of hops to search for graphically related memories" default:"1"`
		SystemPrompt string `help:"System prompt for the agent" default:"You orchestrate a helpful assistant team."`

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

	// Create memory manager
	re := munin.NewMemoryManager(
		memorymanager.WithStorer(
			qdrant.NewStorer(
				storer.WithLocation(cfg.MemoryLocation),
				storer.WithCollection("agent_memory"),
				storer.WithVectorSize(1536),
			),
		),
		memorymanager.WithEmbedder(
			openaiembedder.NewEmbedder(
				embedder.WithApiKey(cfg.EmbedderKey),
				embedder.WithModel(cfg.Embedder),
			),
		),
	)

	// Create primary agent's model
	primaryModel := openaigenerator.NewGenerator(
		generator.WithApiKey(cfg.GeneratorKey),
		generator.WithModel(cfg.Generator),
	)

	// Create custom tooling
	echo := echo.NewToolHandler()

	allToolHandlers := []toolhandler.ToolHandler{
		echo,
	}

	// Create ADK
	adk := agent.New(
		re,
		primaryModel,
		allToolHandlers,
		cfg.Context,
		cfg.Hops,
		cfg.SystemPrompt,
	)
	defer adk.Close()

	fmt.Println("ADK quickstart. Type a message and press enter.")

	sessionId := cfg.Session
	if len(sessionId) == 0 {
		var err error
		sessionId, err = adk.CreateSession(ctx, cfg.Space)
		if err != nil {
			log.Fatalf("❌ failed to start session: %v", err)
		}
		defer adk.FlushSession(ctx, sessionId)
		fmt.Printf("✅ Started Session: %s\n", sessionId)
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}
		input = strings.TrimSpace(input)
		if len(input) == 0 {
			fmt.Println("Goodbye!")
			return
		}

		rsp, err := adk.Generate(ctx, sessionId, input)
		if err != nil {
			fmt.Println("Error generating response:", err)
			continue
		}
		fmt.Printf("%s\n", rsp)
		fmt.Println("---")
	}
}
