package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/w-h-a/agent"
	"github.com/w-h-a/agent/cmd/quickstart/tool/echo"
	"github.com/w-h-a/agent/generator"
	openaigenerator "github.com/w-h-a/agent/generator/openai"
	memorymanager "github.com/w-h-a/agent/memory_manager"
	"github.com/w-h-a/agent/memory_manager/gomento"
	toolhandler "github.com/w-h-a/agent/tool_handler"
)

var (
	cfg struct {
		// Memory config
		MemoryLocation string `help:"Address of memory store for memory manager" default:"http://localhost:4000"`
		Window         int    `help:"Short-term memory window size per session" default:"8"`
		EmbedderKey    string `help:"API Key for the embedder" default:""`
		Embedder       string `help:"Model identifier for embedder" default:"text-embedding-3-small"`

		// Generator config
		GeneratorKey string `help:"API Key for the generator" default:""`
		Generator    string `help:"Model identifier for generator" default:"gpt-3.5-turbo"`

		// Tool Provider config
		ToolProviderClientAddrs []string `help:"List of addresses of servers with exposed tool handlers" default:"http://localhost:8080/tools"`

		// Agent config
		MaxTurns     int    `help:"Number of turns the agent is allowed to take per user prompt" default:"5"`
		Context      int    `help:"Number of conversation turns to send to the model" default:"6"`
		Hops         int    `help:"Number of hops to search for graphically related memories" default:"1"`
		SystemPrompt string `help:"System prompt for the agent" default:"You orchestrate a helpful assistant team."`

		// Space config
		SpaceName string `help:"Optional space name" default:"dark-mode"`
		SpaceId   string `help:"Optional space identifier" default:""`

		// Session config
		SessionId string `help:"Optional fixed session identifier" default:""`
	}
)

func main() {
	// Parse inputs
	_ = kong.Parse(&cfg)
	ctx := context.Background()

	// Create memory manager
	re := gomento.NewMemoryManager(
		memorymanager.WithLocation(cfg.MemoryLocation),
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
		cfg.MaxTurns,
		cfg.Context,
		cfg.Hops,
		cfg.SystemPrompt,
	)
	defer adk.Close()

	fmt.Println("ADK quickstart. Type a message and press enter.")

	spaceName := cfg.SpaceName
	spaceId := cfg.SpaceId
	if len(spaceId) == 0 && len(spaceName) > 0 {
		var err error
		spaceId, err = adk.CreateSpace(ctx, cfg.SpaceName)
		if err != nil {
			log.Fatalf("❌ failed to create space: %v", err)
		}
		fmt.Printf("✅ Connected to Space: %s\n", spaceId)
	}

	sessionId := cfg.SessionId
	if len(sessionId) == 0 {
		var err error
		sessionId, err = adk.CreateSession(ctx, spaceId)
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

		var files map[string]memorymanager.InputFile
		if strings.HasPrefix(input, "/file ") {
			parts := strings.SplitN(input, " ", 3)
			if len(parts) >= 2 {
				path := parts[1]
				f, err := os.Open(path)
				if err != nil {
					fmt.Printf("❌ Failed to open file: %v\n", err)
					continue
				}
				fileName := filepath.Base(path)
				files = map[string]memorymanager.InputFile{
					fileName: {Name: fileName, Reader: f},
				}
				if len(parts) == 3 {
					input = strings.TrimSpace(parts[2])
				} else {
					input = fmt.Sprintf("Uploaded file: %s", fileName)
				}
				fmt.Printf("📎 Attaching %s...\n", fileName)
			}
		}

		rsp, err := adk.Generate(ctx, sessionId, input, files)
		if err != nil {
			fmt.Println("Error generating response:", err)
			continue
		}
		fmt.Printf("%s\n", rsp)
		fmt.Println("---")

		for _, file := range files {
			if closer, ok := file.Reader.(interface{ Close() error }); ok {
				closer.Close()
			}
		}
	}
}
