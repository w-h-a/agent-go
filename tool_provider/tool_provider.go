package toolprovider

import "context"

type ToolProvider interface {
	Name() string
	Description() string
	Run(ctx context.Context, input string) (string, error)
}
