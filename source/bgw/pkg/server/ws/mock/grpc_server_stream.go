package mock

import (
	"context"
	"fmt"
	"time"

	envelopev1 "code.bydev.io/fbu/gateway/proto.git/pkg/envelope/v1"
	"google.golang.org/grpc/metadata"
)

func NewGrpcServerStream() *GrpcServerStream {
	return &GrpcServerStream{}
}

type GrpcServerStream struct {
	closed  bool
	toSdk   *envelopev1.SubscribeResponse
	pushMsg *envelopev1.SubscribeRequest
	md      metadata.MD
}

func (x *GrpcServerStream) SetPushMessage(msg *envelopev1.SubscribeRequest) {
	x.pushMsg = msg
}

func (x *GrpcServerStream) SetHeader(metadata.MD) error {
	return nil
}

func (x *GrpcServerStream) SendHeader(metadata.MD) error {
	return nil
}

func (x *GrpcServerStream) SetTrailer(md metadata.MD) {
	x.md = md
}

func (x *GrpcServerStream) Context() context.Context {
	return context.Background()
}

func (x *GrpcServerStream) SendMsg(m interface{}) error {
	res := m.(*envelopev1.SubscribeResponse)
	if res.Cmd == envelopev1.Command_COMMAND_ADMIN {
		x.toSdk = res
	}
	return nil
}

func (x *GrpcServerStream) RecvMsg(m interface{}) error {
	if x.closed {
		return fmt.Errorf("stream closed")
	}

	r, ok := m.(*envelopev1.SubscribeRequest)
	if !ok {
		return fmt.Errorf("invalid subscribe request type")
	}
	if x.toSdk != nil {
		q := x.toSdk
		x.toSdk = nil
		switch q.Cmd {
		case envelopev1.Command_COMMAND_ADMIN:
			r.Cmd = envelopev1.Command_COMMAND_ADMIN
			r.Header = &envelopev1.Header{RequestId: q.Header.RequestId}
			r.Admin = &envelopev1.Admin{
				Type:   q.Admin.Type,
				Result: &envelopev1.Admin_Result{},
			}
			return nil
		}
	}

	time.Sleep(time.Millisecond * 10)

	if x.pushMsg != nil {
		r.Cmd = x.pushMsg.Cmd
		r.Header = x.pushMsg.Header
		r.MemberId = x.pushMsg.MemberId
		r.Topics = x.pushMsg.Topics
		r.Data = x.pushMsg.Data
		r.Flag = x.pushMsg.Flag
		r.SessionId = x.pushMsg.SessionId
		r.PushMessages = x.pushMsg.PushMessages
	}

	return nil
}

func (x *GrpcServerStream) Close() {
	x.closed = true
}
