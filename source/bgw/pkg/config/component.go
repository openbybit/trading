package config

import (
	"context"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/frameworks/byone/core/conf"
	"code.bydev.io/frameworks/byone/core/discov/nacos"
	"code.bydev.io/frameworks/byone/kafka"
	"code.bydev.io/frameworks/byone/zrpc"
)

var (
	compCfgFile = "conf/component.toml"
)

func initComponentCfg() {
	cfg := new(ComponentConfig)
	conf.MustLoad(compCfgFile, cfg)
	glog.Debug(context.Background(), "component cfg", glog.Any("cfg", *cfg))
	Global.ComponentConfig = *cfg
}

type ComponentConfig struct {
	Nacos nacos.NacosConf `json:",optional"`
	// 合规墙
	Compliance         zrpc.RpcClientConf          `json:",optional"`
	ComplianceKafkaCli kafka.UniversalClientConfig `json:",optional"`

	// smp
	Smp         RemoteConfig                `json:",optional"`
	SmpKafkaCli kafka.UniversalClientConfig `json:",optional"`

	// user
	UserServicePrivate zrpc.RpcClientConf `json:",optional"`
	User               RemoteConfig       `json:",optional"`

	Masq              zrpc.RpcClientConf `json:",optional"`
	Mixer             zrpc.RpcClientConf `json:",optional"`
	BanServicePrivate zrpc.RpcClientConf `json:",optional"`
	UtaRouter         zrpc.RpcClientConf `json:",optional"`
	UtaRouterDa       zrpc.RpcClientConf `json:",optional"`
	Oauth             zrpc.RpcClientConf `json:",optional"`

	OpenInterest RemoteConfig `json:",optional"`
	Bsp          RemoteConfig `json:",optional"`
}
