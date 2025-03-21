package config

import (
	"context"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/frameworks/byone/core/conf"
	"code.bydev.io/frameworks/byone/core/stores/redis"
	"code.bydev.io/frameworks/byone/kafka"
)

var (
	midCfgFile = "conf/middleware.toml"
)

func initMiddlewareCfg() {
	cfg := new(MiddlewareConfig)
	conf.MustLoad(midCfgFile, cfg)
	glog.Debug(context.Background(), "middleware cfg", glog.Any("cfg", *cfg))
	Global.MiddlewareConfig = *cfg
}

type MiddlewareConfig struct {
	Etcd       RemoteConfig
	S3         RemoteConfig
	Geo        RemoteConfig
	Tracing    RemoteConfig
	Alert      RemoteConfig
	WebConsole RemoteConfig
	SechubCfg  RemoteConfig // SechubCfg为了解决和byone sechub冲突
	NacosCfg   RemoteConfig // NacosCfg防止和byone nacos冲突
	Kafka      RemoteConfig

	Redis    redis.RedisConf             `json:",optional"`
	KafkaCli kafka.UniversalClientConfig `json:",optional"`
}
