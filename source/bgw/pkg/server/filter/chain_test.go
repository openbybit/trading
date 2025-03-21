package filter

import (
	"github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestChain_AppendNames(t *testing.T) {
	convey.Convey("TestChain_AppendNames", t, func() {
		chain := NewChain()
		convey.So(chain, convey.ShouldNotBeNil)
		_, err := chain.AppendNames("test_not_exist")
		convey.So(err, convey.ShouldNotBeNil)

		filterName := "test_route_filter"
		Register(filterName, newTestFilterWithoutTag())
		chain, err = chain.AppendNames(filterName)
		convey.So(err, convey.ShouldBeNil)
		convey.So(chain, convey.ShouldNotBeNil)
	})
}

func TestGlobalChain(t *testing.T) {
	convey.Convey("TestGlobalChain", t, func() {
		//:todo should fix AppendNames bug,it's not allowed to return nil when GetFilter return error

		chain := GlobalChain()
		convey.So(chain, convey.ShouldBeNil)
		//convey.So(chain.filters, convey.ShouldNotBeEmpty)
	})
}

func TestChain_Extend(t *testing.T) {
	convey.Convey("TestChain_Extend", t, func() {
		chain := NewChain()
		convey.So(chain, convey.ShouldNotBeNil)
		extendChain := NewChain()
		chain = chain.Extend(*extendChain)
		convey.So(chain, convey.ShouldNotBeNil)
	})
}
