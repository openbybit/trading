package ghdts

import (
	"context"
	"sync"

	"git.bybit.com/tni_go/dtssdk_go_interface/dtssdk"
)

var (
	gConsumer    dtssdk.Consumer
	gConsumerMux sync.RWMutex
)

func getConsumer() (dtssdk.Consumer, error) {
	gConsumerMux.RLock()
	if gConsumer != nil {
		gConsumerMux.RUnlock()
		return gConsumer, nil
	}
	gConsumerMux.RUnlock()

	// init global consumer
	gConsumerMux.Lock()
	defer gConsumerMux.Unlock()
	if gConsumer != nil {
		return gConsumer, nil
	}

	c, err := dtssdk.NewConsumer()
	if err == nil {
		gConsumer = c
	}

	return c, err
}

func newBasicConsumer(topic string, offsetOpt *int64, consumeFn ConsumeFunc, errorFn ConsumeErrorFunc, logger Logger) (*basicConsumer, error) {
	consumer, err := getConsumer()
	if err != nil {
		return nil, err
	}

	offset := int64(dtssdk.MaxOffset)
	if offsetOpt != nil {
		offset, _ = GetOffset(topic, int32(*offsetOpt), 10000)
	}

	tc, err := consumer.ConsumeTopicWithOptions(topic, offset)
	if err != nil {
		return nil, err
	}

	c := &basicConsumer{
		consumer:  tc,
		topic:     topic,
		consumeFn: consumeFn,
		errorFn:   errorFn,
		logger:    logger,
	}

	return c, nil
}

type basicConsumer struct {
	consumer  dtssdk.TopicConsumer
	topic     string
	consumeFn ConsumeFunc
	errorFn   ConsumeErrorFunc
	logger    Logger
}

func (c *basicConsumer) Topic() string   { return c.topic }
func (c *basicConsumer) GroupID() string { return "" }

func (c *basicConsumer) Close() error {
	if c.consumer != nil {
		return c.consumer.Close()
	}

	return nil
}

func (c *basicConsumer) Consume(ctx context.Context) error {
	go c.consumeLoop(ctx)
	return nil
}

func (c *basicConsumer) consumeLoop(ctx context.Context) {
	defer func() {
		c.consumer.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.consumer.Messages():
			if !ok {
				return
			}

			if err := c.consumeFn(ctx, msg); err != nil {
				c.errorFn(c, err)
			}
		case err, ok := <-c.consumer.Errors():
			if !ok {
				return
			}
			c.errorFn(c, err.Err)
		}
	}
}
