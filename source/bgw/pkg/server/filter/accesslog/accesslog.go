package accesslog

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/filesystem"
	"code.bydev.io/fbu/gateway/gway.git/gcore/nets"
	"code.bydev.io/fbu/gateway/gway.git/glog"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/config"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
)

func Init() {
	filter.Register(filter.AccessLogFilterKey, newAccesslog())
}

type accesslog struct {
	logger glog.Logger
}

func newAccesslog() filter.Filter {
	return &accesslog{logger: newLogger()}
}

func newLogger() glog.Logger {
	ac := &glog.Config{
		File:          "data/logs/access.log",
		Type:          "lumberjack",
		MaxSize:       100,
		MaxBackups:    3,
		MaxAge:        30,
		Level:         glog.InfoLevel,
		DisableCaller: true,
	}

	// ignore error, use default
	convertCfg(ac, config.Global.Log.AccessLog)
	if ac.File != "" {
		// lumberjack.Logger默认创建的文件夹权限是0744, 其他用户没有进入目录的权限
		// 这里预先创建文件夹，设置权限为0755
		if err := filesystem.MkdirAll(filepath.Dir(ac.File)); err != nil {
			fmt.Printf("can't make directories for new access logfile: fileName:%s, err:%s", ac.File, err)
		}
	}

	return glog.New(ac).Named(constant.GetAppName())
}

func (l *accesslog) GetName() string {
	return filter.AccessLogFilterKey
}

const DynamicLogKey = "bgw-dynamic-log-key"

func (l *accesslog) Do(next types.Handler) types.Handler {
	return func(c *types.Ctx) (err error) {

		reqTime := time.Now()

		// real call
		err = next(c)

		data := l.build(c, reqTime, err)
		if err != nil {
			l.logger.Error(c, data)
		} else {
			l.logger.Info(c, data)
		}

		return
	}
}

// request time, code, trace id, host, path, remote ip, referer, user agent, device id, guid
// requirement: https://confluence.bybit.com/pages/viewpage.action?pageId=31897371
func (l *accesslog) build(ctx *types.Ctx, reqTime time.Time, err error) string {
	latency := time.Since(reqTime)
	builder := strings.Builder{}
	md := metadata.MDFromContext(ctx)

	status := "0"
	if err != nil {
		status = cast.ToString(berror.GetErrCode(err))
	} else {
		// observe upstream error code
		codes := metadata.CodeFromUpstreamContext(ctx)
		if len(codes) > 0 {
			status = cast.ToString(codes[0])
		}
	}

	// http code
	builder.WriteString(cast.ToString(ctx.Response.StatusCode()))
	builder.WriteString("\t")

	// BGW internal status code and upstream codes
	builder.WriteString(status)
	builder.WriteString("\t")

	// trace id
	builder.WriteString(md.TraceID)
	builder.WriteString("\t")

	// latency
	builder.WriteString(latency.String())
	builder.WriteString("\t")

	// request host
	builder.WriteString(md.Extension.Host)
	builder.WriteString("\t")

	// remote ip
	builder.WriteString(md.Extension.RemoteIP)
	builder.WriteString("\t")

	// user id if authenticated
	builder.WriteString(cast.ToString(md.UID))
	builder.WriteString("\t")

	// account id if authenticated
	builder.WriteString(cast.ToString(md.AccountID))
	builder.WriteString("\t")

	// user agent
	builder.WriteString(md.Extension.UserAgent)
	builder.WriteString("\t")

	// referer
	builder.WriteString(md.Extension.Referer)
	builder.WriteString("\t")

	// request http method
	builder.WriteString(md.Method)
	builder.WriteString("\t")

	// request uri
	builder.WriteString(md.Path)
	builder.WriteString("\t")

	// client platform
	builder.WriteString(md.Extension.Platform)
	builder.WriteString("\t")

	// request fingerprint
	builder.WriteString(md.Extension.Fingerprint)
	builder.WriteString("\t")

	// bgw addr
	builder.WriteString(nets.GetLocalIP())
	builder.WriteString("\t")

	// invoke server
	builder.WriteString(md.InvokeService)
	builder.WriteString("\t")

	// invoke instance address and port
	builder.WriteString(md.InvokeAddr)
	builder.WriteString("\t")

	// invoke latency
	builder.WriteString(md.ReqCost.String())
	builder.WriteString("\t")

	// aws trace id
	builder.WriteString(md.AKMTrace)
	builder.WriteString("\t")

	// client op from
	builder.WriteString(md.Extension.OpFrom)
	builder.WriteString("\t")

	// api key
	builder.WriteString(md.APIKey)
	builder.WriteString("\t")

	// invoke
	if md.InvokeNamespace != "" || md.InvokeGroup != "" {
		info := md.InvokeNamespace + "-" + md.InvokeGroup
		builder.WriteString(info)
	}
	builder.WriteString("\t")

	// biz tags
	builder.WriteString(buildTags(ctx, md))

	return builder.String()
}

func buildTags(ctx *types.Ctx, md *metadata.Metadata) string {
	tags := strings.Builder{}
	tags.WriteString(md.Route.GetAppName(ctx))
	tags.WriteString(":")
	tags.WriteString(md.MemberRelation)
	tags.WriteString(":")
	tags.WriteString(md.ParentUID)
	return tags.String()
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
