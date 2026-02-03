package google

import (
	"context"
	"errors"

	"github.com/google/generative-ai-go/genai"
	"github.com/w-h-a/agent/memory_manager/providers/embedder"
	genaiopt "google.golang.org/api/option"
)

type googleEmbedder struct {
	options embedder.Options
	client  *genai.Client
}

func (e *googleEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	model := e.client.EmbeddingModel(e.options.Model)
	rsp, err := model.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, err
	}

	if rsp == nil || rsp.Embedding == nil || len(rsp.Embedding.Values) == 0 {
		return nil, errors.New("no response from Google")
	}

	return rsp.Embedding.Values, nil
}

func NewEmbedder(opts ...embedder.Option) embedder.Embedder {
	options := embedder.NewOptions(opts...)

	e := &googleEmbedder{
		options: options,
	}

	client, err := genai.NewClient(
		context.Background(),
		genaiopt.WithAPIKey(options.ApiKey),
	)
	if err != nil {
		panic(err)
	}

	e.client = client

	return e
}
