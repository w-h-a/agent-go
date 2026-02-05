package memorymanager

import (
	"context"
	"time"

	"github.com/w-h-a/agent/memory_manager/providers/embedder"
	"github.com/w-h-a/agent/memory_manager/providers/storer"
)

type Option func(*Options)

type Options struct {
	Location          string
	Storer            storer.Storer
	Embedder          embedder.Embedder
	SessionWindowSize int
	Weights           Weights
	Thresholds        Thresholds
	Context           context.Context
}

type Weights struct {
	Similarity float64
	Recency    float64
}

type Thresholds struct {
	Relevance           float64
	HalfLife            time.Duration
	RejectionSimilarity float64
}

func WithLocation(loc string) Option {
	return func(o *Options) {
		o.Location = loc
	}
}

func WithStorer(storer storer.Storer) Option {
	return func(o *Options) {
		o.Storer = storer
	}
}

func WithEmbedder(embedder embedder.Embedder) Option {
	return func(o *Options) {
		o.Embedder = embedder
	}
}

func WithWeights(weights Weights) Option {
	return func(o *Options) {
		o.Weights = weights
	}
}

func WithThresholds(thresholds Thresholds) Option {
	return func(o *Options) {
		o.Thresholds = thresholds
	}
}

func NewOptions(opts ...Option) Options {
	options := Options{
		SessionWindowSize: 20,
		Weights: Weights{
			Similarity: 1.0, // exact semantic match
			Recency:    0.5, // medium bias for recency
		},
		Thresholds: Thresholds{
			Relevance:           0.7,            // mild diversity
			HalfLife:            72 * time.Hour, // 3 days
			RejectionSimilarity: 0.97,           // strong bias against duplicates
		},
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

func NewCreateSessionOptions(opts ...CreateSessionOption) CreateSessionOptions {
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
		Limit:   5,
		Context: context.Background(),
	}
	for _, opt := range opts {
		opt(&options)
	}
	return options
}
