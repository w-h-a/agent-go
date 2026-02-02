package research

import (
	"context"
	"fmt"
	"strings"

	toolhandler "github.com/w-h-a/agent/tool_handler"
)

type researchToolHandler struct {
	options toolhandler.Options
	persona string
}

func (th *researchToolHandler) Spec() toolhandler.ToolSpec {
	return toolhandler.ToolSpec{
		Name:        "research",
		Description: "Synthesizes background information and drafts research summaries.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "The query prompt for research.",
				},
			},
			"required": []any{"query"},
		},
	}
}

func (th *researchToolHandler) Invoke(ctx context.Context, req toolhandler.ToolRequest) (toolhandler.ToolResponse, error) {
	raw, ok := req.Arguments["query"]
	if !ok {
		return toolhandler.ToolResponse{}, fmt.Errorf("missing 'query' argument")
	}
	if raw == nil {
		return toolhandler.ToolResponse{Content: ""}, nil
	}

	query, ok := raw.(string)
	if !ok {
		return toolhandler.ToolResponse{}, fmt.Errorf("argument 'query' has invalid type: expected string, got %T", raw)
	}

	query = strings.TrimSpace(query)

	var prompt strings.Builder

	prompt.WriteString(th.persona)
	prompt.WriteString("\n\nTask\n")
	prompt.WriteString(query)
	prompt.WriteString("\n\nDeliverable: Provide a concise research brief with bullet points and next steps.\n")

	rsp, err := th.options.Generator.Generate(ctx, prompt.String())
	if err != nil {
		return toolhandler.ToolResponse{}, err
	}

	return toolhandler.ToolResponse{Content: rsp}, nil
}

func NewToolHandler(opts ...toolhandler.Option) toolhandler.ToolHandler {
	options := toolhandler.NewOptions(opts...)

	if options.Generator == nil {
		panic("generator is required")
	}

	th := &researchToolHandler{
		options: options,
		persona: "You are a diligent research assistant. Provide structured findings and cite sources when available.",
	}

	return th
}
