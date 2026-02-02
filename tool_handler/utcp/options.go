package utcp

import (
	"context"

	"github.com/universal-tool-calling-protocol/go-utcp"
	toolhandler "github.com/w-h-a/agent/tool_handler"
)

type utcpClientKey struct{}

func WithUtcpClient(client utcp.UtcpClientInterface) toolhandler.Option {
	return func(o *toolhandler.Options) {
		o.Context = context.WithValue(o.Context, utcpClientKey{}, client)
	}
}

func UtcpClientFrom(ctx context.Context) (utcp.UtcpClientInterface, bool) {
	client, ok := ctx.Value(utcpClientKey{}).(utcp.UtcpClientInterface)
	return client, ok
}

type nameKey struct{}

func WithToolName(name string) toolhandler.Option {
	return func(o *toolhandler.Options) {
		o.Context = context.WithValue(o.Context, nameKey{}, name)
	}
}

func ToolNameFrom(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(nameKey{}).(string)
	return name, ok
}

type specKey struct{}

func WithToolSpec(spec toolhandler.ToolSpec) toolhandler.Option {
	return func(o *toolhandler.Options) {
		o.Context = context.WithValue(o.Context, specKey{}, spec)
	}
}

func ToolSpecFrom(ctx context.Context) (toolhandler.ToolSpec, bool) {
	spec, ok := ctx.Value(specKey{}).(toolhandler.ToolSpec)
	return spec, ok
}
