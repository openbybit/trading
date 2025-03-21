package smp

import (
	"context"
	"fmt"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/frameworks/byone/kafka"

	"bgw/pkg/diagnosis"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gapp"
	"code.bydev.io/fbu/gateway/gway.git/gkafka"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gsmp"

	"bgw/pkg/common"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/kafkaconsume"
	"bgw/pkg/config"
	"bgw/pkg/discovery"
)

const (
	impServer = "instmng"
	smpKey    = "smp"

	smpTopic = "imp.smp_group_msg"
)

var (
	grouper gsmp.Grouper
	once    sync.Once
	alert   sync.Once
)

func GetGrouper(ctx context.Context) (gsmp.Grouper, error) {
	var err error
	once.Do(func() {
		smpCfg := config.Global.ComponentConfig.Smp
		registry := smpCfg.GetOptions("registry", impServer)
		ns := config.GetNamespace()
		if env.IsProduction() {
			ns = constant.DEFAULT_NAMESPACE
		}

		cfg := &gsmp.Config{
			Registry:  registry,
			Group:     constant.DEFAULT_GROUP,
			Namespace: ns,
		}

		url, e := common.NewURL(registry,
			common.WithProtocol(constant.NacosProtocol),
			common.WithGroup(constant.DEFAULT_GROUP),
			common.WithNamespace(ns),
		)
		if e != nil {
			err = e
			glog.Info(ctx, "imp NewURL error", glog.String("err", err.Error()))
			return
		}
		glog.Info(ctx, "imp NewURL ok", glog.String("url", url.String()))

		sr := discovery.NewServiceRegistry(ctx)
		if err = sr.Watch(context.TODO(), url); err != nil {
			glog.Info(ctx, "imp Watch error", glog.String("err", err.Error()))
			return
		}

		addr := smpCfg.Address
		if addr != "" {
			cfg.Discovery = newStaticDiscovery(addr)
		} else {
			cfg.Discovery = newDiscovery(ctx, url)
		}
		grouper, err = gsmp.New(cfg)
		if err != nil {
			return
		}

		kafkaconsume.AsyncHandleKafkaMessage(ctx, smpTopic,
			config.Global.SmpKafkaCli, handleSmpMsg, onErr)

		grouper.Init(ctx)
		registerAdmin()
		_ = diagnosis.Register(&diagnose{
			svc:   grouper,
			kCfg:  config.Global.SmpKafkaCli,
			group: cfg.Group,
		})
	})
	if grouper == nil {
		err = fmt.Errorf("smp grouper error: %w", err)
		alert.Do(func() {
			galert.Error(context.TODO(), "openapi smp GetGrouper error", galert.WithField("err", err.Error()))
		})
		return nil, err
	}

	return grouper, err
}

// discovery
func newDiscovery(ctx context.Context, url *common.URL) gsmp.Discovery {
	sr := discovery.NewServiceRegistry(ctx)

	return func(ctx context.Context, registry, namespace, group string) (addrs []string) {
		ins := sr.GetInstances(url)
		res := make([]string, 0, len(ins))
		for _, in := range ins {
			res = append(res, in.GetAddress(constant.GrpcProtocol))
		}
		return res
	}
}

// use for dev and test env
func newStaticDiscovery(addr string) gsmp.Discovery {
	return func(ctx context.Context, registry, namespace, group string) (addrs []string) {
		return []string{addr}
	}
}

// kafka handler
func handleSmpMsg(ctx context.Context, msg *gkafka.Message) {
	glog.Info(ctx, "smp msg", glog.String("msg", string(msg.Value)), glog.Int64("offset", msg.Offset))
	err := grouper.HandleMsg(msg.Value)
	if err != nil {
		glog.Error(ctx, "smp msg err", glog.Int64("offset", msg.Offset), glog.String("err", err.Error()))
	}
}

func onErr(err *gkafka.ConsumerError) {
	if err != nil {
		galert.Error(context.Background(), "smp consumer err "+err.Error())
	}
}

// register admin to get smp group
// http://localhost:6480/admin?cmd=smp&params={{uid}}
func registerAdmin() {
	if grouper == nil {
		return
	}
	gapp.RegisterAdmin("smp", "get smp group", OnSmpAdmin)
}

type resp struct {
	GroupID int32
}

func OnSmpAdmin(args gapp.AdminArgs) (interface{}, error) {
	uid := args.GetInt64At(0)
	gid, err := grouper.GetGroup(context.Background(), uid)
	if err != nil {
		return nil, err
	}
	return resp{gid}, nil
}

type diagnose struct {
	svc   gsmp.Grouper
	kCfg  kafka.UniversalClientConfig
	group string
}

func (o *diagnose) Key() string {
	return impServer
}

func (o *diagnose) Diagnose(ctx context.Context) (interface{}, error) {
	r := make(map[string]interface{})
	r["kafka"] = diagnosis.DiagnoseKafka(ctx, "imp.smp_group_msg", o.kCfg)
	r["grpc"] = diagnosis.DiagnoseGrpcUpstream(ctx, impServer, config.GetRegistryNamespace(), o.group)
	return r, nil
}
