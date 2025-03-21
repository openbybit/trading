package gkafka

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"code.bydev.io/frameworks/byone/core/service"
	"code.bydev.io/frameworks/byone/kafka"
)

// GOARCH=amd64 go test -gcflags='-N -l'
func TestNew(t *testing.T) {
	t.Run("test error", func(t *testing.T) {
		cs := New(Config{})
		_ = cs.Start(context.Background())
		_ = cs.Stop()
	})

	t.Run("test new", func(t *testing.T) {
		address := "k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9090,k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9091,k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9092"
		brokers := os.Getenv("GWAY_KAFKA_BROKERS")
		os.Setenv("GWAY_KAFKA_BROKERS", address)
		defer os.Setenv("GWAY_KAFKA_BROKERS", brokers)
		cs := New(Config{
			Topic:  "trading_result.USDT.0_10",
			Offset: []int64{-1},
		})
		_ = cs.Start(context.Background())
		_ = cs.Stop()
	})

	t.Run("test new group", func(t *testing.T) {
		address := "k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9090,k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9091,k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9092"
		brokers := strings.Split(address, ",")
		cs := New(Config{
			Topic:   "trading_result.USDT.0_10",
			GroupId: "gway_unittest",
			Brokers: brokers,
		})
		go func() {
			_ = cs.Start(context.Background())
			_ = cs.Stop()
		}()
	})
}

func TestPartitionConsume(t *testing.T) {
	err := fmt.Errorf("some error")
	client := &mockClient{newErr: err}
	client1 := &mockClient{partitionErr: err}
	c := &consumer{cfg: &Config{}}
	_ = c.partitionConsume(client)
	_ = c.partitionConsume(client1)
}

func TestGroupConsume(t *testing.T) {
	t.Run("some error", func(t *testing.T) {
		err := fmt.Errorf("some error")
		client := &mockClient{newErr: err}
		c := consumer{cfg: &Config{}}
		_ = c.groupConsume(client)
	})

	t.Run("basic", func(t *testing.T) {
		client := &mockClient{}
		c := consumer{cfg: &Config{
			ConsumerHandler: func(ctx context.Context, message *Message) {},
		}}
		_ = c.groupConsume(client)
	})
}

type mockClient struct {
	kafka.Client
	service      service.Service
	newErr       error
	partitionErr error
}

func (c *mockClient) NewConsumer() (kafka.Consumer, error) {
	return nil, c.newErr
}

func (c *mockClient) Partitions(topic string) ([]int32, error) {
	return nil, c.partitionErr
}

func (c *mockClient) NewConsumerGroup(cc kafka.GroupConfig, handler kafka.Handler) (service.Service, error) {
	return &mockService{handler: handler}, c.newErr
}

type mockService struct {
	handler kafka.Handler
}

func (s *mockService) Start() { s.handler(context.Background(), &kafka.Message{}) }
func (s *mockService) Stop()  {}
