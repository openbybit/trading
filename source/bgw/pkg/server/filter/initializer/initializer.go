package initializer

import (
	"bgw/pkg/common/types"
	"bgw/pkg/config"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service"
	"bgw/pkg/service/gray"
)

var (
	logGrayer = gray.NewGrayer("log", config.GetHTTPServerConfig().ServiceRegistry.ServiceName)
)

type initializer struct {
	routeKey metadata.RouteKey
	service  string
}

// New create a new initializer
func New(routeKey metadata.RouteKey, service string) filter.Filter {
	return &initializer{routeKey: routeKey, service: service}
}

// GetName returns the name of the filter
func (i *initializer) GetName() string {
	return filter.InitFilterKey
}

// Do implements the filter interface
func (i *initializer) Do(next types.Handler) types.Handler {
	return func(c *types.Ctx) error {
		// set context with route
		// !NOTE: this is important for limiters, as first arg
		md := metadata.MDFromContext(c)
		md.Route = i.routeKey
		md.InvokeService = i.service
		md.WithContext(c)

		if ok, _ := logGrayer.GrayStatus(c); ok {
			c.SetUserValue(service.DynamicLogKey, true)
		}

		return next(c)
	}
}
