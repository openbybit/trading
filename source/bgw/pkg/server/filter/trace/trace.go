package trace

import (
	"fmt"

	"code.bydev.io/fbu/gateway/gway.git/gtrace"

	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"

	"github.com/opentracing/opentracing-go"
)

func Init() {
	filter.Register(filter.TracingFilterKey, new)
}

type trace struct {
}

// new create trace filter.
func new() filter.Filter {
	return &trace{}
}

// GetName returns the name of the filter
func (*trace) GetName() string {
	return filter.TracingFilterKey
}

// Do will call next handler
func (t *trace) Do(next types.Handler) types.Handler {
	return func(c *types.Ctx) error {
		carrier := newCarrier(&c.Request.Header)
		clientContext, _ := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)

		span, _ := gtrace.Begin(c, fmt.Sprintf("%s:%s", string(c.Method()), string(c.Path())), opentracing.ChildOf(clientContext))
		defer gtrace.Finish(span)

		metadata.ContextWithTraceId(c, gtrace.TraceIDFromSpan(span))

		return next(c)
	}
}
