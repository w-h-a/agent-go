package research

import (
	"context"
	"strings"

	toolprovider "github.com/w-h-a/agent/tool_provider"
)

type researchToolProvider struct {
	options toolprovider.Options
	persona string
}

func (tp *researchToolProvider) Name() string { return "research" }

func (tp *researchToolProvider) Description() string {
	return "Synthesizes background information and drafts research summaries."
}

func (tp *researchToolProvider) Run(ctx context.Context, input string) (string, error) {
	var prompt strings.Builder

	prompt.WriteString(tp.persona)
	prompt.WriteString("\n\nTask\n")
	prompt.WriteString(strings.TrimSpace(input))
	prompt.WriteString("\n\nDeliverable: Provide a concise research brief with bullet points and next steps.\n")

	rsp, err := tp.options.Generator.Generate(ctx, prompt.String())
	if err != nil {
		return "", err
	}

	return rsp, nil
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
