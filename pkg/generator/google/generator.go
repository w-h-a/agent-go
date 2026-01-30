package google

import (
	"context"
	"errors"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/w-h-a/agent/pkg/generator"
	genaiopt "google.golang.org/api/option"
)

type googleGenerator struct {
	options generator.Options
	client  *genai.Client
}

func (g *googleGenerator) Generate(ctx context.Context, prompt string) (string, error) {
	fullPrompt := prompt
	if len(g.options.PromptPrefix) > 0 {
		fullPrompt = g.options.PromptPrefix + "\n" + prompt
	}

	req := genai.Text(fullPrompt)

	model := g.client.GenerativeModel(g.options.Model)
	rsp, err := model.GenerateContent(ctx, req)
	if err != nil {
		return "", err
	}

	if len(rsp.Candidates) == 0 || rsp.Candidates[0].Content == nil || len(rsp.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("no response from Google")
	}

	var b strings.Builder
	for _, part := range rsp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			b.WriteString(string(text))
		}
	}

	return b.String(), nil
}

func NewGenerator(opts ...generator.Option) generator.Generator {
	options := generator.NewOptions(opts...)

	g := &googleGenerator{
		options: options,
	}

	client, err := genai.NewClient(
		context.Background(),
		genaiopt.WithAPIKey(options.ApiKey),
	)
	if err != nil {
		panic(err)
	}

	g.client = client

	return g
}
