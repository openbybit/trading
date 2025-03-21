package nacos

import (
	"time"
)

type Option struct {
	namespace string
	group     string
	timeout   time.Duration
}

type Options func(*Option)

func WithNameSpace(namespace string) Options {
	return func(o *Option) {
		o.namespace = namespace
	}
}

func WithGroup(group string) Options {
	return func(o *Option) {
		o.group = group
	}
}

// WithTimeout assigns time to opt.Timeout
func WithTimeout(time time.Duration) Options {
	return func(o *Option) {
		o.timeout = time
	}
}
