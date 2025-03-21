package ghdts

import (
	"context"
	"fmt"
	"log"
	"os"

	"git.bybit.com/tni_go/dtssdk_go_interface/dtssdk"
)

const (
	OffsetNewest = int64(dtssdk.MaxOffset)
	OffsetOldest = int64(dtssdk.MinOffset)
)

type ConsumerMessage = dtssdk.ConsumerMessage
type ConsumeFunc func(ctx context.Context, msg *dtssdk.ConsumerMessage) error
type ConsumeErrorFunc func(c Consumer, err error)

var defaultConsumeErrorFunc = func(c Consumer, err error) {}

// SetDefaultConsumeErrorFunc 设置默认的错误处理
func SetDefaultConsumeErrorFunc(f ConsumeErrorFunc) {
	if f != nil {
		defaultConsumeErrorFunc = f
	}
}

type Logger interface {
	Printf(format string, args ...interface{})
}

type Consumer interface {
	Topic() string
	GroupID() string
	Consume(ctx context.Context) error
	Close() error
}

// Consume
// HDTS: https://c1ey4wdv9g.larksuite.com/wiki/wikuso3aKmW905hAvYIrCaWvonh?appStyle=UI4&domain=doesnotexists.larksuite.com&locale=zh-CN&refresh=1&tabName=space&theme=dark&userId=7094178533047549958
func Consume(ctx context.Context, topic string, consumeFn ConsumeFunc, opts ...Option) (Consumer, error) {
	o := &Options{}
	for _, fn := range opts {
		fn(o)
	}

	if o.consumeErrFn == nil {
		o.consumeErrFn = defaultConsumeErrorFunc
	}

	if o.logger == nil {
		o.logger = log.New(os.Stdout, "", 0)
	}

	if consumeFn == nil {
		return nil, fmt.Errorf("consume handle is nil")
	}
	// ensuer safe
	safeConsumeFn := func(ctx context.Context, msg *dtssdk.ConsumerMessage) (err error) {
		defer func() {
			if e := recover(); e != nil {
				err = fmt.Errorf("%v", e)
			}
		}()

		return consumeFn(ctx, msg)
	}

	ensureInitSDK()

	var consumer Consumer
	var err error
	if o.groupID != "" {
		consumer, err = newGroupConsumer(o.pushMode, o.waitTimeMs, o.groupID, topic, o.offset, safeConsumeFn, o.consumeErrFn, o.logger)
	} else {
		consumer, err = newBasicConsumer(topic, o.offset, safeConsumeFn, o.consumeErrFn, o.logger)
	}

	if err != nil {
		return nil, err
	}

	if err := consumer.Consume(ctx); err != nil {
		return nil, err
	}

	return consumer, nil
}
