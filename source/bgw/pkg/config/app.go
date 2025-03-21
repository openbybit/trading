package config

import (
	"context"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/frameworks/byone/core/conf"
)

var (
	appCfgFile = "conf/app.toml"
)

func initAppCfg() {
	cfg := new(AppConfig)
	conf.MustLoad(appCfgFile, cfg)
	glog.Debug(context.Background(), "app cfg", glog.Any("cfg", *cfg))
	Global.AppConfig = *cfg
}

type AppConfig struct {
	App    App
	Server Server
	Data   Data
	Log    Log
}

type App struct {
	Name            string
	Mode            string `json:",default=release,options=release|debug"`
	Cluster         string `json:",optional"`
	Namespace       string `json:",optional"` // 测试环境拿环境变量，testnet主网拿配置
	Group           string
	QpsRate         int
	UpstreamQpsRate int
	Pprof           bool `json:",default=true"`
	BatWing         int  `json:",default=6480"`
	NoHealthBlock   bool `json:",optional"`
	RedisDowngrade  bool `json:",default=false"`
}

type Server struct {
	Http ServerConfig `json:",optional"`
	Ws   ServerConfig `json:",optional"`
}

type Data struct {
	Geo       string    `json:",default=data/geoip"`
	CacheSize CacheSize `json:",optional"`
}

// CacheSize Mb
type CacheSize struct {
	AccountCacheSize       int `json:",optional"`
	CopyTradeCacheSize     int `json:",optional"`
	OpenapiCacheSize       int `json:",optional"`
	BizLimitQuotaCacheSize int `json:",optional"`
	BanCacheSize           int `json:",optional"`
}

type Log struct {
	BgwLog    LogCfg
	AccessLog LogCfg
}

type LogCfg struct {
	Type       string `json:",default=lumberjack,options=lumberjack|stdout"`
	Format     string `json:",optional"` // json
	File       string
	MaxSize    int // MB
	MaxAge     int // MB
	MaxBackups int
}
