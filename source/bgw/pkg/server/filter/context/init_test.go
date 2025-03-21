package context

import (
	"context"
	"testing"

	"bgw/pkg/server/filter"
	"github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {

	convey.Convey("TestInit", t, func() {
		Init()
		f, err := filter.GetFilter(context.Background(), filter.ContextFilterKey)
		convey.So(err, convey.ShouldBeNil)
		convey.So(f, convey.ShouldNotBeNil)

		f, err = filter.GetFilter(context.Background(), filter.ContextFilterKeyGlobal)
		convey.So(err, convey.ShouldBeNil)
		convey.So(f, convey.ShouldNotBeNil)
	})

}
