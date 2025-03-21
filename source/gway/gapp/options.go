package gapp

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type BlockFunc func(ctx context.Context)

type Options struct {
	ctx         context.Context //
	block       BlockFunc       //
	hooks       []Lifecycle     // 生命周期管理hooks
	endpoints   []Endpoint      // http注册endpoints
	addr        string          // http地址,若为空则不启动,通常为6480端口
	exitTimeout time.Duration   // 超时强制退出时间,默认10s
	logger      Logger          // 日志
}

type Option func(o *Options)

// WithContext set global context
func WithContext(ctx context.Context) Option {
	return func(o *Options) {
		o.ctx = ctx
	}
}

// WithAddr set http server address, empty will disable to start http server
func WithAddr(addr string) Option {
	return func(o *Options) {
		o.addr = addr
	}
}

// WithLifecycles add lifecycle hooks
func WithLifecycles(hooks ...Lifecycle) Option {
	return func(o *Options) {
		o.hooks = append(o.hooks, hooks...)
	}
}

// WithEndpoints add endpoints
func WithEndpoints(endpoints ...Endpoint) Option {
	return func(o *Options) {
		o.endpoints = append(o.endpoints, endpoints...)
	}
}

// WithDefaultEndpoints add default endpoints such as prometheus, pprof
func WithDefaultEndpoints() Option {
	return func(o *Options) {
		o.endpoints = append(o.endpoints,
			newPrometheusEndpoint(),
			newPprofEndpoint(),
			newAdminEndpoint(""),
		)
	}
}

// WithPrometheus add prometheus endpoint
func WithPrometheus() Option {
	return func(o *Options) {
		o.endpoints = append(o.endpoints, newPrometheusEndpoint())
	}
}

// WithPprof add pprof endpoint
func WithPprof() Option {
	return func(o *Options) {
		o.endpoints = append(o.endpoints, newPprofEndpoint())
	}
}

// WithHealth add health check endpoint
func WithHealth(cb HealthFunc) Option {
	return func(o *Options) {
		o.endpoints = append(o.endpoints, newHealthEndpoint(cb))
	}
}

// WithAdmin add default admin endpoint
func WithAdmin(route string) Option {
	return func(o *Options) {
		o.endpoints = append(o.endpoints, newAdminEndpoint(route))
	}
}

// WithExitTimeout set exit timeout
func WithExitTimeout(d time.Duration) Option {
	return func(o *Options) {
		if d > 0 {
			o.exitTimeout = d
		}
	}
}

// WithLogger set logger
func WithLogger(l Logger) Option {
	return func(o *Options) {
		o.logger = l
	}
}

// WithBlock set block, default blockOnSignal
func WithBlock(w BlockFunc) Option {
	return func(o *Options) {
		o.block = w
	}
}

// blockOnSignal blocks until context done or posix signal (SIGINT / SIGTERM) received.
// This is useful as a standard blocker function for daemon application.
func blockOnSignal(ctx context.Context) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
	case <-ctx.Done():
	}
}
