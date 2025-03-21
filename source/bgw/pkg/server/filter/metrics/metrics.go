package metrics

import (
	"net/http"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
)

func Init() {
	filter.Register(filter.MetricsFilterKey, newMetrics())
}

func newMetrics() filter.Filter {
	return &metricsFilter{}
}

type metricsFilter struct {
}

func (m *metricsFilter) GetName() string {
	return filter.MetricsFilterKey
}

func (m *metricsFilter) Do(next types.Handler) types.Handler {
	return func(c *types.Ctx) (err error) {
		now := time.Now()

		err = next(c)

		md := metadata.MDFromContext(c)

		if statusCode := c.Response.StatusCode(); statusCode == http.StatusOK {
			total := time.Since(now)                                                        // total
			upstream := md.ReqCost                                                          // invoke upstream
			internal := total - upstream                                                    // bgw filters
			tgw := time.Duration(md.ReqInitTime.UnixNano() - cast.AtoInt64(md.ReqInitAtE9)) // tgw -> bgw

			path := md.GetStaticRoutePath()

			gmetric.ObserveHTTPLatency(total, "total", path, md.Method, md.Route.Registry, md.Intermediate.CallOrigin)
			gmetric.ObserveHTTPLatency(upstream, "upstream", path, md.Method, md.Route.Registry, md.Intermediate.CallOrigin)
			gmetric.ObserveHTTPLatency(internal, "internal", path, md.Method, md.Route.Registry, md.Intermediate.CallOrigin)
			gmetric.ObserveHTTPLatency(tgw, "tgw", path, md.Method, md.Route.Registry, md.Intermediate.CallOrigin)
			if err != nil {
				gmetric.IncHTTPCounter("biz_err", cast.ToString(berror.GetErrCode(err)), path, md.Method, md.Route.Registry)
			}
		} else {
			gmetric.IncDefaultCounter("http_err", cast.ToString(statusCode))
		}

		metadata.Release(md)

		return nil
	}
}
