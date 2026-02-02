package calculate

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	toolhandler "github.com/w-h-a/agent/tool_handler"
)

type calculateToolHandler struct {
	options toolhandler.Options
}

func (th *calculateToolHandler) Spec() toolhandler.ToolSpec {
	return toolhandler.ToolSpec{
		Name:        "calculate",
		Description: "Evaluates simple math expressions such as '2 + 2' or '5 * 3'.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"expression": map[string]any{
					"type":        "string",
					"description": "Expression in the form '<number> <operator> <number>'.",
				},
			},
			"required": []string{"expression"},
		},
	}
}

func (th *calculateToolHandler) Invoke(_ context.Context, req toolhandler.ToolRequest) (toolhandler.ToolResponse, error) {
	raw, ok := req.Arguments["expression"]
	if !ok {
		return toolhandler.ToolResponse{}, fmt.Errorf("missing 'expression' argument")
	}
	if raw == nil {
		return toolhandler.ToolResponse{Content: ""}, nil
	}

	expression, ok := raw.(string)
	if !ok {
		return toolhandler.ToolResponse{}, fmt.Errorf("argument 'expression' has invalid type: expected string, got %T", raw)
	}

	fields := strings.Fields(strings.TrimSpace(expression))
	if len(fields) != 3 {
		return toolhandler.ToolResponse{}, fmt.Errorf("expected format '<number> <op> <number>'")
	}

	left, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return toolhandler.ToolResponse{}, fmt.Errorf("invalid left operand: %w", err)
	}
	right, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return toolhandler.ToolResponse{}, fmt.Errorf("invalid right operand: %w", err)
	}

	var result float64
	switch fields[1] {
	case "+":
		result = left + right
	case "-":
		result = left - right
	case "*", "x", "X":
		result = left * right
	case "/":
		if math.Abs(right) < 1e-12 {
			return toolhandler.ToolResponse{}, fmt.Errorf("division by zero")
		}
		result = left / right
	default:
		return toolhandler.ToolResponse{}, fmt.Errorf("unsupported operator %q", fields[1])
	}

	return toolhandler.ToolResponse{Content: strconv.FormatFloat(result, 'f', -1, 64)}, nil
}

func NewToolHandler(opts ...toolhandler.Option) toolhandler.ToolHandler {
	return &calculateToolHandler{
		options: toolhandler.NewOptions(opts...),
	}
}
