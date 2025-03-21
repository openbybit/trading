package gcompliance

import (
	"context"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cache/lru"
)

func TestWall_GetUserInfo(t *testing.T) {
	Convey("test GetUserInfo", t, func() {
		uis, _ := lru.NewLRU(20)
		uis.Add(int64(123), nil)
		uis.Add(int64(1245), UserInfo{Country: "CHN"})

		w := &wall{
			userInfos: uis,
		}

		res, err := w.GetUserInfo(context.Background(), 1245)
		So(err, ShouldBeNil)
		So(res.Country, ShouldEqual, "CHN")

		_, err = w.GetUserInfo(context.Background(), 123)
		So(err, ShouldNotBeNil)

		_, err = w.GetUserInfo(context.Background(), 125)
		So(err, ShouldNotBeNil)

	})
}
