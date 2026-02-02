package utcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/universal-tool-calling-protocol/go-utcp"
	toolhandler "github.com/w-h-a/agent/tool_handler"
)

type utcpToolHandler struct {
	options  toolhandler.Options
	client   utcp.UtcpClientInterface
	toolName string
	spec     toolhandler.ToolSpec
}

func (th *utcpToolHandler) Spec() toolhandler.ToolSpec {
	return th.spec
}

func (th *utcpToolHandler) Invoke(ctx context.Context, req toolhandler.ToolRequest) (toolhandler.ToolResponse, error) {
	raw, err := th.client.CallTool(ctx, th.toolName, req.Arguments)
	if err != nil {
		return toolhandler.ToolResponse{}, err
	}

	var content string
	switch v := raw.(type) {
	case string:
		content = v
	default:
		if b, err := json.Marshal(v); err == nil {
			content = string(b)
		} else {
			content = fmt.Sprintf("%v", v)
		}
	}

	return toolhandler.ToolResponse{
		Content: content,
		Metadata: map[string]string{
			"source": "utcp",
			"tool":   th.toolName,
		},
	}, nil
}

func NewToolHandler(opts ...toolhandler.Option) toolhandler.ToolHandler {
	options := toolhandler.NewOptions(opts...)

	th := &utcpToolHandler{
		options: options,
	}

	if client, ok := UtcpClientFrom(options.Context); ok {
		th.client = client
	}

	if name, ok := ToolNameFrom(options.Context); ok {
		th.toolName = name
	}

	if spec, ok := ToolSpecFrom(options.Context); ok {
		th.spec = spec
	}

	return th
}
