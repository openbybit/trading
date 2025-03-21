package etcd

import (
	"context"
	"errors"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/getcd"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gtrace"

	"bgw/pkg/config_center"
	retcd "bgw/pkg/remoting/etcd"
)

const (
	Get       = "etcd-get"
	GetPrefix = "etcd-prefix"
	Put       = "etcd-put"
	Del       = "etcd-del"
)

var (
	slowQueryThreshold                         = time.Millisecond * 10
	_                  config_center.Configure = &EtcdConfigure{}

	ErrKVPairNotFound  = errors.New("kv not found")
	ErrNilETCDV3Client = errors.New("etcd raw client is nil")
)

type EtcdConfigure struct {
	client getcd.Client
	// reconnectTimeout time.Deadline
}

func NewEtcdConfigure(ctx context.Context) (*EtcdConfigure, error) {
	client, err := retcd.NewConfigClient(ctx)
	if err != nil {
		return nil, err
	}

	return &EtcdConfigure{
		client: client,
	}, nil
}

func (e *EtcdConfigure) Get(ctx context.Context, key string) (string, error) {
	start := time.Now()
	span, _ := gtrace.Begin(ctx, Get)
	defer func() {
		gtrace.Finish(span)
		timeCost := time.Now().Sub(start)
		if timeCost > slowQueryThreshold {
			glog.Info(ctx, "etcd get slow", glog.String("key", key), glog.Duration("cost", timeCost))
		}
	}()

	data, err := e.client.Get(key)
	if err != nil {
		if errors.Is(err, getcd.ErrKVPairNotFound) {
			return "", ErrKVPairNotFound
		}
		return "", err
	}

	return data, nil
}

func (e *EtcdConfigure) GetChildren(ctx context.Context, key string) ([]string, []string, error) {
	span, _ := gtrace.Begin(ctx, GetPrefix)
	defer gtrace.Finish(span)

	ks, vs, err := e.client.GetChildrenKVList(key)
	if err != nil {
		if errors.Is(err, getcd.ErrNilETCDV3Client) {
			return nil, nil, ErrNilETCDV3Client
		}
		return nil, nil, err
	}
	return ks, vs, nil
}

func (e *EtcdConfigure) Listen(ctx context.Context, key string, listener observer.EventListener) error {
	el := getcd.NewEventListener(ctx, e.client)
	el.ListenWithChildren(key, listener)
	return nil
}

func (e *EtcdConfigure) Put(ctx context.Context, key, value string) error {
	span, _ := gtrace.Begin(ctx, Put)
	defer gtrace.Finish(span)

	err := e.client.Put(key, value)
	if err != nil {
		return err
	}

	return nil
}

func (e *EtcdConfigure) Del(ctx context.Context, key string) error {
	span, _ := gtrace.Begin(ctx, Del)
	defer gtrace.Finish(span)

	return e.client.Delete(key)
}
