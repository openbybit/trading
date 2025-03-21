package limiter

import (
	"context"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tj/assert"
	"golang.org/x/time/rate"

	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
)

func TestLimiterReserve(t *testing.T) {
	a := assert.New(t)

	limit := rate.NewLimiter(10, 10)
	a.NotNil(limit)

	reservation := limit.Reserve()
	t.Log(reservation.OK())
	t.Log(reservation.Delay())
}

var rateLimiter = rate.NewLimiter(10, 10)

func BenchmarkXrate(b *testing.B) {
	succ, false := 0, 0
	b.SetParallelism(32)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			b := rateLimiter.Allow()
			if b {
				succ++
			} else {
				false++
			}
		}
	})
}

func TestOnEvent(t *testing.T) {
	aa := newGlobalLimiter().(*globalLimiter)

	assert.Equal(t, nil, aa.GetEventType())
	assert.Equal(t, filter.QPSRateLimitFilterKeyGlobal, aa.GetName())
	assert.Equal(t, 0, aa.GetPriority())

	err := aa.OnEvent(observer.DefaultEvent{})
	assert.NoError(t, err)

	err = aa.OnEvent(observer.DefaultEvent{
		Value: "ass",
	})
	assert.NoError(t, err)
}

func TestLimiter(t *testing.T) {
	Convey("test limiter", t, func() {
		l := new().(*limiter)
		So(l.GetName(), ShouldEqual, filter.QPSRateLimitFilterKey)

		err := l.Init(context.Background())
		So(err, ShouldBeNil)

		err = l.Init(context.Background(), "route", "--wrongArgs=99")
		So(err, ShouldNotBeNil)

		err = l.Init(context.Background(), "route", "--rate=1")
		So(err, ShouldBeNil)

		next := func(ctx *types.Ctx) error { return nil }
		handler := l.Do(next)

		ctx := &types.Ctx{}
		err = handler(ctx)
		So(err, ShouldBeNil)

		err = handler(ctx)
		So(err, ShouldNotBeNil)
	})
}
