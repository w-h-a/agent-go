package storer

import "context"

type Option func(*Options)

type Options struct {
	Location    string
	ApiKey      string
	Collection  string
	VectorIndex string
	VectorSize  uint64
	Distance    string
	Context     context.Context
}

func WithLocation(loc string) Option {
	return func(o *Options) {
		o.Location = loc
	}
}

func WithApiKey(key string) Option {
	return func(o *Options) {
		o.ApiKey = key
	}
}

func WithCollection(coll string) Option {
	return func(o *Options) {
		o.Collection = coll
	}
}

func WithVectorIndex(index string) Option {
	return func(o *Options) {
		o.VectorIndex = index
	}
}

func WithVectorSize(size uint64) Option {
	return func(o *Options) {
		o.VectorSize = size
	}
}

func WithDistance(dist string) Option {
	return func(o *Options) {
		o.Distance = dist
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
