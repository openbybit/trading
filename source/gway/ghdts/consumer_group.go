package ghdts

import (
	"context"
	"sync/atomic"

	"git.bybit.com/tni_go/dtssdk_go_interface/dtssdk"
)

func newGroupConsumer(pushMode bool, waitTimeMs int32, groupID string, topic string, offsetOpt *int64, consumeFn ConsumeFunc, errorFn ConsumeErrorFunc, logger Logger) (*groupConsumer, error) {
	if waitTimeMs <= 0 {
		const defaultWaitTime = 100
		waitTimeMs = defaultWaitTime
	}

	var offset *int64
	if offsetOpt != nil {
		if x, err := GetOffset(topic, int32(*offsetOpt), 10000); err == nil {
			offset = &x
		}
	}

	c := &groupConsumer{
		pushMode:   pushMode,
		waitTimeMs: waitTimeMs,
		groupID:    groupID,
		topic:      topic,
		offset:     offset,
		consumeFn:  consumeFn,
		errorFn:    errorFn,
		logger:     logger,
	}
	return c, nil
}

// sdk: https://c1ey4wdv9g.larksuite.com/wiki/wikusQxBdIUW3LaHtPzErn3h2vb?appStyle=UI4&domain=doesnotexists.larksuite.com&locale=zh-CN&refresh=1&tabName=space&theme=dark&userId=7094178533047549958
type groupConsumer struct {
	ctx          context.Context
	consumer     dtssdk.CousumerGroup
	pullStopFlag *int32 // 通知pull协程退出
	pushMode     bool
	waitTimeMs   int32
	groupID      string
	topic        string
	offset       *int64
	consumeFn    ConsumeFunc
	errorFn      ConsumeErrorFunc
	logger       Logger
}

func (c *groupConsumer) GroupID() string {
	return c.groupID
}

func (c *groupConsumer) Topic() string {
	return c.topic
}

func (c *groupConsumer) Close() error {
	c.logger.Printf("[ghdts] close")

	if c.pullStopFlag != nil {
		atomic.StoreInt32(c.pullStopFlag, 1)
		c.pullStopFlag = nil
	}

	if c.consumer != nil {
		c.consumer.Close()
		c.consumer = nil
	}

	return nil
}

func (c *groupConsumer) Consume(ctx context.Context) error {
	consumer, err := dtssdk.NewConsumerGroup(c.pushMode)
	if err != nil {
		return err
	}

	// 仅初始化时设置一次
	if c.offset != nil {
		_ = consumer.Seek(c.topic, *c.offset)
		c.offset = nil
	}

	if err := consumer.Subscribe(c.groupID, []string{c.topic}, c); err != nil {
		return err
	}

	c.ctx = ctx
	c.consumer = consumer
	if !c.pushMode {
		// 每次创建一个新的指针,因为有restart逻辑
		c.pullStopFlag = new(int32)
		go c.pullLoop(ctx, c.consumer, c.pullStopFlag)
	}

	return nil
}

func (c *groupConsumer) pullLoop(ctx context.Context, consumer dtssdk.CousumerGroup, stopFlag *int32) {
	for {
		select {
		case <-ctx.Done():
			c.logger.Printf("[ghdts] pullLoop stopped by ctx")
			return
		default:
			if atomic.LoadInt32(stopFlag) == 1 {
				c.logger.Printf("[ghdts] pullLoop stopped by stop flag")
				return
			}
			consumer.PullMessages(c.waitTimeMs)
		}
	}
}

func (c *groupConsumer) OnConsumeCallback(msg *dtssdk.ConsumerMessage) {
	if err := c.consumeFn(context.Background(), msg); err == nil {
		c.consumer.CommitOffset(msg.Topic, msg.Offset)
	}
}

func (c *groupConsumer) OnConsumeGroupError(err error) {
	c.logger.Printf("[ghdts] OnConsumeGroupError err=%v", err)

	if err != nil {
		c.errorFn(c, err)
	}

	c.restart()
}

func (c *groupConsumer) OnConsumeTopicError(topic string, err error) {
	c.logger.Printf("[ghdts] OnConsumeTopicError, topic=%v, err=%v", topic, err)

	if err != nil {
		c.errorFn(c, err)
	}

	c.restart()
}

func (c *groupConsumer) restart() {
	if err := c.Close(); err != nil {
		c.logger.Printf("[ghdts] restart close fail, err=%v", err)
	}

	if err := c.Consume(c.ctx); err != nil {
		c.logger.Printf("[ghdts] restart consume fail, err=%v", err)
	}
}

func (c *groupConsumer) OnRebalanceStrategy(topics []string, subscriptions map[string]dtssdk.Subscription) map[string]dtssdk.AssignResult {
	res := make(map[string]dtssdk.AssignResult)
	// 不考虑粘性和owner，平均分配给每个消费者
	num := len(subscriptions)
	ts := make([][]string, num)
	for i, t := range topics {
		ts[i%num] = append(ts[i%num], t)
	}
	m := 0
	for key := range subscriptions {
		var ar dtssdk.AssignResult
		ar.Topics = ts[m]
		res[key] = ar
		m++
	}

	return res
}

func (c *groupConsumer) OnAssign(topics []string) {
	c.logger.Printf("[ghdts] OnAssign, topics: %v", topics)
}

func (c *groupConsumer) OnUnassign(topics []string) {
	c.logger.Printf("[ghdts] OnUnassign, topics: %v", topics)
}
