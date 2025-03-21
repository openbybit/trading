package openapi

import (
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gapp"
	"git.bybit.com/svc/stub/pkg/pb/api/user"
	"github.com/coocood/freecache"
	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/protobuf/proto"
)

func Test_registerAdmin(t *testing.T) {
	Convey("test register admin", t, func() {
		registerAdmin()
		defaultOpenapiService = &openapiService{}
		registerAdmin()
	})
}

func Test_admin(t *testing.T) {
	Convey("test admin", t, func() {
		cache := freecache.NewCache(100 * 1024)
		defaultOpenapiService = &openapiService{
			cache: cache,
		}

		// test OnQueryAPIkeyCache
		// test empty cache
		args := gapp.AdminArgs{
			Params: []string{"apikey"},
		}
		res, err := OnQueryAPIkeyCache(args)
		So(err, ShouldBeNil)
		So(res.(string), ShouldEqual, "empty apikey")

		// test wrong cache
		_ = defaultOpenapiService.(*openapiService).cache.Set([]byte("apikey"), []byte("wrong"), 1000)
		res, err = OnQueryAPIkeyCache(args)
		So(err, ShouldNotBeNil)
		So(res, ShouldBeNil)

		// test useful cache
		msg := &user.MemberLogin{}
		val, _ := proto.Marshal(msg)
		_ = defaultOpenapiService.(*openapiService).cache.Set([]byte("apikey"), val, 1000)
		res, err = OnQueryAPIkeyCache(args)
		So(err, ShouldBeNil)
		So(res, ShouldNotBeNil)

		// test OnQueryAPIkey
		args2 := gapp.AdminArgs{
			Options: map[string]string{"apikey": "apikey", "xoriginfrom": ""},
		}
		res, err = OnQueryAPIkey(args2)
		So(err, ShouldBeNil)
		So(res, ShouldNotBeNil)

		// test OnDeleteAPIkeyCache
		res, err = OnDeleteAPIkeyCache(args)
		So(err, ShouldBeNil)
		So(res.(string), ShouldEqual, "success")
	})
}
