package bsp

import (
	"context"
	"errors"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gbsp"
	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
)

func TestNewBsp(t *testing.T) {
	Convey("test bsp", t, func() {
		Init()
		b := newBsp()
		name := b.GetName()
		So(name, ShouldEqual, filter.BspFilterKey)
	})
}

func TestBsp_Init(t *testing.T) {
	Convey("test Bsp init", t, func() {
		b := &bsp{}

		patch := gomonkey.ApplyFunc(getBspPublicKey, func() string { return "" })
		err := b.Init(context.Background())
		So(err, ShouldNotBeNil)
		So(checker, ShouldBeNil)
		patch.Reset()

		// 覆盖率
		_ = getBspPublicKey()

	})
}

func TestBsp_Do(t *testing.T) {
	Convey("test bsp do", t, func() {
		next := func(ctx *types.Ctx) error {
			return nil
		}
		ctx := &types.Ctx{}
		b := newBsp()
		handler := b.Do(next)

		checker = &mockChecker{}
		// check err
		err := handler(ctx)
		So(err, ShouldNotBeNil)

		// check success
		ctx.Request.Header.Set(BspHeaderAuthApp, "success")
		err = handler(ctx)
		So(err, ShouldBeNil)
	})
}

type mockChecker struct{}

func (m *mockChecker) Check(ctx context.Context, auth, timestamp, app []byte) (*gbsp.UserInfo, error) {
	if string(app) == "success" {
		return &gbsp.UserInfo{User: "u"}, nil
	}
	return nil, errors.New("mock err")
}
