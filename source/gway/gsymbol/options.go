package gsymbol

const (
	brokerID_BYBIT = 0 // 主站ID
)

type Options struct {
	brokerID int
}

func (o *Options) init(opts ...Option) {
	for _, fn := range opts {
		fn(o)
	}
}

type Option func(o *Options)

// WithBrokerID 指定brokerID, 仅future使用
func WithBrokerID(id int) Option {
	return func(o *Options) {
		o.brokerID = id
	}
}
