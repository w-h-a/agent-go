package openai

import (
	"context"
	"errors"

	"github.com/sashabaranov/go-openai"
	"github.com/w-h-a/agent/embedder"
)

type openAIEmbedder struct {
	options embedder.Options
	client  *openai.Client
}

func (e *openAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	rsp, err := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(e.options.Model),
	})
	if err != nil {
		return nil, err
	}

	if len(rsp.Data) == 0 || len(rsp.Data[0].Embedding) == 0 {
		return nil, errors.New("no response from OpenAI")
	}

	return rsp.Data[0].Embedding, nil
}

func NewEmbedder(opts ...embedder.Option) embedder.Embedder {
	options := embedder.NewOptions(opts...)

	e := &openAIEmbedder{
		options: options,
	}

	client := openai.NewClient(options.ApiKey)

	e.client = client

	return e
}
