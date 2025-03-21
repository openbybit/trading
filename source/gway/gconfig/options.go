package gconfig

type Options struct {
	Group    string // 用于指定nacos group
	ForceGet bool   // 当listen时,是否强制Get一次初始数据,默认false不触发
	Prefix   bool   // 是否前缀匹配,默认false,etcd中Listen和Delete使用,Get由于返回一个值,不使用前缀
	Logger   Logger //
}

func (o *Options) Init(opts ...Option) {
	o.Logger = nil
	for _, fn := range opts {
		fn(o)
	}
}

type Option func(o *Options)

func WithGroup(group string) Option {
	return func(o *Options) {
		o.Group = group
	}
}

func WithForceGet(v bool) Option {
	return func(o *Options) {
		o.ForceGet = v
	}
}

func WithPrefix() Option {
	return func(o *Options) {
		o.Prefix = true
	}
}

func WithLogger(l Logger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}
