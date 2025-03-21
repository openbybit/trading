package pool

import (
	"context"

	"google.golang.org/grpc"
)

// DefaultOptions sets a list of recommended options for good performance.
// Feel free to modify these to suit your needs.
var DefaultOptions = Options{
	MaxIdle:              4,
	MaxActive:            32,
	MaxConcurrentStreams: 64,
	Reuse:                true,
}

// DialFunc is a function that dials a grpc connection.
type DialFunc func(ctx context.Context, address string, opts ...grpc.DialOption) (*grpc.ClientConn, error)

// Options are params for creating grpc connect pool.
type Options struct {
	// Dial is an application supplied function for creating and configuring a connection.
	DialOptions []grpc.DialOption

	// Maximum number of idle connections in the pool.
	MaxIdle int

	// Maximum number of connections allocated by the pool at a given time.
	// When zero, there is no limit on the number of connections in the pool.
	MaxActive int

	// MaxConcurrentStreams limit on the number of concurrent streams to each single connection
	MaxConcurrentStreams int

	// If Reuse is true and the pool is at the MaxActive limit, then Get() reuse
	// the connection to return, If Reuse is false and the pool is at the MaxActive limit,
	// create a one-time connection to return.
	Reuse bool
}

// Option is a function that sets some option.
type Option func(*Options)

func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(o *Options) {
		o.DialOptions = append(o.DialOptions, opts...)
	}
}

// WithMaxIdle set maxIdle
func WithMaxIdle(maxIdle int) Option {
	return func(o *Options) {
		o.MaxIdle = maxIdle
	}
}

// WithMaxActive set maxActive
func WithMaxActive(maxActive int) Option {
	return func(o *Options) {
		o.MaxActive = maxActive
	}
}

// WithMaxConcurrentStreams set maxConcurrentStreams
func WithMaxConcurrentStreams(maxConcurrentStreams int) Option {
	return func(o *Options) {
		o.MaxConcurrentStreams = maxConcurrentStreams
	}
}

// WithReuse set reuse
func WithReuse(reuse bool) Option {
	return func(o *Options) {
		o.Reuse = reuse
	}
}
