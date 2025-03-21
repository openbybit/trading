package gopeninterest

import (
	"code.bydev.io/fbu/future/sdk.git/pkg/future"
	"github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestMergeExtraExceedResult(t *testing.T) {
	convey.Convey("TestMergeExtraExceedResult", t, func() {
		result := mergeExtraExceedResult(nil)
		convey.So(result, convey.ShouldBeNil)

		dto := &oiExceededResultDTO{}
		exceedResult := mergeExtraExceedResult(dto)
		convey.So(exceedResult, convey.ShouldEqual, dto)

		dto = &oiExceededResultDTO{
			Symbol: future.Symbol(5),
			BuyExceededResultMap: map[future.UserID]int64{
				future.UserID(521312): 30,
				future.UserID(521313): 30,
			},
			SellExceededResultMap: map[future.UserID]int64{
				future.UserID(521312): 30,
				future.UserID(521313): 30,
			},
			ExtraBuyExceedResultMap: map[future.UserID]int64{
				future.UserID(521312): 30,
				future.UserID(521314): 30,
			},
			ExtraSellExceedResultMap: map[future.UserID]int64{
				future.UserID(521312): 30,
				future.UserID(521315): 30,
			},
		}
		exceedResult = mergeExtraExceedResult(dto)
		convey.So(len(exceedResult.BuyExceededResultMap), convey.ShouldBeGreaterThanOrEqualTo, len(exceedResult.ExtraBuyExceedResultMap))
		convey.So(len(exceedResult.SellExceededResultMap), convey.ShouldBeGreaterThanOrEqualTo, len(exceedResult.ExtraSellExceedResultMap))

		convey.So(exceedResult.BuyExceededResultMap, convey.ShouldContainKey, future.UserID(521312))
		convey.So(exceedResult.BuyExceededResultMap, convey.ShouldContainKey, future.UserID(521314))

		convey.So(exceedResult.SellExceededResultMap, convey.ShouldContainKey, future.UserID(521312))
		convey.So(exceedResult.SellExceededResultMap, convey.ShouldContainKey, future.UserID(521313))
		convey.So(exceedResult.SellExceededResultMap, convey.ShouldContainKey, future.UserID(521315))
	})
}
