package ban

import (
	"fmt"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gapp"
	"github.com/coocood/freecache"
	jsoniter "github.com/json-iterator/go"
	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/protobuf/proto"
)

func Test_registerAdmin(t *testing.T) {
	Convey("test register admin", t, func() {
		registerAdmin()
		defaultBanService = &banService{}
		registerAdmin()
	})
}

func TestBanAdmin(t *testing.T) {
	Convey("test OnQueryBanCache", t, func() {
		cache := freecache.NewCache(100 * 1024)
		defaultBanService = &banService{
			cache: cache,
		}
		// test OnQueryBanCache
		// empty cache
		args := gapp.AdminArgs{
			Params: []string{"234"},
		}
		res, err := OnQueryBanCache(args)
		So(err, ShouldBeNil)
		So(res.(string), ShouldEqual, "empty ban status")

		// cache with no userState
		key := fmt.Sprintf("%d_banned_info", 234)
		usi := &userStatusInternal{}
		val, _ := jsoniter.Marshal(usi)
		_ = cache.Set([]byte(key), val, 1000)

		res, err = OnQueryBanCache(args)
		So(err, ShouldBeNil)
		So(res, ShouldBeNil)

		// cache with wrong userState
		usi.UserState = []byte("wrong")
		val, _ = jsoniter.Marshal(usi)
		_ = cache.Set([]byte(key), val, 1000)

		res, err = OnQueryBanCache(args)
		So(err, ShouldNotBeNil)
		So(res, ShouldBeNil)

		// useful cache
		us := &UserStatus{
			IsNormal: true,
		}

		d, _ := proto.Marshal(us)
		usi.UserState = d
		val, _ = jsoniter.Marshal(usi)
		_ = cache.Set([]byte(key), val, 1000)

		res, err = OnQueryBanCache(args)
		So(err, ShouldBeNil)
		So(res, ShouldNotBeNil)

		// test OnQueryBan
		res, err = OnQueryBan(args)
		So(err, ShouldBeNil)
		So(res, ShouldNotBeNil)

		// test OnDeleteBanCache
		res, err = OnDeleteBanCache(args)
		So(err, ShouldBeNil)
		So(res.(string), ShouldEqual, "success")
	})
}
