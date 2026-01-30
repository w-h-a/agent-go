package calculate

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	toolprovider "github.com/w-h-a/agent/tool_provider"
)

type calculateToolProvider struct {
	options toolprovider.Options
}

func (tp *calculateToolProvider) Name() string { return "calculate" }

func (tp *calculateToolProvider) Description() string {
	return "Evaluates simple math expressions such as '2 + 2' or '5 * 3'."
}

func (tp *calculateToolProvider) Run(_ context.Context, input string) (string, error) {
	fields := strings.Fields(input)
	if len(fields) != 3 {
		return "", fmt.Errorf("expected format '<number> <op> <number>'")
	}

	left, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "", fmt.Errorf("invalid left operand: %w", err)
	}

	right, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return "", fmt.Errorf("invalid right operand: %w", err)
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
			return "", fmt.Errorf("division by zero")
		}
		result = left / right
	default:
		return "", fmt.Errorf("unsupported operator %q", fields[1])
	}

	return strconv.FormatFloat(result, 'f', -1, 64), nil
}

func NewToolProvider(opts ...toolprovider.Option) toolprovider.ToolProvider {
	return &calculateToolProvider{
		options: toolprovider.NewOptions(opts...),
	}
}
