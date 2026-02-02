package toolprovider

import "context"

type Option func(*Options)

type Options struct {
	Addrs   []string
	Context context.Context
}

func WithAddrs(addrs ...string) Option {
	return func(o *Options) {
		o.Addrs = addrs
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
