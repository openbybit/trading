package ws

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"bgw/pkg/server/ws/mock"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	envelopev1 "code.bydev.io/fbu/gateway/proto.git/pkg/envelope/v1"
	"code.bydev.io/frameworks/byone/core/proc"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

func TestGrpcServer(t *testing.T) {
	getMetadata("abc")

	var g grpcServer

	getConfigMgr().LoadStaticConfig()
	cfg := getStaticConf()
	cfg.RPC.ListenType = "all"
	cfg.App.Cluster = "site"

	_ = g.Start()
	time.Sleep(time.Second)
	g.Stop()
	proc.Shutdown()
}

type SubscribeServerMock struct {
	grpc.ServerStream
	ctx context.Context
	req *envelopev1.SubscribeRequest
	err error
}

func (m *SubscribeServerMock) Context() context.Context {
	return m.ctx
}

func (m *SubscribeServerMock) Recv() (*envelopev1.SubscribeRequest, error) {
	return m.req, m.err
}

func (m *SubscribeServerMock) Send(*envelopev1.SubscribeResponse) error {
	return nil
}

func TestSubscribe(t *testing.T) {
	glog.SetLevel(glog.FatalLevel)

	initExchange()
	srv := grpcServer{}

	t.Run("success", func(t *testing.T) {
		req := &envelopev1.SubscribeRequest{
			Header: &envelopev1.Header{
				AppId:          "mock-appid",
				ConnectorId:    "mock-connectorid",
				UserShardIndex: -1,
				UserShardTotal: 0,
				FocusEvents:    0,
				TopicConfigs: []*envelopev1.TopicConfig{
					{
						Name: "sub_public_topic",
						Type: envelopev1.PushType_PUSH_TYPE_PUBLIC,
						Mode: "full",
					},
					{
						Name: "sub_private_topic",
						Type: envelopev1.PushType_PUSH_TYPE_PRIVATE,
					},
					{
						Name: "",
						Type: envelopev1.PushType_PUSH_TYPE_PUBLIC,
						Mode: "full",
					},
				},
			},
			Topics: []string{"topic1"},
		}
		pushMsg := &envelopev1.SubscribeRequest{
			Cmd:      envelopev1.Command_COMMAND_PUSH,
			Header:   &envelopev1.Header{},
			MemberId: 1,
			Topics:   []string{"mock-topic1"},
			Data:     []byte("test"),
		}
		stream := mock.NewGrpcServerStream()
		stream.SetPushMessage(pushMsg)
		addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
		ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: addr})
		go func() {
			// Subscribe方法会阻塞
			err := srv.Subscribe(&SubscribeServerMock{ServerStream: stream, ctx: ctx, req: req})
			assert.Nil(t, err)
		}()
		time.Sleep(time.Millisecond)
		// 重复注册
		dupErr := srv.Subscribe(&SubscribeServerMock{ServerStream: stream, ctx: ctx, req: req})
		assert.NotNil(t, dupErr)
		stream.Close()
		time.Sleep(time.Millisecond)
	})

	t.Run("recv err", func(t *testing.T) {
		err := srv.Subscribe(&SubscribeServerMock{err: errors.New("not implemented")})
		assert.NotNil(t, err)
	})

	t.Run("no header", func(t *testing.T) {
		err := srv.Subscribe(&SubscribeServerMock{req: &envelopev1.SubscribeRequest{}})
		assert.NotNil(t, err)
	})

	t.Run("check config", func(t *testing.T) {
		err := srv.checkSubscribeConfig(&envelopev1.SubscribeRequest{})
		assert.NotNilf(t, err, "no header")
		err = srv.checkSubscribeConfig(&envelopev1.SubscribeRequest{Header: &envelopev1.Header{ConnectorId: "", AppId: ""}})
		assert.NotNilf(t, err, "invalid appid")
		err = srv.checkSubscribeConfig(&envelopev1.SubscribeRequest{Header: &envelopev1.Header{ConnectorId: "a", AppId: "a", UserShardIndex: -1, UserShardTotal: 2}})
		assert.Nilf(t, err, "invalid shard index")
	})

}
