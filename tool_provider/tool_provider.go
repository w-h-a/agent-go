package toolprovider

import (
	"context"

	toolhandler "github.com/w-h-a/agent/tool_handler"
)

type ToolProvider interface {
	Load(ctx context.Context, query string, limit int) ([]toolhandler.ToolHandler, error)
}
