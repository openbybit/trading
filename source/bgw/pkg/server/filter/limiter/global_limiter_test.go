package limiter

import (
	"context"
	"reflect"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/time/rate"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/filter/limiter/manual_intervent"
)

func TestGlobalLimiter(t *testing.T) {
	Convey("test init", t, func() {
		Init()
		gmetric.Init("bgw")
		g := newGlobalLimiter().(*globalLimiter)
		So(g.GetName(), ShouldEqual, filter.QPSRateLimitFilterKeyGlobal)

		err := g.Init(context.Background())
		So(err, ShouldBeNil)

		next := func(*types.Ctx) error { return nil }
		handler := g.Do(next)
		ctx := &types.Ctx{}
		err = handler(ctx)
		So(err, ShouldBeNil)

		method := gomonkey.ApplyPrivateMethod(reflect.TypeOf(g), "getRule", func(l *globalLimiter, ctx *types.Ctx) *rate.Limiter {
			return rate.NewLimiter(0, 0)
		})
		defer method.Reset()
		err = handler(ctx)
		So(err, ShouldNotBeNil)

		interveneMethod := gomonkey.ApplyMethod(reflect.TypeOf(g.interveneLimiter), "Intervene", func(*manual_intervent.InterveneLimiter, *types.Ctx) bool {
			return true
		})
		err = handler(ctx)
		So(err, ShouldBeError, berror.ErrVisitsLimit)
		interveneMethod.Reset()

		allowMethod := gomonkey.ApplyFunc((*rate.Limiter).Allow, func(*rate.Limiter) bool {
			return false
		})
		err = handler(ctx)
		So(err, ShouldBeError, berror.ErrVisitsLimit)
		allowMethod.Reset()
	})
}

func TestGlobalLimiter_OnEvent(t *testing.T) {
	Convey("test GlobalLimiter OnEvent", t, func() {
		val := `
qps_limits:
  unify-dev-1: 1000
  unify-test-1: 2000
rate_limit:
  headers:
    pressure_openapi: 25000000
    pressure_brokerapi: 2500000
    pressure: 100000
`

		l := &globalLimiter{
			cluster: "unify-dev-1",
		}

		event := &observer.DefaultEvent{}
		event.Value = val
		err := l.OnEvent(event)
		So(err, ShouldBeNil)

		event.Value = "1234"
		err = l.OnEvent(event)
		So(err, ShouldBeNil)

		val = `
qps_limits:
  unify-dev-1:
rate_limit:
  headers:
    pressure_openapi: 25000000
    pressure_brokerapi: 2500000
    pressure: 100000
`
		event.Value = val
		err = l.OnEvent(event)
		So(err, ShouldBeNil)
	})
}
