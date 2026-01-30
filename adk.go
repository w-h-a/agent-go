package agent

import (
	"context"

	"github.com/w-h-a/agent/generator"
	"github.com/w-h-a/agent/generator/anthropic"
	"github.com/w-h-a/agent/generator/google"
	"github.com/w-h-a/agent/generator/openai"
	"github.com/w-h-a/agent/retriever"
	"github.com/w-h-a/agent/retriever/gomento"
	"github.com/w-h-a/agent/retriever/postgres"
	toolprovider "github.com/w-h-a/agent/tool_provider"
	"github.com/w-h-a/agent/tool_provider/calculate"
	"github.com/w-h-a/agent/tool_provider/echo"
	"github.com/w-h-a/agent/tool_provider/research"
	"github.com/w-h-a/agent/tool_provider/time"
)

func InitAgent(
	ctx context.Context,
	retrieverType string,
	retrieverLocation string,
	generatorType string,
	apiKey string,
	model string,
	embedder string,
	systemPrompt string,
	contextLimit int,
) *Agent {
	retriever := initRetriever(retrieverType, retrieverLocation, apiKey, embedder)

	primary := initGenerator(generatorType, apiKey, model, "Coordinator response:")

	researcher := initGenerator(generatorType, apiKey, model, "Research summary:")

	tps := map[string]toolprovider.ToolProvider{
		"echo":      echo.NewToolProvider(),
		"calculate": calculate.NewToolProvider(),
		"time":      time.NewToolProvider(),
		"research":  research.NewToolProvider(toolprovider.WithGenerator(researcher)),
	}

	return New(
		retriever,
		primary,
		tps,
		systemPrompt,
		contextLimit,
	)
}

func initRetriever(
	choice string,
	location string,
	apiKey string,
	model string,
) retriever.Retriever {
	switch choice {
	case "gomento":
		return gomento.NewRetriever(
			retriever.WithLocation(location),
		)
	case "postgres":
		return postgres.NewRetriever(
			retriever.WithLocation(location),
			retriever.WithApiKey(apiKey),
			retriever.WithModel(model),
			retriever.WithShortTermMemorySize(16),
		)
	default:
		panic("unknown retriever choice")
	}
}

func initGenerator(
	choice string,
	apiKey string,
	model string,
	promptPrefix string,
) generator.Generator {
	opts := []generator.Option{
		generator.WithApiKey(apiKey),
		generator.WithModel(model),
		generator.WithPromptPrefix(promptPrefix),
	}

	switch choice {
	case "openai":
		return openai.NewGenerator(opts...)
	case "google":
		return google.NewGenerator(opts...)
	case "anthropic":
		return anthropic.NewGenerator(opts...)
	default:
		panic("unknown model choice")
	}
}
