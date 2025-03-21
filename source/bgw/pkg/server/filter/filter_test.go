package filter

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/pkg/errors"
	"github.com/smartystreets/goconvey/convey"
	"github.com/tj/assert"
	"github.com/valyala/fasthttp"

	"bgw/pkg/common/types"
)

type testFilter struct {
	tag string
}

func newTestFilter(tag string) Filter {
	return &testFilter{tag: tag}
}

func newTestFilterWithoutTag() Filter {
	return newTestFilter("")
}

func (t *testFilter) Init(ctx context.Context, args ...string) error {
	return nil
}

func (t *testFilter) GetName() string {
	return "test_filter"
}

func (t *testFilter) Do(next types.Handler) types.Handler {
	return func(c *types.Ctx) error { // handler
		err := next(c) // next stack
		fmt.Printf("call next @%s, err: %v\n", t.tag, err)
		return err
	}
}

type globalFilter struct {
	tag string
}

func TestChain(t *testing.T) {
	a := assert.New(t)

	t1 := newTestFilter("t1")
	t2 := newTestFilter("t2")
	t3 := newTestFilter("t3")

	var h types.Handler = func(_ *types.Ctx) error {
		fmt.Println("finally invoked")
		return nil
	}

	chained := NewChain(t1, t2, t3).Finally(h)
	err := chained(nil)
	a.Nil(err)
}

func TestChainAppend(t *testing.T) {
	convey.Convey("TestChainAppend", t, func() {
		t1 := newTestFilter("t1")
		t2 := newTestFilter("t2")
		t3 := newTestFilter("t3")
		chain := NewChain()

		n := chain.Append(t1, t2, t3)
		convey.ShouldBeTrue(len(n.filters) == 1)

		convey.Convey("TestChainInvoke", func() {
			var h types.Handler = func(_ *types.Ctx) error {
				fmt.Println("finally invoked")
				return nil
			}

			err := n.Finally(h)(nil)
			convey.ShouldBeNil(err)
		})

		convey.Convey("TestChainInvokeError", func() {
			ferr := errors.New("finally error")
			var h types.Handler = func(_ *types.Ctx) error {
				fmt.Println("finally invoked")
				return ferr
			}

			err := n.Finally(h)(nil)
			convey.ShouldBeError(err, ferr)
		})
	})
}

func ExampleChain() {
	t1 := newTestFilter("t1")
	t2 := newTestFilter("t2")
	chain := NewChain()
	n := chain.Append(t1, t2)
	var h types.Handler = func(_ *types.Ctx) error {
		fmt.Println("finally invoked")
		return nil
	}
	_ = n.Finally(h)(nil)
	// Output:
	// finally invoked
	// call next @t2, err: <nil>
}

func TestGetFilter(t *testing.T) {
	convey.Convey("TestGetFilter", t, func() {
		filterName := "test_filter"
		_, err := GetFilter(context.Background(), filterName)
		convey.So(err, convey.ShouldBeError)

		Register(filterName, newTestFilter)
		filter, err := GetFilter(context.Background(), filterName)
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(filter, convey.ShouldBeNil)

		filterName = "test_route_filter"
		Register(filterName, newTestFilterWithoutTag)
		filter, err = GetFilter(context.Background(), filterName)
		convey.So(err, convey.ShouldBeNil)
		convey.So(filter, convey.ShouldNotBeNil)

		Register(filterName, newTestFilterWithoutTag())
		filter, err = GetFilter(context.Background(), filterName)
		convey.So(err, convey.ShouldBeNil)
		convey.So(filter, convey.ShouldNotBeNil)

	})
}

func TestFuncDo(t *testing.T) {
	convey.Convey("test func do", t, func() {
		var f Func = func(next types.Handler) types.Handler {
			log.Println("func do")
			return next
		}

		next := func(*fasthttp.RequestCtx) error {
			return nil
		}

		h := f.Do(next)
		convey.So(h, convey.ShouldNotBeNil)
	})
}
