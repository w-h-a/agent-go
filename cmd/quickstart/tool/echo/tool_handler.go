package echo

import (
	"context"
	"fmt"
	"strings"

	toolhandler "github.com/w-h-a/agent/tool_handler"
)

type echoToolHandler struct {
	options toolhandler.Options
}

func (th *echoToolHandler) Spec() toolhandler.ToolSpec {
	return toolhandler.ToolSpec{
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

func (th *echoToolHandler) Invoke(_ context.Context, req toolhandler.ToolRequest) (toolhandler.ToolResponse, error) {
	raw, ok := req.Arguments["input"]
	if !ok {
		return toolhandler.ToolResponse{}, fmt.Errorf("missing 'input' argument")
	}
	if raw == nil {
		return toolhandler.ToolResponse{Content: ""}, nil
	}

	input, ok := raw.(string)
	if !ok {
		return toolhandler.ToolResponse{}, fmt.Errorf("argument 'input' has invalid type: expected string, got %T", raw)
	}

	return toolhandler.ToolResponse{Content: strings.TrimSpace(input)}, nil
}

func NewToolHandler(opts ...toolhandler.Option) toolhandler.ToolHandler {
	return &echoToolHandler{
		options: toolhandler.NewOptions(opts...),
	}
}
