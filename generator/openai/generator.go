package openai

import (
	"context"
	"errors"

	"github.com/sashabaranov/go-openai"
	"github.com/w-h-a/agent/generator"
)

type openAIGenerator struct {
	options generator.Options
	client  *openai.Client
}

func (g *openAIGenerator) Generate(ctx context.Context, prompt string) (string, error) {
	fullPrompt := prompt
	if len(g.options.PromptPrefix) > 0 {
		fullPrompt = g.options.PromptPrefix + "\n" + prompt
	}

	req := openai.ChatCompletionRequest{
		Model: g.options.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: fullPrompt,
			},
		},
	}

	rsp, err := g.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(rsp.Choices) == 0 || len(rsp.Choices[0].Message.Content) == 0 {
		return "", errors.New("no response from OpenAI")
	}

	return rsp.Choices[0].Message.Content, nil
}

func NewGenerator(opts ...generator.Option) generator.Generator {
	options := generator.NewOptions(opts...)

	g := &openAIGenerator{
		options: options,
	}

	client := openai.NewClient(options.ApiKey)

	g.client = client

	return g
}
