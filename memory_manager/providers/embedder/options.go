package embedder

import "context"

type Option func(*Options)

type Options struct {
	ApiKey  string
	Model   string
	Context context.Context
}

func WithApiKey(apiKey string) Option {
	return func(o *Options) {
		o.ApiKey = apiKey
	}
}

func WithModel(model string) Option {
	return func(o *Options) {
		o.Model = model
	}
}

func NewOptions(opts ...Option) Options {
	options := Options{
		Context: context.Background(),
	}
	for _, opt := range opts {
		opt(&options)
	}
	return options
}
