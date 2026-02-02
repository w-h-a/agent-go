package time

import (
	"context"
	"time"

	toolhandler "github.com/w-h-a/agent/tool_handler"
)

type timeToolHandler struct {
	options toolhandler.Options
}

func (th *timeToolHandler) Spec() toolhandler.ToolSpec {
	return toolhandler.ToolSpec{
		Name:        "time",
		Description: "Returns the current UTC time.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

func (th *timeToolHandler) Invoke(_ context.Context, _ toolhandler.ToolRequest) (toolhandler.ToolResponse, error) {
	return toolhandler.ToolResponse{Content: time.Now().UTC().Format(time.RFC3339)}, nil
}

func NewToolHandler(opts ...toolhandler.Option) toolhandler.ToolHandler {
	return &timeToolHandler{
		options: toolhandler.NewOptions(opts...),
	}
}
