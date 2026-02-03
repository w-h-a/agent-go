package memorymanager

import (
	"context"
)

type Option func(*Options)

type Options struct {
	Location string
	Context  context.Context
}

func WithLocation(loc string) Option {
	return func(o *Options) {
		o.Location = loc
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

type CreateSessionOption func(*CreateSessionOptions)

type CreateSessionOptions struct {
	SpaceId string
	Context context.Context
}

func WithSpaceId(spaceId string) CreateSessionOption {
	return func(o *CreateSessionOptions) {
		o.SpaceId = spaceId
	}
}

func NewSessionOptions(opts ...CreateSessionOption) CreateSessionOptions {
	options := CreateSessionOptions{
		Context: context.Background(),
	}
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

type AddToShortTermOption func(*AddToShortTermOptions)

type AddToShortTermOptions struct {
	Files   map[string]File
	Context context.Context
}

func WithFiles(files map[string]File) AddToShortTermOption {
	return func(o *AddToShortTermOptions) {
		o.Files = files
	}
}

func NewAddToShortTermOptions(opts ...AddToShortTermOption) AddToShortTermOptions {
	options := AddToShortTermOptions{
		Context: context.Background(),
	}
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

type ListShortTermOption func(*ListShortTermOptions)

type ListShortTermOptions struct {
	Limit   int
	Context context.Context
}

func WithShortTermLimit(limit int) ListShortTermOption {
	return func(o *ListShortTermOptions) {
		o.Limit = limit
	}
}

func NewListShortTermOptions(opts ...ListShortTermOption) ListShortTermOptions {
	options := ListShortTermOptions{
		Context: context.Background(),
	}
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

type SearchLongTermOption func(*SearchLongTermOptions)

type SearchLongTermOptions struct {
	Limit   int
	SpaceId string
	Context context.Context
}

func WithSearchLongTermLimit(limit int) SearchLongTermOption {
	return func(o *SearchLongTermOptions) {
		o.Limit = limit
	}
}

func WithSearchLongTermSpaceId(spaceId string) SearchLongTermOption {
	return func(o *SearchLongTermOptions) {
		o.SpaceId = spaceId
	}
}

func NewSearchOptions(opts ...SearchLongTermOption) SearchLongTermOptions {
	options := SearchLongTermOptions{
		Context: context.Background(),
	}
	for _, opt := range opts {
		opt(&options)
	}
	return options
}
