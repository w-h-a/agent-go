package toolhandler

import "context"

type ToolHandler interface {
	Spec() ToolSpec
	Invoke(ctx context.Context, req ToolRequest) (ToolResponse, error)
}
