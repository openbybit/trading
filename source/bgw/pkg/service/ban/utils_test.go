package ban

import (
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	"github.com/segmentio/encoding/json"
	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common/berror"
	"bgw/pkg/server/metadata/bizmetedata"
	"bgw/pkg/service/symbolconfig"
	"bgw/pkg/test"
)

func TestTradeCheckSingleSymbol(t *testing.T) {
	Convey("TestTradeCheckSingleSymbol", t, func() {

		Convey("GetBanService err", func() {
			ctx, _ := test.NewReqCtx()
			p := gomonkey.ApplyFuncReturn(GetBanService, nil, errors.New("xxxx"))
			defer p.Reset()
			err := TradeCheckSingleSymbol(ctx, "ccc", "as", 1, true, nil)
			So(err, ShouldBeNil)
			tr := bizmetedata.TradeCheckFromContext(ctx)
			So(tr, ShouldBeNil)
		})

		Convey("VerifyTrade err", func() {
			ctx, _ := test.NewReqCtx()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			bsvc := NewMockBanServiceIface(ctrl)
			p := gomonkey.ApplyFuncReturn(GetBanService, bsvc, nil)
			defer p.Reset()
			s := &UserStatusWrap{}
			bsvc.EXPECT().VerifyTrade(gomock.Any(), gomock.Eq(int64(1)), gomock.Eq("ccc"), gomock.Eq(s), gomock.Any()).Return(false, errors.New("xxx"))
			err := TradeCheckSingleSymbol(ctx, "ccc", "as", 1, true, s)
			So(err.Error(), ShouldEqual, "xxx")
			tr := bizmetedata.TradeCheckFromContext(ctx)
			So(tr, ShouldBeNil)
		})

		Convey("symbol field name ''", func() {
			ctx, _ := test.NewReqCtx()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			bsvc := NewMockBanServiceIface(ctrl)
			p := gomonkey.ApplyFuncReturn(GetBanService, bsvc, nil)
			defer p.Reset()
			s := &UserStatusWrap{}
			bsvc.EXPECT().VerifyTrade(gomock.Any(), gomock.Eq(int64(1)), gomock.Eq("ccc"), gomock.Eq(s), gomock.Any()).Return(true, nil)
			err := TradeCheckSingleSymbol(ctx, "ccc", "", 1, true, s)
			So(err, ShouldBeNil)
			tr := bizmetedata.TradeCheckFromContext(ctx)
			So(tr.BannedReduceOnly, ShouldEqual, true)
		})

		Convey("symbol field name not ''", func() {
			ctx, _ := test.NewReqCtx()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			bsvc := NewMockBanServiceIface(ctrl)
			p := gomonkey.ApplyFuncReturn(GetBanService, bsvc, nil)
			defer p.Reset()
			s := &UserStatusWrap{}
			bsvc.EXPECT().VerifyTrade(gomock.Any(), gomock.Eq(int64(1)), gomock.Eq("ccc"), gomock.Eq(s), gomock.Any()).Return(true, nil)
			err := TradeCheckSingleSymbol(ctx, "ccc", "as", 1, true, s)
			So(err, ShouldBeNil)
			tr := bizmetedata.TradeCheckFromContext(ctx)
			So(tr.BannedReduceOnly, ShouldEqual, true)
		})
	})
}

func TestTradeCheckBatchSymbol(t *testing.T) {
	Convey("TestTradeCheckBatchSymbol", t, func() {

		Convey("GetBanService err", func() {
			ctx, _ := test.NewReqCtx()
			p := gomonkey.ApplyFuncReturn(GetBanService, nil, errors.New("xxxx"))
			defer p.Reset()
			jj, err := TradeCheckBatchSymbol(ctx, "ccc", "as", 1, true, nil)
			So(err, ShouldBeNil)
			So(jj, ShouldEqual, "")
			tr := bizmetedata.TradeCheckFromContext(ctx)
			So(tr, ShouldBeNil)
		})

		Convey("success", func() {
			ctx, _ := test.NewReqCtx()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			bsvc := NewMockBanServiceIface(ctrl)
			p := gomonkey.ApplyFuncReturn(GetBanService, bsvc, nil)
			defer p.Reset()
			m := make(map[string]struct{})
			m["verify err"] = struct{}{}
			m["verify err-biz"] = struct{}{}
			m["verify success"] = struct{}{}
			m["verify success ro=true"] = struct{}{}
			p.ApplyFuncReturn(retrieveSymbols, m, nil)

			s := &UserStatusWrap{}

			bsvc.EXPECT().VerifyTrade(gomock.Any(), gomock.Eq(int64(1)), gomock.Eq("ccc"), gomock.Eq(s), gomock.Any()).Return(true, nil)
			bsvc.EXPECT().VerifyTrade(gomock.Any(), gomock.Eq(int64(1)), gomock.Eq("ccc"), gomock.Eq(s), gomock.Any()).Return(false, nil)
			bsvc.EXPECT().VerifyTrade(gomock.Any(), gomock.Eq(int64(1)), gomock.Eq("ccc"), gomock.Eq(s), gomock.Any()).Return(false, errors.New("xxx"))
			bsvc.EXPECT().VerifyTrade(gomock.Any(), gomock.Eq(int64(1)), gomock.Eq("ccc"), gomock.Eq(s), gomock.Any()).Return(false, berror.ErrOpenAPIUserAllBanned)
			jj, err := TradeCheckBatchSymbol(ctx, "ccc", "ccc", 1, true, s)
			mm := make(map[string]int)
			So(err, ShouldBeNil)
			err = json.Unmarshal([]byte(jj), &mm)
			So(err, ShouldBeNil)
			So(len(mm), ShouldEqual, 3)
			i := 0
			// map遍历结果会变，统计value个数
			for _, v := range mm {
				if v > 2 {
					So(v, ShouldEqual, 11108)
				}
				if v == 1 {
					i++
				}
				if v == 0 {
					i++
				}
			}
			So(i, ShouldEqual, 2)
			tr := bizmetedata.TradeCheckFromContext(ctx)
			So(tr, ShouldBeNil)
		})
	})
}

func TestRetrieveSymbols(t *testing.T) {
	Convey("TestRetrieveSymbols", t, func() {
		ctx, _ := test.NewReqCtx()
		p := gomonkey.ApplyFuncReturn(symbolconfig.GetBatchSymbol, []string{"AA", "VV"}, nil)
		p.ApplyFuncReturn(symbolconfig.GetBatchSymbolByFieldName, []string{"AA", "VV"}, nil)
		defer p.Reset()
		mm, err := retrieveSymbols(ctx, "")
		So(err, ShouldBeNil)
		So(len(mm), ShouldEqual, 2)
		So(mm, ShouldContainKey, "AA")
		So(mm, ShouldContainKey, "VV")

		mm, err = retrieveSymbols(ctx, "222")
		So(err, ShouldBeNil)
		So(len(mm), ShouldEqual, 2)
		So(mm, ShouldContainKey, "AA")
		So(mm, ShouldContainKey, "VV")

		p.ApplyFuncReturn(symbolconfig.GetBatchSymbolByFieldName, []string{}, errors.New("xxx"))
		mm, err = retrieveSymbols(ctx, "222")
		So(mm, ShouldBeNil)
		So(err.Error(), ShouldEqual, "xxx")
	})
}

func TestRetrieveSymbolOptions(t *testing.T) {
	Convey("TestRetrieveSymbolOptions", t, func() {
		ctx, _ := test.NewReqCtx()
		mm := retrieveSymbolOptions(ctx, true, "")
		So(len(mm), ShouldEqual, 2)
		o := &Options{}
		for _, opt := range mm {
			opt(o)
		}
		So(o.siteAPI, ShouldBeTrue)
		So(o.symbol, ShouldEqual, "")

		mm = retrieveSymbolOptions(ctx, true, "")
		So(len(mm), ShouldEqual, 2)
		So(o.siteAPI, ShouldBeTrue)
		So(o.symbol, ShouldNotBeNil)
		So(o.symbol, ShouldEqual, "")

		mm = retrieveSymbolOptions(ctx, true, "222")
		So(len(mm), ShouldEqual, 2)
		o = &Options{}
		for _, opt := range mm {
			opt(o)
		}
		So(o.siteAPI, ShouldBeTrue)
		So(o.symbol, ShouldNotBeNil)
		So(o.symbol, ShouldEqual, "")
	})

}

func TestIsUsdcBanType(t *testing.T) {
	Convey("TestIsUsdcBanType", t, func() {
		So(IsUsdcBanType(1111), ShouldBeFalse)
		So(IsUsdcBanType(BantypeUsdcFutureAllKo), ShouldBeTrue)
		So(IsUsdcBanType(BantypeUsdcFutureAllKo), ShouldBeTrue)
	})
}
