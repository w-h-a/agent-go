package toolprovider

import "context"

type Option func(*Options)

type Options struct {
	Context context.Context
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
