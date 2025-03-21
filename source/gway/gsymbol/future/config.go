package future

import (
	"time"

	"code.bydev.io/frameworks/sarama"
)

type Config struct {
	Server           string
	ResultTopic      string
	ResultAckTopic   string
	Addr             []string
	LogResult        bool
	AllBrokerSymbols bool
}

func newConfig() *sarama.Config {
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
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	saramaConfig.ChannelBufferSize = 10240
	saramaConfig.Version = sarama.V2_0_0_0

	return saramaConfig
}
