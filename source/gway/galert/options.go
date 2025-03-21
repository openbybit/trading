package galert

type Options struct {
	webhook string
	title   string
	fields  []*Field
	footers []*Field
	limiter Limter
}

type Option func(o *Options)

func WithWebhook(v string) Option {
	return func(o *Options) {
		o.webhook = toLarkWebhook(v)
	}
}

func WithTitle(v string) Option {
	return func(o *Options) {
		o.title = v
	}
}

func WithField(key string, value interface{}) Option {
	return func(o *Options) {
		o.fields = append(o.fields, BasicField(key, value))
	}
}

func WithFields(fields ...*Field) Option {
	return func(o *Options) {
		o.fields = append(o.fields, fields...)
	}
}

func WithFooter(key string, value interface{}) Option {
	return func(o *Options) {
		o.footers = append(o.footers, BasicField(key, value))
	}
}

func WithFooters(fields ...*Field) Option {
	return func(o *Options) {
		o.footers = append(o.footers, fields...)
	}
}

func WithLimter(l Limter) Option {
	return func(o *Options) {
		o.limiter = l
	}
}
