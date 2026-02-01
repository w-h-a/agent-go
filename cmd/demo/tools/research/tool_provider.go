package research

import (
	"context"
	"fmt"
	"strings"

	toolprovider "github.com/w-h-a/agent/tool_provider"
)

type researchToolProvider struct {
	options toolprovider.Options
	persona string
}

func (tp *researchToolProvider) Spec() toolprovider.ToolSpec {
	return toolprovider.ToolSpec{
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

func (tp *researchToolProvider) Invoke(ctx context.Context, req toolprovider.ToolRequest) (toolprovider.ToolResponse, error) {
	raw, ok := req.Arguments["query"]
	if !ok {
		return toolprovider.ToolResponse{}, fmt.Errorf("missing 'query' argument")
	}
	if raw == nil {
		return toolprovider.ToolResponse{Content: ""}, nil
	}

	query, ok := raw.(string)
	if !ok {
		return toolprovider.ToolResponse{}, fmt.Errorf("argument 'query' has invalid type: expected string, got %T", raw)
	}

	query = strings.TrimSpace(query)

	var prompt strings.Builder

	prompt.WriteString(tp.persona)
	prompt.WriteString("\n\nTask\n")
	prompt.WriteString(query)
	prompt.WriteString("\n\nDeliverable: Provide a concise research brief with bullet points and next steps.\n")

	rsp, err := tp.options.Generator.Generate(ctx, prompt.String())
	if err != nil {
		return toolprovider.ToolResponse{}, err
	}

	return toolprovider.ToolResponse{Content: rsp}, nil
}

func NewToolProvider(opts ...toolprovider.Option) toolprovider.ToolProvider {
	options := toolprovider.NewOptions(opts...)

	if options.Generator == nil {
		panic("generator is required")
	}

	tp := &researchToolProvider{
		options: options,
		persona: "You are a diligent research assistant. Provide structured findings and cite sources when available.",
	}

	return tp
}
