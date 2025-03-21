package diagnosis

import (
	"context"
	"fmt"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/frameworks/byone/kafka"
	"code.bydev.io/frameworks/byone/zrpc"

	"bgw/pkg/common/constant"
	"bgw/pkg/config"
)

func ConfigPreflight() {
	go configPreflight()
}

func configPreflight() {
	defer func() {
		if r := recover(); r != nil {
			galert.Error(context.Background(), fmt.Sprintf("config preflight falied, %v", r))
		}
	}()

	ctx := context.Background()
	var res Result
	// etcd and redis
	res = DiagnoseEtcd(ctx)
	glog.Info(ctx, "etcd diagnose res", glog.Any("res", res))
	if len(res.Errs) > 0 {
		galert.Error(context.Background(), fmt.Sprintf("config preflight etcd falied, %v", res.Errs))
	}
	res = DiagnoseRedis(ctx)
	glog.Info(ctx, "redis diagnose res", glog.Any("res", res))
	if len(res.Errs) > 0 {
		galert.Error(context.Background(), fmt.Sprintf("config preflight redis falied, %v", res.Errs))
	}

	// grpc
	for _, cfg := range grpcCfg {
		res = DiagnoseGrpcDependency(ctx, cfg.cfg)
		glog.Info(ctx, "rpc diagnose res", glog.String("name", cfg.name), glog.Any("res", res))
		if len(res.Errs) > 0 {
			galert.Error(context.Background(),
				fmt.Sprintf("config preflight rpc falied, name: %s, err %v", cfg.name, res.Errs))
		}
	}

	// kafka
	for _, cfg := range kafkaCfgs {
		res = DiagnoseKafka(ctx, cfg.topic, cfg.cfg)
		glog.Info(ctx, "kafka diagnose res", glog.String("topic", cfg.topic), glog.Any("res", res))
		if len(res.Errs) > 0 {
			galert.Error(context.Background(),
				fmt.Sprintf("config preflight kafka falied, topic: %s, err: %v", cfg.topic, res.Errs))
		}
	}
}

type rpcCfg struct {
	name string
	cfg  zrpc.RpcClientConf
}

var grpcCfg = []rpcCfg{
	{
		name: "Compliance",
		cfg:  config.Global.Compliance,
	},
	{
		name: "UserServicePrivate",
		cfg:  config.Global.UserServicePrivate,
	},
	{
		name: "Masq",
		cfg:  config.Global.Masq,
	},
	{
		name: "Mixer",
		cfg:  config.Global.Mixer,
	},
	{
		name: "BanServicePrivate",
		cfg:  config.Global.BanServicePrivate,
	},
	{
		name: "UtaRouter",
		cfg:  config.Global.UtaRouter,
	},
	{
		name: "UtaRouterDa",
		cfg:  config.Global.UtaRouterDa,
	},
	{
		name: "Oauth",
		cfg:  config.Global.Oauth,
	},
}

type kafkaCfg struct {
	topic string
	cfg   kafka.UniversalClientConfig
}

var kafkaCfgs = []kafkaCfg{
	{
		topic: constant.EventRateLimitChange,
		cfg:   config.Global.KafkaCli,
	},
	{
		topic: "cht-compliance-wall-whitelist",
		cfg:   config.Global.ComplianceKafkaCli,
	},
	{
		topic: "cht-compliance-wall-kyc",
		cfg:   config.Global.ComplianceKafkaCli,
	},
	{
		topic: "cht-compliance-wall-strategy",
		cfg:   config.Global.ComplianceKafkaCli,
	},
	{
		topic: "cht-compliance-wall-event",
		cfg:   config.Global.ComplianceKafkaCli,
	},
	{
		topic: constant.EventMemberBanned,
		cfg:   config.Global.KafkaCli,
	},
	{
		topic: "imp.smp_group_msg",
		cfg:   config.Global.SmpKafkaCli,
	},
	{
		topic: constant.EventMemberTagChange,
		cfg:   config.Global.KafkaCli,
	},
	{
		topic: constant.EventSpecialSubMemberCreate,
		cfg:   config.Global.KafkaCli,
	},
}
