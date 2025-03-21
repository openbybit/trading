package ban

import (
	"bgw/pkg/common/berror"
	"bgw/pkg/server/filter"
	"bgw/pkg/service/ban"
	"bgw/pkg/test"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"errors"
	ban2 "git.bybit.com/svc/stub/pkg/pb/api/ban"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	"github.com/smartystreets/goconvey/convey"
	"github.com/valyala/fasthttp"
	"testing"
)

func TestGetName(t *testing.T) {
	convey.Convey("TestGetName", t, func() {
		Init()
		bf := newBanFilter()
		convey.So(bf.GetName(), convey.ShouldEqual, filter.BanFilterKey)
	})
}

func TestDo(t *testing.T) {
	gmetric.Init("TestDo_ban")
	convey.Convey("TestDo", t, func() {
		bf := &banFilter{}
		args := []string{"ban", `--banTags=[{"bizType":"1", "tagName":"2", "tagValue":"3"}]`}
		ctx, _ := test.NewReqCtx()
		err := bf.Init(ctx, args...)
		convey.So(err, convey.ShouldBeNil)
		convey.So(len(bf.br.bts), convey.ShouldEqual, 1)
		convey.So(bf.br.bts, convey.ShouldContainKey, banTag{
			BizType:  "1",
			TagName:  "2",
			TagValue: "3",
		})

		p := gomonkey.ApplyFuncReturn(ban.GetBanService, nil, errors.New("xxx"))
		defer p.Reset()

		// ban svc err
		h := bf.Do(func(rctx *fasthttp.RequestCtx) error {
			return nil
		})
		err = h(ctx)
		convey.So(err, convey.ShouldBeNil)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		bsvc := ban.NewMockBanServiceIface(ctrl)
		p.ApplyFuncReturn(ban.GetBanService, bsvc, nil)

		// GetMemberStatus err
		bsvc.EXPECT().GetMemberStatus(gomock.Any(), gomock.Eq(int64(0))).Return(nil, errors.New("xxx"))
		h = bf.Do(func(rctx *fasthttp.RequestCtx) error {
			return nil
		})
		err = h(ctx)
		convey.So(err, convey.ShouldBeNil)

		// GetMemberStatus success
		bsvc.EXPECT().GetMemberStatus(gomock.Any(), gomock.Eq(int64(0))).Return(&ban.UserStatusWrap{
			UserState: &ban.UserStatus{
				IsNormal: false,
				Uid:      0,
				BanItems: []*ban2.UserStatus_BanItem{
					{
						BizType:  "aas",
						TagName:  "asa",
						TagValue: "asas",
					},
					{
						BizType:  "1",
						TagName:  "2",
						TagValue: "3",
					},
				},
			},
		}, nil)
		h = bf.Do(func(rctx *fasthttp.RequestCtx) error {
			return nil
		})
		err = h(ctx)
		convey.So(err, convey.ShouldEqual, berror.ErrOpenAPIUserLoginBanned)

		// GetMemberStatus success2
		bsvc.EXPECT().GetMemberStatus(gomock.Any(), gomock.Eq(int64(0))).Return(&ban.UserStatusWrap{
			UserState: &ban.UserStatus{
				IsNormal: false,
				Uid:      0,
				BanItems: []*ban2.UserStatus_BanItem{
					{
						BizType:   "1",
						TagName:   "2",
						TagValue:  "3",
						ErrorCode: 1212,
					},
				},
			},
		}, nil)
		h = bf.Do(func(rctx *fasthttp.RequestCtx) error {
			return nil
		})
		err = h(ctx)
		e := berror.NewBizErr(1212, "")
		convey.So(err.Error(), convey.ShouldEqual, e.Error())

		// GetMemberStatus no ban info
		bsvc.EXPECT().GetMemberStatus(gomock.Any(), gomock.Eq(int64(0))).Return(&ban.UserStatusWrap{
			UserState: &ban.UserStatus{
				IsNormal: false,
				Uid:      0,
				BanItems: []*ban2.UserStatus_BanItem{},
			},
		}, nil)
		h = bf.Do(func(rctx *fasthttp.RequestCtx) error {
			return nil
		})
		err = h(ctx)
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestInit(t *testing.T) {
	convey.Convey("TestInit", t, func() {
		bf := &banFilter{}
		convey.So(bf.GetName(), convey.ShouldEqual, filter.BanFilterKey)
		ctx, _ := test.NewReqCtx()
		args := []string{"--skipAID=true", "--bizType=0"}
		err := bf.Init(ctx, args...)
		convey.So(err.Error(), convey.ShouldEqual, "flag provided but not defined: -bizType")

		args = []string{"ban", "--banTags=xas"}
		err = bf.Init(ctx, args...)
		convey.So(err.Error(), convey.ShouldEqual, "invalid character 'x' looking for beginning of value")

		args = []string{"ban", `--banTags=[{"bizType":"213", "tagName":"!212", "tagValue":"111"}]`}
		err = bf.Init(ctx, args...)
		convey.So(err, convey.ShouldBeNil)
		convey.So(len(bf.br.bts), convey.ShouldEqual, 1)
		convey.So(bf.br.bts, convey.ShouldContainKey, banTag{
			BizType:  "213",
			TagName:  "!212",
			TagValue: "111",
		})

	})
}
