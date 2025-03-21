package etcd

import (
	"context"
	"errors"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/getcd"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
)

func TestNewConfigClient(t *testing.T) {
	convey.Convey("TestNewConfigClient", t, func() {
		ec, err := NewConfigClient(context.TODO())
		convey.So(err, convey.ShouldBeNil)
		convey.So(ec, convey.ShouldNotBeNil)

		convey.Convey("TestNewConfigClient when password is not empty,but not init sechub", func() {
			_, err := NewConfigClient(context.Background())
			convey.So(err, convey.ShouldBeNil)
		})

		convey.Convey("TestNewConfigClient when NewClient return error", func() {
			applyFunc := gomonkey.ApplyFunc(getcd.NewClient, func(ctx context.Context, opts ...getcd.Option) (getcd.Client, error) {
				return nil, errors.New("test error")
			})
			defer applyFunc.Reset()
			_, err := NewConfigClient(context.TODO())
			convey.So(err, convey.ShouldNotBeNil)
		})

	})
}
