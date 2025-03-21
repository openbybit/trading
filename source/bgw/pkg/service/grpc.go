package service

import (
	"context"
	"fmt"
	"time"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gapp"
	"code.bydev.io/fbu/gateway/gway.git/ggrpc/pool"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/fbu/gateway/gway.git/gtrace"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// InitialWindowSize we set it 1GB is to provide system's throughput.
	InitialWindowSize = 1 << 30

	// InitialConnWindowSize we set it 1GB is to provide system's throughput.
	InitialConnWindowSize = 1 << 30

	// MaxSendMsgSize set max gRPC request message size sent to server.
	// If any request message size is larger than current value, an error will be reported from gRPC.
	MaxSendMsgSize = 4 << 30

	// MaxRecvMsgSize set max gRPC receive message size received from server.
	// If any message size is larger than current value, an error will be reported from gRPC.
	MaxRecvMsgSize = 4 << 30
)

// DefaultDialOptions default dial options
var DefaultDialOptions = []grpc.DialOption{
	grpc.WithInitialWindowSize(InitialWindowSize),
	grpc.WithInitialConnWindowSize(InitialConnWindowSize),
	grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(MaxSendMsgSize)),
	grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxRecvMsgSize)),
	grpc.WithUnaryInterceptor(withDefaultGrpcUnaryInterceptor()),
}

// InitGrpc init grpc
func InitGrpc() {
	pools := pool.NewPools(pool.WithDialOptions(
		grpc.WithInitialWindowSize(InitialWindowSize),
		grpc.WithInitialConnWindowSize(InitialConnWindowSize),
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(MaxSendMsgSize)),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxRecvMsgSize)),
		grpc.WithUnaryInterceptor(withDefaultGrpcUnaryInterceptor()),
	))
	pool.SetDefault(pools)

	queryPoolStatus := func(args gapp.AdminArgs) (interface{}, error) {
		target := args.GetStringAt(0)
		res, ok := pools.GetStatus(context.Background(), target)
		if !ok {
			return fmt.Sprintf("no pool for target %s", target), nil
		}
		return res, nil
	}

	// curl 'http://localhost:6480/admin?cmd=QueryPoolStatus&params={{target}}'
	gapp.RegisterAdmin("QueryPoolStatus", "query pool status", queryPoolStatus)
}

func withDefaultGrpcUnaryInterceptor() grpc.UnaryClientInterceptor {
	return func(originCtx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx := GetContext(originCtx)
		// recover
		defer func() {
			if e := recover(); e != nil {
				glog.Error(ctx, "grpc interceptor recover", glog.Any("error", e))
				if e1, ok := e.(error); ok {
					galert.Error(ctx, "grpc interceptor recover "+e1.Error())
				}
			}
		}()

		address := cc.Target()

		// metrics
		start := time.Now()

		// trace
		span, _ := gtrace.Begin(ctx, fmt.Sprintf("grpc-invoke:%s", method), opentracing.Tags{"addr": address})
		defer gtrace.Finish(span)
		traceId := gtrace.UberTraceIDFromSpan(span)
		if traceId != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "uber-trace-id", traceId)
			ctx = metadata.AppendToOutgoingContext(ctx, "traceparent", gtrace.TraceparentFromSpan(span))
		}

		err := invoker(ctx, method, req, reply, cc, opts...)
		// access log
		if glog.Enabled(glog.DebugLevel) || DynamicFromCtx(originCtx) {
			fields := []glog.Field{glog.String("addr", address), glog.String("method", method), glog.String("traceid", traceId)}
			var reqStr, respStr string
			req1, ok1 := req.(fmt.Stringer)
			if ok1 {
				reqStr = req1.String()
				if len(reqStr) > 1024 {
					reqStr = reqStr[:1024]
				}
				fields = append(fields, glog.String("req", reqStr))
			} else {
				fields = append(fields, glog.Any("req", req))
			}
			rsp1, ok2 := reply.(fmt.Stringer)
			if ok2 {
				respStr = rsp1.String()
				if len(respStr) > 1024 {
					respStr = respStr[:1024]
				}
				fields = append(fields, glog.String("rsp", respStr))
			} else {
				fields = append(fields, glog.Any("rsp", reply))
			}
			DynamicLog(originCtx, "grpc invoke", fields...)
		}

		gmetric.ObserveDefaultLatencySince(start, "grpc", method)
		return err
	}
}

// GetContext get jaeger context from fasthttp context
func GetContext(ctx context.Context) context.Context {
	if c, ok := ctx.(*types.Ctx); ok {
		if val, ok := c.UserValue(constant.ContextKey).(context.Context); ok && val != nil {
			if DynamicFromCtx(c) {
				val = context.WithValue(val, CtxDynamicLogKey{}, true)
			}
			return val
		}
		return context.Background()
	}
	return ctx
}
