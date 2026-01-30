package anthropic

import (
	"context"
	"errors"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	anthropicopt "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/w-h-a/agent/generator"
)

type anthropicGenerator struct {
	options generator.Options
	client  *anthropic.Client
}

func (g *anthropicGenerator) Generate(ctx context.Context, prompt string) (string, error) {
	fullPrompt := prompt
	if len(g.options.PromptPrefix) > 0 {
		fullPrompt = g.options.PromptPrefix + "\n" + prompt
	}

	req := anthropic.MessageNewParams{
		Model:     anthropic.Model(g.options.Model),
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(fullPrompt)),
		},
	}

	rsp, err := g.client.Messages.New(ctx, req)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	for _, content := range rsp.Content {
		if text, ok := content.AsAny().(anthropic.TextBlock); ok {
			b.WriteString(text.Text)
		}
	}

	result := b.String()
	if len(result) == 0 {
		return "", errors.New("no response from Anthropic")
	}

	return result, nil
}

func NewGenerator(opts ...generator.Option) generator.Generator {
	options := generator.NewOptions(opts...)

	g := &anthropicGenerator{
		options: options,
	}

	client := anthropic.NewClient(
		anthropicopt.WithAPIKey(options.ApiKey),
	)

	g.client = &client

	return g
}
