package toolprovider

import (
	"context"

	"github.com/w-h-a/agent/generator"
)

type Option func(*Options)

type Options struct {
	Generator generator.Generator
	Context   context.Context
}

func WithGenerator(gen generator.Generator) Option {
	return func(opts *Options) {
		opts.Generator = gen
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
