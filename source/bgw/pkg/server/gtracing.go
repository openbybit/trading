package server

import (
	"context"
	"fmt"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gcore/nets"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gtrace"
	"github.com/opentracing/opentracing-go"
	"github.com/valyala/fasthttp"

	"bgw/pkg/config"
)

func initTracing() {
	gtrace.SetBuildTagsFunc(buildTracingTags)
	gtrace.SetContextFunc(extractCtx, injectCtx)

	r := &config.Global.Tracing
	if r == nil {
		glog.Errorf(context.TODO(), "no tracing config")
		return
	}
	serviceName := config.AppCfg().Name
	if serviceName == "" {
		serviceName = "bgw"
	}

	samplerType := r.GetOptions("sampler_type", "")
	if samplerType == "" {
		samplerType = gtrace.SamplerTypeRemote
	}

	var samplingServerURL string
	url := r.GetOptions("sampling_server_url", "")
	if url != "" {
		samplingServerURL = fmt.Sprintf("%s?service=%s&env=%s", url, serviceName, env.EnvName())
	}

	conf := gtrace.Config{
		Address:           r.Address,
		ServiceName:       serviceName,
		SamplerType:       samplerType,
		SamplingServerURL: samplingServerURL,
	}
	if err := gtrace.Init(&conf); err != nil {
		galert.Error(context.Background(), "init jaeger failed "+err.Error())
	}
}

func extractCtx(ctx context.Context) context.Context {
	const traceCtxKey = "trace-context"
	if c, ok := ctx.(*fasthttp.RequestCtx); ok {
		if val, ok := c.UserValue(traceCtxKey).(context.Context); ok && val != nil {
			return val
		}
		return context.Background()
	}
	return ctx
}

func injectCtx(oldCtx context.Context, newCtx context.Context) context.Context {
	const traceCtxKey = "trace-context"
	if c, ok := oldCtx.(*fasthttp.RequestCtx); ok {
		c.SetUserValue(traceCtxKey, newCtx)
	}
	return newCtx
}

func buildTracingTags(ctx context.Context) opentracing.Tags {
	return opentracing.Tags{
		"ip": nets.GetLocalIP(),
	}
}
