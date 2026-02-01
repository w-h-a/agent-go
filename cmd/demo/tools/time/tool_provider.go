package time

import (
	"context"
	"time"

	toolprovider "github.com/w-h-a/agent/tool_provider"
)

type timeToolProvider struct {
	options toolprovider.Options
}

func (tp *timeToolProvider) Spec() toolprovider.ToolSpec {
	return toolprovider.ToolSpec{
		Name:        "time",
		Description: "Returns the current UTC time.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

func (tp *timeToolProvider) Invoke(_ context.Context, _ toolprovider.ToolRequest) (toolprovider.ToolResponse, error) {
	return toolprovider.ToolResponse{Content: time.Now().UTC().Format(time.RFC3339)}, nil
}

func NewToolProvider(opts ...toolprovider.Option) toolprovider.ToolProvider {
	return &timeToolProvider{
		options: toolprovider.NewOptions(opts...),
	}
}
