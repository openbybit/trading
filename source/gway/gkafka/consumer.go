package gkafka

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"

	"code.bydev.io/frameworks/byone/core/threading"
	"code.bydev.io/frameworks/byone/kafka"
)

type (
	Message       = kafka.Message
	ConsumerError = kafka.ConsumerError
)

const (
	// broker environment variables, separated by ",".
	brokersEnv   = "GWAY_KAFKA_BROKERS"
	authPassword = "password"

	byoneRange   = "range"
	offsetNewest = "newest"
)

var errEmptyBrokers = errors.New("[bad config]empty brokers list")

type Consumer interface {
	Start(ctx context.Context) error
	Stop() error
}

type consumer struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	cfg        *Config
}

func New(cfg Config) Consumer {
	c := &consumer{
		cfg: &cfg,
	}
	return c
}

func (c *consumer) Start(ctx context.Context) error {
	c.ctx, c.cancelFunc = context.WithCancel(ctx)
	return c.consume()
}

// Stop consumer no need stop now
func (c *consumer) Stop() error {
	c.cancelFunc()
	return nil
}

func (c *consumer) consume() error {
	if len(c.cfg.Brokers) == 0 {
		addr := os.Getenv(brokersEnv)
		if addr == "" {
			return errEmptyBrokers
		}
		c.cfg.Brokers = strings.Split(addr, ",")
	}

	if c.cfg.Config == nil {
		c.cfg.Config = newBaseCfg()
	}

	config, err := toByoneConfig(c.cfg)
	if err != nil {
		log.Printf("kafka transform byone kafka config, err %s", err.Error())
		return err
	}

	client, err := kafka.NewClient(config)
	if err != nil {
		log.Printf("new byone kafka client, err %s", err.Error())
		return err
	}

	if c.cfg.GroupId == "" {
		return c.partitionConsume(client)
	}

	return c.groupConsume(client)
}

func (c *consumer) partitionConsume(client kafka.Client) error {
	consumer, err := client.NewConsumer()
	if err != nil {
		log.Printf("kafka new consumer, err %s", err.Error())
		return err
	}

	partitions, err := client.Partitions(c.cfg.Topic)
	if err != nil {
		log.Printf("partitionConsume fetch partition err, topic %s, err %s", c.cfg.Topic, err.Error())
		return err
	}

	log.Printf("get partitins %v from topic %s", partitions, c.cfg.Topic)

	for i, partition := range partitions {
		p := partition
		offset := kafka.OffsetNewest
		if i < len(c.cfg.Offset) {
			offset = c.cfg.Offset[i]
		}
		threading.GoSafe(func() {
			_, err := consumer.ConsumePartition(c.cfg.Topic, p, offset, c.cfg.ConsumerHandler, c.cfg.ConsumerErrorHandler)
			if err != nil {
				log.Printf("partitionConsume partition err, topic %s, partition %d, offset %d, err %s", c.cfg.Topic, p, offset, err.Error())
				return
			}
		})
	}
	return nil
}

func (c *consumer) groupConsume(client kafka.Client) error {
	groupCfg := kafka.GroupConfig{
		Topic:         c.cfg.Topic,
		GroupID:       c.cfg.GroupId,
		InitialOffset: offsetNewest,
	}

	handler := func(ctx context.Context, message *Message) error {
		c.cfg.ConsumerHandler(ctx, message)
		return nil
	}

	cg, err := client.NewConsumerGroup(groupCfg, handler)
	if err != nil {
		return err
	}

	cg.Start()
	return nil
}
