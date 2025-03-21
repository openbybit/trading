package gkafka

import (
	"context"
	"time"

	"code.bydev.io/frameworks/byone/kafka"
	"code.bydev.io/frameworks/sarama"

	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
)

type Config struct {
	// Public parameters
	Topic string
	// In kafka consumer, you can set the offset of multiple partitions with the partition
	Offset []int64

	// Brokers of kafka. If null, the GWAY_KAFKA_BROKERS environment variable is fetched
	Brokers []string
	// Partition of kafka
	Partition []int32
	// GroupId for kafka. If GroupId is not empty, multicast is used
	GroupId string

	// username
	Username string
	// password
	Password string

	ConsumerHandler      func(ctx context.Context, message *Message)
	ConsumerErrorHandler func(consumerError *ConsumerError)

	// base Config
	*sarama.Config
}

func newBaseCfg() *sarama.Config {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Net.MaxOpenRequests = 1
	saramaConfig.Producer.MaxMessageBytes = 524288000
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Idempotent = true
	saramaConfig.Producer.Flush.Bytes = 40960000
	saramaConfig.Producer.Flush.Messages = 10000
	saramaConfig.Producer.Flush.Frequency = 1 * time.Millisecond
	saramaConfig.Producer.Flush.MaxMessages = 100000
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.ChannelBufferSize = 10240
	saramaConfig.Version = sarama.V2_4_0_0 // byone request V2_4_0_0 at least

	return saramaConfig
}

func toByoneConfig(cfg *Config) (kafka.UniversalClientConfig, error) {
	clientConfig := kafka.ClientConfig{
		Brokers: cfg.Brokers,
	}

	if cfg.Username != "" && cfg.Password != "" {
		clientConfig.AuthType = authPassword
		clientConfig.SaslUsername = cfg.Username
		clientConfig.SaslPassword = cfg.Password
		clientConfig.SaslMechanism = sarama.SASLTypeSCRAMSHA256
	}

	clientConfig.AzEnabled = true
	clientConfig.Version = cfg.Version.String()
	clientConfig.ChannelBufferSize = 10240 // default

	err := clientConfig.Validate()
	if err != nil {
		return kafka.UniversalClientConfig{}, err
	}

	producerCfg := kafka.SharedProducerConfig{
		MaxMessageBytes: cfg.Producer.MaxMessageBytes,
		RequiredAcks:    "all",
		Idempotent:      true,
		Flush: struct {
			Bytes       int           `json:",default=65536"`
			Messages    int           `json:",default=0"`
			Frequency   time.Duration `json:",default=1ms"`
			MaxMessages int           `json:",default=100000"`
		}(struct {
			Bytes       int
			Messages    int
			Frequency   time.Duration
			MaxMessages int
		}{
			Bytes:       cfg.Producer.Flush.Bytes,
			Messages:    cfg.Producer.Flush.Messages,
			Frequency:   cfg.Producer.Flush.Frequency,
			MaxMessages: cfg.Producer.Flush.MaxMessages,
		}),
	}

	consumerCfg := kafka.SharedConsumerConfig{}
	consumerCfg.FetchMinBytes = 1
	consumerCfg.FetchMaxBytes = 10000000
	consumerCfg.FetchMaxWaitTime = 5 * time.Millisecond
	if env.ServiceName() == "bgw" {
		consumerCfg.FetchMaxWaitTime = 500 * time.Millisecond
	}
	consumerCfg.BalanceStrategy = byoneRange
	consumerCfg.MaxProcessingTime = 100 * time.Millisecond

	config := kafka.UniversalClientConfig{
		Client:   clientConfig,
		Producer: producerCfg,
		Consumer: consumerCfg,
	}

	return config, nil
}
