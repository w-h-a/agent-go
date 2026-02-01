package toolprovider

import "context"

type ToolProvider interface {
	Spec() ToolSpec
	Invoke(ctx context.Context, req ToolRequest) (ToolResponse, error)
}
