package ghdts

import "time"

type Options struct {
	pushMode     bool             // 是否使用推模式，默认pull模式
	waitTimeMs   int32            // pull等待时间
	groupID      string           // 组消费id
	offset       *int64           // 偏移
	consumeErrFn ConsumeErrorFunc // 错误回调函数
	logger       Logger           // 日志,默认系统控制台输出
}

type Option func(o *Options)

func WithPushMode(f bool) Option {
	return func(o *Options) {
		o.pushMode = f
	}
}

func WithWaitTime(d time.Duration) Option {
	return func(o *Options) {
		o.waitTimeMs = int32(d.Milliseconds())
	}
}

func WithGroupID(id string) Option {
	return func(o *Options) {
		o.groupID = id
	}
}

func WithOffset(offset int64) Option {
	return func(o *Options) {
		o.offset = &offset
	}
}

func WithConsumeErrorFunc(f ConsumeErrorFunc) Option {
	return func(o *Options) {
		o.consumeErrFn = f
	}
}

func WithLogger(l Logger) Option {
	return func(o *Options) {
		o.logger = l
	}
}
