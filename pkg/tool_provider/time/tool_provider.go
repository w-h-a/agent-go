package time

import (
	"context"
	"time"

	toolprovider "github.com/w-h-a/agent/pkg/tool_provider"
)

type timeToolProvider struct {
	options toolprovider.Options
}

func (tp *timeToolProvider) Name() string { return "time" }

func (tp *timeToolProvider) Description() string { return "Returns the current UTC time." }

func (tp *timeToolProvider) Run(_ context.Context, _ string) (string, error) {
	return time.Now().UTC().Format(time.RFC3339), nil
}

func NewToolProvider(opts ...toolprovider.Option) toolprovider.ToolProvider {
	return &timeToolProvider{
		options: toolprovider.NewOptions(opts...),
	}
}
