package core

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"gopkg.in/yaml.v2"

	"bgw/pkg/breaker"
	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/config"
	"bgw/pkg/config_center"
	"bgw/pkg/config_center/nacos"
	gmetadata "bgw/pkg/server/metadata"
)

type invokeFunc func(ctx *types.Ctx, route *MethodConfig, md *gmetadata.Metadata) (err error)

type invokeMidware interface {
	Do(invokeFunc) invokeFunc
}

func getInvokeMidwares() []invokeMidware {
	return []invokeMidware{newBreakerMidware(breakerMgr)}
}

const (
	file      = "breaker_config.yaml"
	group     = "BGW_GROUP"
	namespace = "bgw"
)

var breakerMgr = breaker.NewBreakerMgr()

type breakerMidware struct {
	nacosCli config_center.Configure
	observer.EmptyListener

	cluster string
	on      atomic.Bool

	breakerMgr breaker.BreakerMgr
}

func newBreakerMidware(bgr breaker.BreakerMgr) invokeMidware {
	bmw := &breakerMidware{
		cluster:    config.GetHTTPServerConfig().ServiceRegistry.ServiceName,
		breakerMgr: bgr,
	}
	bmw.on.Store(true)
	nacosCfg, err := nacos.NewNacosConfigure(
		context.Background(),
		nacos.WithGroup(group),
		nacos.WithNameSpace(namespace),
	)
	if err != nil {
		gmetric.IncDefaultError("breaker", "init_cfg_err")
		galert.Error(context.Background(), fmt.Sprintf("break config new nacos cli error, err = %s", err.Error()))
	} else {
		bmw.nacosCli = nacosCfg
		if err = nacosCfg.Listen(context.Background(), file, bmw); err != nil {
			galert.Error(context.Background(), fmt.Sprintf("break config listen error, err = %s", err.Error()))
			gmetric.IncDefaultError("breaker", "cfg_listen_err")
		}
	}
	return bmw
}

func (b *breakerMidware) Do(next invokeFunc) invokeFunc {
	return func(ctx *types.Ctx, route *MethodConfig, md *gmetadata.Metadata) (err error) {
		if !route.Breaker || !b.on.Load() {
			return next(ctx, route, md)
		}
		serviceName := route.Service().Registry
		target := md.InvokeAddr
		var method string
		if route.Service().Protocol == constant.HttpProtocol {
			method = fmt.Sprintf("%s:%s", md.Method, md.StaticRoutePath)
		} else {
			method = fmt.Sprintf("%s.%s", route.Service().GetFullQulifiedName(), route.Name)
		}

		bkr := b.breakerMgr.GetOrSet(serviceName, target, method)
		promise, berr := bkr.Allow()
		if berr != nil {
			return berror.NewUpStreamErr(berror.InternalErrBreaker, fmt.Sprintf("%s %s breaker err: %s", serviceName, method, berr.Error()))
		}

		defer func() {
			accept := acceptable(err)
			// 针对http服务需要判断status code
			var status int
			if accept && route.Service().Protocol == constant.HttpProtocol {
				source, ok := ctx.UserValue(constant.CtxInvokeResult).(respStatus)
				if ok {
					status = source.GetStatus()
				}
				accept = status < http.StatusInternalServerError
			}
			if accept {
				promise.Accept()
				return
			}
			var reason string
			if status >= http.StatusInternalServerError {
				reason = fmt.Sprintf("http %d", status)
			} else {
				reason = err.Error()
			}
			promise.Reject(reason)
		}()

		return next(ctx, route, md)
	}
}

func (b *breakerMidware) OnEvent(event observer.Event) error {
	e, ok := event.(*observer.DefaultEvent)
	if !ok {
		return nil
	}
	if e.Value == "" {
		return nil
	}
	glog.Info(context.TODO(), "break config OnEvent", glog.String("key", e.Key))

	data := make(map[string]bool)
	if err := yaml.Unmarshal([]byte(e.Value), &data); err != nil {
		galert.Error(context.Background(), fmt.Sprintf("break config unmarshsl failed, err = %s", err.Error()))
		gmetric.IncDefaultError("breaker", "cfg_cb_err")
		return nil
	}

	res, ok := data[b.cluster]
	if !ok {
		return nil
	}
	glog.Info(context.TODO(), "break config update", glog.String("cluster", b.cluster), glog.Bool("val", res))
	b.on.Store(res)
	return nil
}

func acceptable(err error) bool {
	if err == nil {
		return true
	}
	berr, ok := err.(codeErr)
	if !ok {
		return false
	}
	code := berr.GetCode()
	switch code {
	case berror.TimeoutErr, berror.UpstreamErrInvokerBreaker:
		return false
	}
	return true
}

type codeErr interface {
	GetCode() int64
}

type respStatus interface {
	GetStatus() int
}
