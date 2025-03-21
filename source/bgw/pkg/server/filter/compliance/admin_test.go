package compliance

import (
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gapp"
	"code.bydev.io/fbu/gateway/gway.git/gcompliance"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRegisterAdmin(t *testing.T) {
	Convey("test register admin", t, func() {
		registerAdmin()
	})
}

func TestAdmin(t *testing.T) {
	Convey("test admin", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockWall := gcompliance.NewMockWall(ctrl)
		mockWall.EXPECT().GetUserInfo(gomock.Any(), gomock.Any()).Return(gcompliance.UserInfo{}, nil)
		mockWall.EXPECT().GetStrategy(gomock.Any(), gomock.Any()).Return(nil)
		mockWall.EXPECT().RemoveUserInfo(gomock.Any(), gomock.Any())
		mockWall.EXPECT().QuerySiteConfig(gomock.Any()).Return(nil)

		gw = mockWall
		registerAdmin()
		args := gapp.AdminArgs{
			Params: []string{"123"},
		}

		_, err := OnGetUserInfo(args)
		So(err, ShouldBeNil)

		_, err = OnGetStrategy(args)
		So(err, ShouldBeNil)

		_, err = OnRemoveUserInfo(args)
		So(err, ShouldBeNil)

		_, err = OnQuerySiteConfig(args)
		So(err, ShouldBeNil)
	})
}
