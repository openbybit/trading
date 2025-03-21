package ghdts

import (
	"context"
	"testing"
	"time"
)

func TestConsume(t *testing.T) {
	// unify-test-1
	// dir: $HOME/data/resource-mgr-sdk/conf
	topic := "trading_result.USDT.0_1"

	consumers := make([]Consumer, 0)
	for i := 0; i < 2; i++ {
		c, err := Consume(context.Background(), topic, newConsumeHandler(t), WithConsumeErrorFunc(newErrorHandler(t)))
		if err != nil {
			t.Error(err)
		}
		consumers = append(consumers, c)
	}

	time.Sleep(time.Second * 20)

	for _, c := range consumers {
		c.Close()
	}

	time.Sleep(time.Second)
}

func TestConsumeOffset(t *testing.T) {
	// 测试offset是否生效
	topic := "trading_result.USDT.0_1"
	c, err := Consume(context.Background(), topic, newConsumeHandler(t), WithConsumeErrorFunc(newErrorHandler(t)), WithOffset(OffsetOldest))
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second * 20)

	c.Close()
	time.Sleep(time.Second)
}

func TestConsumeGroup(t *testing.T) {
	// unify_test_1
	topic := "trading_result.USDT.0_1"

	// 预期只能有一个输出
	consumers := make([]Consumer, 0)
	for i := 0; i < 2; i++ {
		c, err := Consume(context.Background(), topic, newConsumeHandler(t), WithConsumeErrorFunc(newErrorHandler(t)), WithGroupID("gway_test"))
		if err != nil {
			t.Error(err)
		}
		consumers = append(consumers, c)
	}

	time.Sleep(time.Second * 20)

	for _, c := range consumers {
		c.Close()
	}

	time.Sleep(time.Second)
}

func newConsumeHandler(t *testing.T) ConsumeFunc {
	return func(ctx context.Context, msg *ConsumerMessage) error {
		t.Logf(
			"onConsume, uid:%s, topic: %v, offset: %v, size:%v, timestamp:%v, ConfirmTimestamp: %v\n",
			GetHeader(msg.Headers, []byte("user_id")), msg.Topic, msg.Offset, len(msg.Value), msg.CommitTimeStamp.String(), msg.ConfirmTimestamp.String(),
		)
		return nil
	}
}

func newErrorHandler(t *testing.T) ConsumeErrorFunc {
	return func(c Consumer, err error) {
		t.Logf("consume topic:%v, error: %v", c.Topic(), err)
	}
}
