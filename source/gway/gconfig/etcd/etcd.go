package etcd

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gconfig"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
)

func init() {
	gconfig.Register("etcd", New)
}

var (
	clientMap = make(map[string]*clientv3.Client)
	clientMux sync.Mutex
)

func New(addr string) (gconfig.Configure, error) {
	cli, err := newClient(addr)
	if err != nil {
		return nil, err
	}

	return NewWithClient(cli), nil
}

func newClient(addr string) (*clientv3.Client, error) {
	if addr == "" {
		addr = os.Getenv("_ETCD_ENDPOINTS")
	}
	if addr == "" {
		return nil, fmt.Errorf("empty etcd address")
	}

	if !strings.Contains(addr, "://") {
		addr = "etcd://" + addr
	}

	clientMux.Lock()
	defer clientMux.Unlock()

	if cli, ok := clientMap[addr]; ok {
		return cli, nil
	}

	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	q := u.Query()

	endpoints := strings.Split(u.Host, ",")
	username := u.User.Username()
	password, _ := u.User.Password()
	timeout := time.Second * 10
	if q.Has("timeout") {
		if t, err := time.ParseDuration(q.Get("timeout")); err == nil {
			timeout = t
		}
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		Username:    username,
		Password:    password,
		DialTimeout: timeout,
		DialOptions: []grpc.DialOption{
			grpc.WithBlock(),
			grpc.WithConnectParams(
				grpc.ConnectParams{
					Backoff: backoff.Config{
						BaseDelay:  time.Second,
						Multiplier: 1.6,
						Jitter:     0.2,
						MaxDelay:   30 * time.Second,
					},
					MinConnectTimeout: 2 * time.Second,
				},
			),
		},
	})

	if err != nil {
		return nil, err
	}

	clientMap[addr] = cli

	return cli, nil
}

func NewWithClient(cli *clientv3.Client) gconfig.Configure {
	return &etcdConfigure{client: cli}
}

type etcdConfigure struct {
	client *clientv3.Client
}

func (c *etcdConfigure) Get(ctx context.Context, key string, opts ...gconfig.Option) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	rsp, err := c.client.Get(ctx, key)
	if err != nil {
		return "", err
	}

	if len(rsp.Kvs) == 0 {
		return "", nil
	}

	return string(rsp.Kvs[0].Value), nil
}

func (c *etcdConfigure) Put(ctx context.Context, key string, value string, opts ...gconfig.Option) error {
	if ctx == nil {
		ctx = context.Background()
	}
	_, err := c.client.Put(ctx, key, value)
	if err != nil {
		return fmt.Errorf("etcd put fail, key:%v, value: %v, err: %w", key, value, err)
	}

	return nil
}

func (c *etcdConfigure) Delete(ctx context.Context, key string, opts ...gconfig.Option) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	o := gconfig.Options{}
	o.Init(opts...)

	if o.Prefix {
		_, err = c.client.Delete(ctx, key, clientv3.WithPrefix())
	} else {
		_, err = c.client.Delete(ctx, key)
	}

	if err != nil {
		return fmt.Errorf("error delete fail, key: %v, err: %w", key, err)
	}
	return nil
}

func (c *etcdConfigure) Listen(ctx context.Context, key string, listener gconfig.Listener, opts ...gconfig.Option) error {
	if ctx == nil {
		ctx = context.Background()
	}

	o := gconfig.Options{}
	o.Init(opts...)

	if o.ForceGet && !o.Prefix {
		rsp, err := c.client.Get(ctx, key)
		if err != nil {
			return err
		}
		value := ""
		if len(rsp.Kvs) > 0 {
			value = string(rsp.Kvs[0].Value)
		}

		if value != "" {
			listener.OnEvent(&gconfig.Event{Type: gconfig.EventTypeUpdate, Key: key, Value: value})
		}
	}

	go func() {
		c.doWatch(ctx, key, listener, o.Prefix, o.Logger)
	}()

	return nil
}

func (c *etcdConfigure) doWatch(ctx context.Context, key string, listener gconfig.Listener, prefix bool, logger gconfig.Logger) {
	etcdOpts := []clientv3.OpOption{}
	if prefix {
		etcdOpts = append(etcdOpts, clientv3.WithPrefix())
	}
	ch := c.client.Watch(ctx, key, etcdOpts...)
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				if logger != nil {
					logger.Infof(context.Background(), "[gconfig] etcd watch exit, key: %v", key)
				} else {
					log.Printf("[gconfig] etcd watch exit, key: %v\n", key)
				}
				return
			}

			if msg.Err() != nil {
				if logger != nil {
					logger.Errorf(context.Background(), "[gconfig] etcd watch fail, key: %v, err: %v", key, msg.Err())
				} else {
					log.Printf("[gconfig] etcd watch fail, key: %v, err: %v", key, msg.Err())
				}
				continue
			}

			for _, ev := range msg.Events {
				var et gconfig.EventType
				if ev.Type == clientv3.EventTypePut {
					if ev.IsCreate() {
						et = gconfig.EventTypeCreate
					} else {
						et = gconfig.EventTypeUpdate
					}
				} else {
					et = gconfig.EventTypeDelete
				}

				listener.OnEvent(&gconfig.Event{Type: et, Key: string(ev.Kv.Key), Value: string(ev.Kv.Value)})
			}
		}
	}
}
