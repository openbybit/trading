package gsymbol

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestFutureBuild(t *testing.T) {
	data := `{"data":[{"baseCurrency":"BTC","baseInitialMarginRateE4":100,"baseMaintenanceMarginRateE4":50,"baseMaxOrdPzValueX":100000000000000,"buyAdlUserId":423213,"coin":1,"coinName":"BTC","contractStatus":1,"contractType":1,"crossIdx":1,"crossName":"BTCIP","dailyAdlQtyX":5000000,"defaultMakerFeeRateE8":-25000,"defaultSettleFeeRateE8":50000,"defaultTakerFeeRateE8":60000,"enableConfig":"{\"enable_shard_list\":[\"ALL\"]}","exchangeDetail":[{"exchange":"FromKraken","originalPairTo":"ToUSD","reqChannel":"BTC/USD","rspChannel":"BTC/USD","weightE4":2318},{"exchange":"FromGemini","originalPairTo":"ToUSD","reqChannel":"btcusd","rspChannel":"btcusd","weightE4":429},{"exchange":"FromBittrex","originalPairTo":"ToUSDT","reqChannel":"BTC-USDT","rspChannel":"BTC-USDT","weightE4":294},{"exchange":"FromBitStamp","originalPairTo":"ToUSD","reqChannel":"btcusd","rspChannel":"live_trades_btcusd","weightE4":2266},{"exchange":"FromCoinBase","originalPairTo":"ToUSD","reqChannel":"BTC-USD","rspChannel":"BTC-USD","weightE4":4693}],"fundingRateClampE8":375000,"fundingRateIntervalMin":480,"hasFundingFee":true,"impactMarginNotionalX":10,"importTimeE9":1627813261000000000,"indexPriceCount":0,"indexPriceOffset":-1,"indexPriceSum":0,"indexSort":1,"lotFraction":0,"lotSizeX":1,"markPricePassthroughCross":false,"maxMakerFeeRateE8":-1,"maxNewOrderQtyX":1000000,"maxOrderBookQtyX":100000000,"maxPositionSizeX":100000000,"maxPriceX":9999990000,"maxTakerFeeRateE8":75000,"maxValueX":9223372036854776000,"minMakerFeeRateE8":-75000,"minPriceX":5000,"minQtyX":1,"minTakerFeeRateE8":1,"minValueX":1,"mode":0,"obDepthMergeTimes":"1,2,4,10","oneE4":10000,"oneE8":100000000,"oneX":1,"openInterestLimitX":1000000,"orderBookDepthValueX":0,"postOnlyFactor":5,"priceFraction":2,"priceLimitPntE6":30000,"priceScale":4,"qtyScale":0,"quoteCurrency":"USD","quoteSymbol":1,"riskLimitCount":10,"riskLimitList":[{"initialMarginRateE4":100,"lowestRisk":true,"maintenanceMarginRateE4":50,"maxLeverageE2":10000,"maxOrdPzValueX":100000000000000,"riskId":1,"symbol":1,"symbolStr":"BTCUSD"},{"initialMarginRateE4":150,"lowestRisk":false,"maintenanceMarginRateE4":100,"maxLeverageE2":6667,"maxOrdPzValueX":200000000000000,"riskId":2,"symbol":1,"symbolStr":"BTCUSD"},{"initialMarginRateE4":200,"lowestRisk":false,"maintenanceMarginRateE4":150,"maxLeverageE2":5000,"maxOrdPzValueX":300000000000000,"riskId":3,"symbol":1,"symbolStr":"BTCUSD"},{"initialMarginRateE4":250,"lowestRisk":false,"maintenanceMarginRateE4":200,"maxLeverageE2":4000,"maxOrdPzValueX":400000000000000,"riskId":4,"symbol":1,"symbolStr":"BTCUSD"},{"initialMarginRateE4":300,"lowestRisk":false,"maintenanceMarginRateE4":250,"maxLeverageE2":3334,"maxOrdPzValueX":500000000000000,"riskId":5,"symbol":1,"symbolStr":"BTCUSD"},{"initialMarginRateE4":350,"lowestRisk":false,"maintenanceMarginRateE4":300,"maxLeverageE2":2858,"maxOrdPzValueX":600000000000000,"riskId":6,"symbol":1,"symbolStr":"BTCUSD"},{"initialMarginRateE4":400,"lowestRisk":false,"maintenanceMarginRateE4":350,"maxLeverageE2":2500,"maxOrdPzValueX":700000000000000,"riskId":7,"symbol":1,"symbolStr":"BTCUSD"},{"initialMarginRateE4":450,"lowestRisk":false,"maintenanceMarginRateE4":400,"maxLeverageE2":2223,"maxOrdPzValueX":800000000000000,"riskId":8,"symbol":1,"symbolStr":"BTCUSD"},{"initialMarginRateE4":500,"lowestRisk":false,"maintenanceMarginRateE4":450,"maxLeverageE2":2000,"maxOrdPzValueX":900000000000000,"riskId":9,"symbol":1,"symbolStr":"BTCUSD"},{"initialMarginRateE4":550,"lowestRisk":false,"maintenanceMarginRateE4":500,"maxLeverageE2":1819,"maxOrdPzValueX":1000000000000000,"riskId":10,"symbol":1,"symbolStr":"BTCUSD"}],"sellAdlUserId":423213,"settleCurrency":"BTC","settleFundingImmediately":true,"settleTimeE9":0,"startCalcSettlePriceTimeE9":0,"startRiskId":1,"startTradingTimeE9":1000000000,"stepInitialMarginRateE4":50,"stepMaintenanceMarginRateE4":50,"stepMaxOrdPzValueX":100000000000000,"symbol":1,"symbolAlias":"BTCUSD","symbolDesc":"BTCUSD","symbolName":"BTCUSD","tickSizeFraction":1,"tickSizeX":5000,"unrealisedProfitBorrowable":false,"upnlFraction":4,"valueScale":8,"version":2,"walletBalanceFraction":8}],"version":"dd33a314-59b4-46fa-b1b6-9b9e8bf0a42b"}`
	m := FutureManager{}
	if err := m.build(data); err != nil {
		t.Error(err)
	}
	printJson(m.list)
}

func TestManager(t *testing.T) {
	if err := Start(nil); err != nil {
		t.Error(err)
	}

	fmr := globalMgr.GetFutureManager()

	if len(fmr.list) == 0 {
		t.Errorf("empty future list")
	} else {
		bybit := fmr.brokerMap[brokerID_BYBIT]
		t.Logf("future size: %d, %d", len(fmr.list), len(fmr.brokerMap))
		t.Logf("bybit size:  %d, %d, %d, %d, %d", len(bybit.list), len(bybit.mapByID), len(bybit.mapByName), len(bybit.mapByAlias), len(bybit.mapByCoin))
	}

	omr := globalMgr.GetOptionManager()
	if len(omr.list) == 0 {
		t.Errorf("empty option list")
	} else {
		t.Logf("option size: %d, %d, %d", len(omr.list), len(omr.mapByID), len(omr.mapByName))
	}

	smr := globalMgr.GetSpotManager()
	if len(smr.list) == 0 {
		t.Errorf("empty spot list")
	} else {
		t.Logf("spot size: %d, %d, %d", len(smr.list), len(smr.mapByID), len(smr.mapByName))
	}

	// printJson(globalMgr.getFutureMgr().list)
	// printJson(globalMgr.getOptionMgr().list)
	// printJson(globalMgr.getSpotMgr().list)
}

func printJson(x interface{}) {
	s, _ := json.MarshalIndent(x, "", "\t")
	fmt.Println(string(s))
}
