package nacos

import (
	"context"
	"fmt"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gnacos"
	"code.bydev.io/fbu/gateway/gway.git/gtrace"
	"code.bydev.io/frameworks/nacos-sdk-go/v2/vo"
	"github.com/pkg/errors"

	"bgw/pkg/config_center"
	"bgw/pkg/remoting/nacos"
)

const (
	Get       = "nacos-get"
	GetPrefix = "nacos-prefix"
	Put       = "nacos-put"
	Del       = "nacos-del"
)

var (
	slowQueryThreshold                         = time.Millisecond * 10
	_                  config_center.Configure = &nacosConfigure{}
)

type nacosConfigure struct {
	client      gnacos.ConfigClient
	NamespaceId string
	Group       string
}

func NewNacosConfigure(ctx context.Context, opts ...Options) (config_center.Configure, error) {
	opt := &Option{
		group: gnacos.DEFAULT_GROUP,
	}

	for _, o := range opts {
		o(opt)
	}

	cc := nacosConfigure{
		NamespaceId: opt.namespace,
		Group:       opt.group,
	}

	cfg, err := nacos.GetNacosConfig(opt.namespace)
	if err != nil {
		return nil, fmt.Errorf("GetNacosConfig error, namespace:%s, err:%w", opt.namespace, err)
	}
	client, err := gnacos.NewConfigClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("gnacos.NewConfigClient error, namespace:%s, err:%w", opt.namespace, err)
	}
	cc.client = client

	return &cc, nil
}

func (n *nacosConfigure) Listen(ctx context.Context, key string, listener observer.EventListener) error {
	if data, err := n.Get(ctx, key); err == nil {
		event := &observer.DefaultEvent{
			Key:    key,
			Action: observer.EventTypeUpdate,
			Value:  data,
		}
		if err = listener.OnEvent(event); err != nil {
			glog.Error(ctx, "nacos:listen config On GET fail", glog.String("key", key), glog.String("error", err.Error()))
		}
	} else {
		glog.Error(ctx, "nacos:listen config On GET error", glog.String("key", key), glog.String("error", err.Error()))
	}

	err := n.client.ListenConfig(vo.ConfigParam{
		DataId: key,
		Group:  n.Group,
		OnChange: func(namespace, group, dataId, data string) {
			err := listener.OnEvent(&observer.DefaultEvent{
				Key:    dataId,
				Action: observer.EventTypeUpdate,
				Value:  data,
			})
			if err != nil {
				glog.Error(ctx, "nacos:listen config OnEvent fail", glog.String("key", key), glog.String("error", err.Error()))
				return
			}
		},
	})

	if err != nil {
		glog.Error(ctx, "nacos:listen config fail", glog.String("key", key), glog.String("error", err.Error()))
		return err
	}

	return nil
}

func (n *nacosConfigure) Get(ctx context.Context, key string) (string, error) {
	start := time.Now()
	span, _ := gtrace.Begin(ctx, Get)
	defer gtrace.Finish(span)

	defer func() {
		gtrace.Finish(span)
		timeCost := time.Now().Sub(start)
		if timeCost > slowQueryThreshold {
			glog.Info(ctx, "nacos get slow", glog.String("key", key), glog.Duration("cost", timeCost))
		}
	}()
	param := vo.ConfigParam{
		DataId: key,
		Group:  n.Group,
	}

	data, err := n.client.GetConfig(param)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return data, nil
}

func (n *nacosConfigure) GetChildren(ctx context.Context, key string) ([]string, []string, error) {
	span, _ := gtrace.Begin(ctx, GetPrefix)
	defer gtrace.Finish(span)

	panic("not implement")
}

func (n *nacosConfigure) Put(ctx context.Context, key, value string) error {
	span, _ := gtrace.Begin(ctx, Put)
	defer gtrace.Finish(span)

	param := vo.ConfigParam{
		DataId:  key,
		Content: value,
		Group:   n.Group,
	}

	ok, err := n.client.PublishConfig(param)
	if err != nil {
		return errors.WithStack(err)
	}
	if !ok {
		return errors.New("PublishConfig fail")
	}

	return nil
}

func (n *nacosConfigure) Del(ctx context.Context, key string) error {
	span, _ := gtrace.Begin(ctx, Del)
	defer gtrace.Finish(span)

	param := vo.ConfigParam{
		DataId: key,
		Group:  n.Group,
	}

	ok, err := n.client.DeleteConfig(param)
	if err != nil {
		return errors.WithStack(err)
	}
	if !ok {
		return errors.New("DeleteConfig fail")
	}
	return nil
}
