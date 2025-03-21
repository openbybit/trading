package service

import (
	"context"
	"fmt"
	"path/filepath"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/filesystem"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gtrace"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/config"
)

const DynamicLogKey = "bgw-dynamic-log-key"

type CtxDynamicLogKey struct{}

func InitLogger() {
	conf := &glog.Config{
		File:       "data/logs/bgw.log",
		Type:       "lumberjack",
		Format:     "",
		MaxSize:    100,
		MaxAge:     30,
		MaxBackups: 3,
	}

	level := glog.DebugLevel
	if config.AppCfg().Mode != "debug" {
		level = glog.InfoLevel
	}

	logCfg := config.Global.Log.BgwLog
	convertCfg(conf, logCfg)

	if conf.File != "" {
		// lumberjack.Logger默认创建的文件夹权限是0744, 其他用户没有进入目录的权限
		// 这里预先创建文件夹，设置权限为0755
		if err := filesystem.MkdirAll(filepath.Dir(conf.File)); err != nil {
			fmt.Printf("can't make directories for new bgw logfile: fileName:%s, err:%s", conf.File, err)
		}
	}

	cluster := config.AppCfg().Cluster
	if cluster != "" {
		conf.Fields = append(conf.Fields, glog.String("cluster", cluster))
	}

	conf.Level = level
	conf.ContextFn = traceID
	globalLogger := glog.New(conf)
	glog.SetLogger(globalLogger)
}

func traceID(ctx context.Context, fs []glog.Field) []glog.Field {
	span := gtrace.SpanFromContext(ctx)
	tid := gtrace.TraceIDFromSpan(span)
	if tid != "" {
		fs = append(fs, glog.String("trace_id", tid))
	}

	c, ok := ctx.(*types.Ctx)
	if ok {
		akmTraceID := cast.UnsafeBytesToString(c.Request.Header.Peek(constant.XAKMTraceID))
		fs = append(fs, glog.String("akm_id", akmTraceID))
	}

	return fs
}

func DynamicLog(ctx context.Context, msg string, fields ...glog.Field) {
	if DynamicFromCtx(ctx) {
		fields = append(fields, glog.Bool("[dynamic log]", true))
		glog.Info(ctx, msg, fields...)
	} else {
		fields = append(fields, glog.Bool("[dynamic log]", false))
		glog.Debug(ctx, msg, fields...)
	}
}

func DynamicFromCtx(ctx context.Context) bool {
	var v interface{}
	if c, ok := ctx.(*types.Ctx); ok {
		v = c.Value(DynamicLogKey)
	} else {
		v = ctx.Value(CtxDynamicLogKey{})
	}

	if v != nil {
		if res, ok := v.(bool); ok {
			return res
		}
	}
	return false
}

// todo
func convertCfg(conf *glog.Config, cfg config.LogCfg) {
	if cfg.Type != "" {
		conf.Type = cfg.Type
	}
	if cfg.Format != "" {
		conf.Format = cfg.Format
	}
	if cfg.File != "" {
		conf.File = cfg.File
	}
	if cfg.MaxSize != 0 {
		conf.MaxSize = cfg.MaxSize
	}
	if cfg.MaxAge != 0 {
		conf.MaxAge = cfg.MaxAge
	}
	if cfg.MaxBackups != 0 {
		conf.MaxBackups = cfg.MaxBackups
	}
}
