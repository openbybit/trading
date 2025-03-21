package ghttp

const (
	defaultReadBufferSize      = 30 << 10 // read buffer size
	defaultMaxIdleConnsPerHost = 128      // max idle conns per host
	defaultMaxIdleConns        = 1024     // all idle conn
	defaultMaxConnsPerHost     = 10000    // max conns per host
)

type options struct {
	readBufferSize      int
	maxIdleConnsPerHost int
	maxIdleConns        int
	maxConnsPerHost     int
}

func defaultOptions() *options {
	return &options{
		readBufferSize:      defaultReadBufferSize,
		maxIdleConnsPerHost: defaultMaxIdleConnsPerHost,
		maxIdleConns:        defaultMaxIdleConns,
		maxConnsPerHost:     defaultMaxConnsPerHost,
	}
}

// Option s3 session option
type Option func(*options)

// WithReadBufferSize set read buffer size
func WithReadBufferSize(size int) Option {
	return func(o *options) {
		o.readBufferSize = size
	}
}

// WithMaxIdleConnsPerHost set max idle conns per host
func WithMaxIdleConnsPerHost(conns int) Option {
	return func(o *options) {
		o.maxIdleConnsPerHost = conns
	}
}

// WithMaxIdleConns set max idle conns
func WithMaxIdleConns(conns int) Option {
	return func(o *options) {
		o.maxIdleConns = conns
	}
}

// WithMaxConnsPerHost set max conns per host
func WithMaxConnsPerHost(conns int) Option {
	return func(o *options) {
		o.maxConnsPerHost = conns
	}
}
