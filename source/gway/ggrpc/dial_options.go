package ggrpc

import "code.bydev.io/fbu/gateway/gway.git/ggrpc/pool"

type dialOptions struct {
	poolEnable     bool
	poolOptions    []pool.Option
	disableDefault bool
}

type DialOption func(o *dialOptions)

func WithPool(opts ...pool.Option) DialOption {
	return func(o *dialOptions) {
		o.poolEnable = true
		o.poolOptions = append(o.poolOptions, opts...)
	}
}

func WithDisableDefault() DialOption {
	return func(o *dialOptions) {
		o.disableDefault = true
	}
}
