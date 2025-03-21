package user

import (
	"fmt"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gapp"
	"github.com/coocood/freecache"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_registerAdmin(t *testing.T) {
	Convey("test register admin", t, func() {
		registerAccountAdmin()
		registerCopytradeAdmin()

		accountService = &AccountService{}
		copyTradeService = &CopyTradeService{}

		registerAccountAdmin()
		registerCopytradeAdmin()
	})
}

func TestAccountAdmin(t *testing.T) {
	Convey("test account admin", t, func() {
		cache := freecache.NewCache(100 * 1024)
		accountService = &AccountService{
			accountCache: cache,
		}
		// test OnQueryMembertagCache
		// test empty cache
		args := gapp.AdminArgs{
			Options: map[string]string{"uid": "1234", "tag": "ut_tag"},
		}

		res, err := OnQueryMembertagCache(args)
		So(err, ShouldBeNil)
		So(res.(string), ShouldEqual, "empty tag")

		// test useful cache
		key := fmt.Sprintf("%dmember_tag_%s", 1234, "ut_tag")
		_ = accountService.accountCache.Set([]byte(key), []byte("tag1"), 1000)

		res, err = OnQueryMembertagCache(args)
		So(err, ShouldBeNil)
		So(res.(string), ShouldEqual, "tag1")

		// test OnQueryMembertag
		res, err = OnQueryMembertag(args)
		So(err, ShouldBeNil)
		So(res.(string), ShouldEqual, "tag1")

		// test OnDeleteMembertagCache
		res, err = OnDeleteMembertagCache(args)
		So(err, ShouldBeNil)
		So(res.(string), ShouldEqual, "success")

		// test OnQueryAccountID
		args2 := gapp.AdminArgs{
			Options: map[string]string{"uid": "1234", "accountType": "normal", "bizType": "futures"},
		}
		res, err = OnQueryAccountID(args2)
		So(err, ShouldNotBeNil)

		// test OnDeleteAccountID
		res, err = OnDeleteAccountID(args2)
		So(err, ShouldBeNil)
		So(res.(string), ShouldEqual, "success")

		args3 := gapp.AdminArgs{
			Options: map[string]string{"tag": "tag1", "val": "val1"},
		}
		res, err = OnSetMembertag(args3)
		So(err, ShouldBeNil)
		So(res, ShouldBeNil)
	})
}

func TestCopytradeAdmin(t *testing.T) {
	Convey("test copytrade admin", t, func() {
		cache := freecache.NewCache(100 * 1024)
		copyTradeService = &CopyTradeService{
			copytradeCache: cache,
		}

		args := gapp.AdminArgs{
			Options: map[string]string{"uid": "1234"},
		}

		res, err := OnQueryCopytradeData(args)
		So(err, ShouldBeNil)
		So(res, ShouldBeNil)

		res, err = OnDeleteCopytradeData(args)
		So(err, ShouldBeNil)
		So(res.(string), ShouldEqual, "success")
	})
}
