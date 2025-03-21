package kafkaconsume

import (
	"context"
	"fmt"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/frameworks/byone/core/threading"
	"code.bydev.io/frameworks/byone/kafka"
)

type Handler = func(ctx context.Context, msg *kafka.Message)
type errorHandler = func(err *kafka.ConsumerError)

func AsyncHandleKafkaMessage(ctx context.Context, topic string, config kafka.UniversalClientConfig, handler Handler, errHandler errorHandler) {
	threading.GoSafe(func() {
		client, err := kafka.NewClient(config)
		if err != nil {
			glog.Error(ctx, fmt.Sprintf("kafka new client failed,topic = %s, config = %v, err = %s", topic, config, err.Error()))
			galert.Error(ctx, fmt.Sprintf("kafka new client failed,topic = %s, config = %v, err = %s", topic, config, err.Error()))
			return
		}

		partitions, err := client.Partitions(topic)
		if err != nil {
			glog.Error(ctx, fmt.Sprintf("kafkaClient partitions failed,topic = %s, err = %s", topic, err.Error()))
			galert.Error(ctx, fmt.Sprintf("kafkaClient partitions failed,topic = %s, err = %s", topic, err.Error()))
			return
		}

		consumer, err := client.NewConsumer()
		if err != nil {
			glog.Error(ctx, fmt.Sprintf("kafkaClient newConsumer failed,topic = %s, err = %s", topic, err.Error()))
			galert.Error(ctx, fmt.Sprintf("kafkaClient newConsumer failed,topic = %s, err = %s", topic, err.Error()))
			return
		}

		for _, partition := range partitions {
			_, err := consumer.ConsumePartition(topic, partition, kafka.OffsetNewest, handler, errHandler)
			if err != nil {
				glog.Error(ctx, fmt.Sprintf("kafkaClient consumePartition failed,topic = %s, err = %s", topic, err.Error()))
				galert.Error(ctx, fmt.Sprintf("kafkaClient consumePartition failed,topic = %s, err = %s", topic, err.Error()))
				return
			}
		}
	})
}
