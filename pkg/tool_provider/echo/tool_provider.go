package echo

import (
	"context"
	"strings"

	toolprovider "github.com/w-h-a/agent/pkg/tool_provider"
)

type echoToolProvider struct {
	options toolprovider.Options
}

func (tp *echoToolProvider) Name() string { return "echo" }

func (tp *echoToolProvider) Description() string {
	return "Echoes the provided text back to the caller."
}

func (tp *echoToolProvider) Run(_ context.Context, input string) (string, error) {
	return strings.TrimSpace(input), nil
}

func NewToolProvider(opts ...toolprovider.Option) toolprovider.ToolProvider {
	return &echoToolProvider{
		options: toolprovider.NewOptions(opts...),
	}
}
