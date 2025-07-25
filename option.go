package rtvbp

import (
	"github.com/babelforce/rtvbp-go/proto"
	"log/slog"
	"time"
)

type sessionOptions struct {
	id              string
	logger          *slog.Logger
	transport       TransportFunc
	handler         SessionHandler
	audioBufferSize int
	requestTimeout  time.Duration
}

type Option func(opts *sessionOptions)

func withDefaults() Option {
	return withOptions(
		WithLogger(slog.Default()),
		WithRequestTimeout(5*time.Second),
		WithAudioBufferSize(1024*1024),
		WithID(proto.ID()),
	)
}

func withOptions(os ...Option) Option {
	return func(opts *sessionOptions) {
		for _, o := range os {
			o(opts)
		}
	}
}

func WithRequestTimeout(timeout time.Duration) Option {
	return func(opts *sessionOptions) {
		opts.requestTimeout = timeout
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(opts *sessionOptions) {
		opts.logger = logger
	}
}

func WithID(id string) Option {
	return func(opts *sessionOptions) {
		opts.id = id
	}
}

func WithTransport(f TransportFunc) Option {
	return func(opts *sessionOptions) {
		opts.transport = f
	}
}

func WithHandler(h SessionHandler) Option {
	return func(opts *sessionOptions) {
		opts.handler = h
	}
}

func WithAudioBufferSize(size int) Option {
	return func(opts *sessionOptions) {
		opts.audioBufferSize = size
	}
}
