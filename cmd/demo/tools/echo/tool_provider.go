package echo

import (
	"context"
	"fmt"
	"strings"

	toolprovider "github.com/w-h-a/agent/tool_provider"
)

type echoToolProvider struct {
	options toolprovider.Options
}

func (tp *echoToolProvider) Spec() toolprovider.ToolSpec {
	return toolprovider.ToolSpec{
		Name:        "echo",
		Description: "Echoes the provided text back to the caller.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"input": map[string]any{
					"type":        "string",
					"description": "Text to echo back.",
				},
			},
			"required": []any{"input"},
		},
	}
}

func (tp *echoToolProvider) Invoke(_ context.Context, req toolprovider.ToolRequest) (toolprovider.ToolResponse, error) {
	raw, ok := req.Arguments["input"]
	if !ok {
		return toolprovider.ToolResponse{}, fmt.Errorf("missing 'input' argument")
	}
	if raw == nil {
		return toolprovider.ToolResponse{Content: ""}, nil
	}

	input, ok := raw.(string)
	if !ok {
		return toolprovider.ToolResponse{}, fmt.Errorf("argument 'input' has invalid type: expected string, got %T", raw)
	}

	return toolprovider.ToolResponse{Content: strings.TrimSpace(input)}, nil
}

func NewToolProvider(opts ...toolprovider.Option) toolprovider.ToolProvider {
	return &echoToolProvider{
		options: toolprovider.NewOptions(opts...),
	}
}
