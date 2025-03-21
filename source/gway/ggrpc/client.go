package ggrpc

import (
	"context"
	"time"

	"code.bydev.io/frameworks/byone/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/keepalive"

	"code.bydev.io/fbu/gateway/gway.git/ggrpc/pool"
)

type ClientConnInterface = grpc.ClientConnInterface

func init() {
	pool.SetDialFunc(dial)
}

var defaultDialOptions = []grpc.DialOption{
	grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                30 * time.Second,
		Timeout:             10 * time.Second,
		PermitWithoutStream: true,
	}),
	grpc.WithConnectParams(grpc.ConnectParams{
		Backoff: backoff.Config{
			BaseDelay:  100 * time.Millisecond,
			Multiplier: 1.6,
			Jitter:     0.2,
			MaxDelay:   500 * time.Millisecond,
		},
		MinConnectTimeout: time.Second,
	}),
}

func Dial(ctx context.Context, target string, opts ...interface{}) (grpc.ClientConnInterface, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	dialOpts := dialOptions{}
	var grpcOpts []grpc.DialOption

	for _, opt := range opts {
		switch x := opt.(type) {
		case grpc.DialOption:
			grpcOpts = append(grpcOpts, x)
		case DialOption:
			x(&dialOpts)
		}
	}

	if !dialOpts.disableDefault {
		grpcOpts = append(defaultDialOptions, grpcOpts...)
	}

	if dialOpts.poolEnable {
		if len(grpcOpts) > 0 {
			dialOpts.poolOptions = append(dialOpts.poolOptions, pool.WithDialOptions(grpcOpts...))
		}
		p, err := pool.New(ctx, target, dialOpts.poolOptions...)
		if err != nil {
			return nil, err
		}

		c := &poolClient{pool: p}
		return c, nil
	} else {
		return dialWithBreaker(ctx, target, grpcOpts...)
	}
}

func dialWithBreaker(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	cli, err := zrpc.NewClient(newConfig(target, true), zrpc.WithDialOptions(opts...))
	if err != nil {
		return nil, err
	}
	return cli.Conn().(*grpc.ClientConn), nil
}

func dial(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	cli, err := zrpc.NewClient(newConfig(target, false), zrpc.WithDialOptions(opts...))
	if err != nil {
		return nil, err
	}
	return cli.Conn().(*grpc.ClientConn), nil
}

type poolClient struct {
	pool pool.Pool
}

func (c *poolClient) Invoke(ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption) error {
	cli, err := c.pool.Get(ctx)
	if err != nil {
		return err
	}
	err = cli.Client().Invoke(ctx, method, args, reply, opts...)
	_ = cli.Close()
	return err
}

func (c *poolClient) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	cli, err := c.pool.Get(ctx)
	if err != nil {
		return nil, err
	}

	res, err := cli.Client().NewStream(ctx, desc, method, opts...)
	_ = cli.Close()
	return res, err
}
