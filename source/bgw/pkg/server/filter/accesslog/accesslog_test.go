package accesslog

import (
	"context"
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/tj/assert"
	"github.com/valyala/fasthttp"

	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
)

var errMock = errors.New("mock err")

func TestAccessLog(t *testing.T) {
	Init()
	a := assert.New(t)

	al := newLogger()
	a.NotNil(al)
	al.Info(context.Background(), "foo\t")
	_ = al.Sync()
}

func TestAccesslog_GetName(t *testing.T) {
	Convey("test get name", t, func() {
		n := newAccesslog().GetName()
		So(n, ShouldEqual, filter.AccessLogFilterKey)
	})
}

func TestAccesslog_Do(t *testing.T) {
	Convey("test accesslog do", t, func() {
		next := func(ctx *types.Ctx) error {
			return nil
		}
		errNext := func(ctx *types.Ctx) error {
			return errMock
		}

		ctx := &fasthttp.RequestCtx{}
		al := newAccesslog()

		Convey("test info logger", func() {
			h := al.Do(next)
			err := h(ctx)
			So(err, ShouldBeNil)
		})

		Convey("test err logger", func() {
			h := al.Do(errNext)
			err := h(ctx)
			So(err, ShouldNotBeNil)
		})

	})
}
