package gformula

import (
	"math"
	"testing"
	"time"
)

var (
	callModel      BSModel
	callModelInter BSModel
	putModel       BSModel
	putModelInter  BSModel
	array          []float64
)

func init() {
	t, _ := time.Parse("2006/01/02", "20210701")
	curTimeSec := t.Unix()
	expireSec := curTimeSec + 18*24*3600
	callModel = NewBSModelWithTime(OptionTypeCall, 0, curTimeSec, expireSec)
	callModelInter = NewBSModelWithTime(OptionTypeCall, 0.1, curTimeSec, expireSec)
	putModel = NewBSModelWithTime(OptionTypePut, 0, curTimeSec, expireSec)
	putModelInter = NewBSModelWithTime(OptionTypePut, 0.01, curTimeSec, expireSec)

	array = []float64{
		37818.59, 37211.15, 36859.35, 36710.82, 36754.87, 37276.06, 36940.05, 36842.56, 36206.54,
		36245.14, 36449.31, 36605.22, 36160.34, 37025.26, 36630.22, 36229.2,
		35914.22, 36164.21, 36229.74, 35977.88, 36324, 36424.26, 36436.37,
		36683.96, 36185.28, 36290.57, 36276.22, 36479.2, 36673.44, 36817.41,
		36738.79, 37224.95, 37387.83, 37278.05, 37158.31, 37205.8, 37284.93,
		37538.43, 37869.29, 38046.71, 37882.48, 38017.99, 38018.72, 37884.26,
		37747.58, 37805.85, 37542.06, 37580.1, 37486.54, 37300.15, 37412.21,
		37648.22, 37975.81, 38815.25, 38960.54, 38643.8, 38823.62, 39128.91,
		39285.05, 38788.62, 39034.03, 38449.4, 38638.53, 38567.96, 38889.28,
		38811.32, 38559.46, 38591.18, 38705.93, 38752.02, 38901.86, 39239.24,
		38688.99, 37923.38, 37801.88, 37960.13, 37580.93, 36737.85, 36787.79,
		36616.07, 36720.74, 36862.53, 36639.48, 36585.16, 36900.27, 36751.24,
		36738.71, 36989.34, 36923.32, 37115.59, 37134.6, 36827.76, 37053.57,
		37229.29, 37127.77, 36857, 37697.33, 37717.54, 37672.06, 37432.08,
		37696.21, 37619.43, 37724.99, 37594.76, 37796.48, 36362.02, 36486.05,
		35904.45, 36044.13, 35748.9, 35654.73, 36325.52, 36080.32, 36026.27,
		36046.22, 35790.06, 35855.06, 34988.15, 35082.36, 35538.88, 35595.01,
		36244.1, 36159.51, 36019.21, 36109.01, 36249.91, 35873.31, 36161.77,
		36354.78, 36194.35, 36036.18, 35895.41, 35888.42, 35649.02, 36125.27,
		36172.98, 36250.17, 36044.1, 35896, 35768.09, 35982.66, 35952.95, 35567.76,
		35805.65, 36305.13, 36695.98, 36660.4, 36400.72, 36272.76, 36307.92, 36094.72,
		36161.79, 36125.36, 36305.18, 36523.72, 36398.7, 36635.73, 36321.56, 36062.78,
		36035.44, 35804.59, 35730.4, 35498.34, 35647.65, 34453.96, 34132.14, 34169.36,
		33570.22, 33689.75, 33802.31, 32739.16, 32844.87, 32869.01, 32923.3, 32621.56,
		32983.22, 32929.38, 32836.71, 32823.6, 33127.43, 32910.81, 32394.54, 31773.6, 31728,
		32194.89, 32192.45, 32777.17, 32903.14, 33629.33, 33497.84, 33465.1, 33407.04,
		32917.38, 32537.83, 32885.67, 32884.09, 32995.34, 33532.95, 34239.29, 34195.56,
		34075.19, 34028.84, 34512.52, 35015.16, 34919.99, 35097.97, 34694.66, 36377.76,
		36530.74, 36554.5, 35953.07, 36293.86, 36411.02, 36680.46, 37112.56, 37405.51,
		36981.32, 36909.99, 37128.01, 36986.15, 36807.67, 36839.87, 36767.34, 36984.42,
		36419.95, 37440.21, 37854.5, 37842.28, 37799.93, 37876.99, 37028.22, 36911.3,
		36838.84, 36472.53, 36788.48, 36700.07, 36486.08, 36712.12, 36852.85, 36696.4,
		36434.5, 36311.58, 36364.42, 36772.8, 37056.17, 37015.83, 36527.33, 37166.43,
		37035.51, 37060.68, 37497.76, 37404.93, 37332.13, 37464.58, 37261.25, 36824.29,
		36665.13, 36920.07, 37181.53, 37261.54, 37154.36, 36914.07, 37357.21, 37336.16,
		37066.09, 36250.32, 35548.6, 35595.34, 35251.46, 35181.89, 35337.04, 35297.83,
		35186.73, 35256.47, 35694.39, 35623.54, 36062.9, 36006.96, 35687.62, 35411.45,
		35546.01, 35747.98, 35924.62, 35744.26, 36015.41, 35829.88, 35705.25, 35550.81,
		35735.25, 35801.78, 35476.06, 34883.61, 35196.28, 34986.15, 35378.94, 35284.85,
		35626.03, 35845.26, 35992.29, 35912.61, 35898.58, 35898.89, 35776.98, 36043.6,
		37028.65, 37322.14, 37508.01, 37672.39, 39267.83, 38908.39, 38882.55, 39022.12,
		39133.42, 39193.41, 38992.27, 38990.03, 39285, 39433.41, 39605.09, 39563.73,
		39027.21, 39097.09, 39336.14, 39168.29, 39596.27, 40551.06, 40558.56, 40696.85,
		40177.5, 40232.69, 39746.72, 39713.34, 39842.29, 40291.02, 40187.76, 40531.07,
		40376.94, 39942.18, 40272.95, 40431.97, 40531.38, 40348.38, 40302.33, 40387.61,
		39819.4, 40119.55, 39926.2, 39846.46, 40427.16, 40304.13, 40002.94, 40059.67,
		40192.83, 40708.84, 40589.78, 39965.79, 39958.09, 39981.84, 40135.56, 40165.64,
		39891.66, 40108.83,
	}
}

func getTime(value string) int64 {
	t, err := time.Parse("2006/01/02 15:04:05", value)
	if err != nil {
		panic(err)
	}
	return t.Unix()
}

func assertFloatEqual(t *testing.T, x1, x2, tolerance float64) {
	if math.Abs(x1-x2) > tolerance {
		t.Errorf("not equal, x1: %v, x2: %v", x1, x2)
	}
}

func assertFalse(t *testing.T, x bool) {
	if x {
		t.Errorf("should be false")
	}
}

func assertTrue(t *testing.T, x bool) {
	if !x {
		t.Errorf("should be true")
	}
}

func TestOptionValue(t *testing.T) {
	err := kDefaultAccuracy

	optionValue, _ := callModel.Setup(34000, 32000).CalOptionValue(2.5)
	t.Log("call optionvalue:", optionValue)
	assertFloatEqual(t, 8258.01373473844, optionValue, err)

	putValue, _ := putModel.Setup(34000, 32000).CalOptionValue(2.5)
	t.Log("put optionvalue:", putValue)
	assertFloatEqual(t, 6258.013734738441, putValue, err)
	putValue, _ = putModel.Setup(34000, 32000).CalOptionValue(2.5)
	t.Log("put optionvalue:", putValue)
	assertFloatEqual(t, 6258.013734738441, putValue, err)

	putValue, _ = putModel.Setup(32000, 34000).CalOptionValue(2.5)
	t.Log("put optionvalue2: ", putValue)
	assertFloatEqual(t, 8258.01373473844, putValue, err)

	{
		iv, _ := callModel.Setup(34000, 32000).CalImpliedVolatility(8258.01373473844)
		t.Log("call iv:", iv)
		assertFloatEqual(t, 2.5, iv, err)

		putIv, _ := putModel.Setup(34000, 32000).CalImpliedVolatility(6258.013734738441)
		t.Log("put iv:", putIv)
		assertFloatEqual(t, 2.5, putIv, err)
	}
}

func TestIV1(t *testing.T) {
	callModel.CalTimeRate(getTime("2021/06/09 09:49:17"), getTime("2021/06/18 08:00:00"))
	callModel.Setup(34277.23, 30000)
	iv, err := callModel.CalImpliedVolatility(8226.02)
	t.Log("sample iv1:", iv, err)
	assertFloatEqual(t, iv, 2.943, 0.001)
}

func TestInterIV(t *testing.T) {
	callModelInter.CalTimeRate(getTime("2021/06/09 09:49:17"), getTime("2021/06/18 08:00:00"))
	callModelInter.Setup(34277.23, 30000)
	price, _ := callModelInter.CalOptionValue(2.943)
	t.Logf("price: %v", price)
	iv, _ := callModelInter.CalImpliedVolatility(price)
	t.Logf("sample iv1: %v", iv)
	assertFloatEqual(t, iv, 2.943, 0.001)
}

func TestIV2(t *testing.T) {
	callModel.CalTimeRate(getTime("2021/06/09 09:49:17"), getTime("2021/06/18 08:00:00"))
	//2979.559
	/**
	 * underlying/strikprice确定的情况，ITM/OTM时，BS模型有个最低理论价格，期权价格不能无限制小下去
	 * 32979.56
	 * */
	callModel.Setup(32979.56, 30000)
	iv, _ := callModel.CalImpliedVolatility(2979.56)
	t.Logf("iv: %v", iv)
	assertFloatEqual(t, iv, 0, 0.0001)
	callModel.Setup(32979.56, 30000)
	iv, _ = callModel.CalImpliedVolatility(32979.56)
	t.Logf("iv: %v", iv)
	assertFloatEqual(t, iv, kMaxVolatility, 0.0001)
}

func TestIV3(t *testing.T) {
	putModel.CalTimeRate(getTime("2021/06/09 09:49:17"), getTime("2021/06/18 08:00:00"))
	putModel.Setup(34277.23, 30000)
	iv, _ := putModel.CalImpliedVolatility(3948.79)
	t.Logf("put sample iv3: %v", iv)
	assertFloatEqual(t, iv, 2.943, 0.0001)
}

func TestInterIV3(t *testing.T) {
	putModelInter.CalTimeRate(getTime("2021/06/09 09:49:17"), getTime("2021/06/18 08:00:00"))
	putModelInter.Setup(34277.23, 30000)
	price, _ := putModelInter.CalOptionValue(2.943)
	t.Logf("price: %v", price)
	iv, _ := putModelInter.CalImpliedVolatility(price)
	t.Logf("put sample iv3: %v", iv)
	assertFloatEqual(t, iv, 2.943, 0.001)
}

func TestIV5(t *testing.T) {
	expire := getTime("2021/07/24 08:00:00")
	putModel.CalTimeRate(expire-4*3600, expire)
	putModel.Setup(34562.8, 150000)
	iv, _ := putModel.CalImpliedVolatility(116800.265815)
	t.Logf("put sample iv5: %v", iv)
	value, _ := putModel.CalOptionValue(18.773262925734173)
	t.Logf("put optionvalue5: %v", value)

	callModel.Setup(30000, 32000)
	value, _ = callModel.CalOptionValue(kDefaultAccuracy)
	t.Logf("call otm min optionvalue: %v", value)
	assertFloatEqual(t, 0, value, 0.0001)

	value, _ = callModel.CalOptionValue(kMaxVolatility)
	t.Logf("call otm max optionvalue1: %v", value)
	assertFloatEqual(t, 30000, value, 0.0001)

	callModel.Setup(34000, 32000)
	value, _ = callModel.CalOptionValue(kDefaultAccuracy)
	t.Logf("call itm min optionvalue: %v", value)

	value, _ = callModel.CalOptionValue(kMaxVolatility)
	t.Logf("call itm max optionvalue1: %v", value)
	assertFloatEqual(t, 34000, value, 0.0001)

	putModel.Setup(34000, 32000)
	value, _ = putModel.CalOptionValue(kDefaultAccuracy)
	t.Logf("put otm optionvalue: %v", value)
	assertFloatEqual(t, 0, value, 0.0001)

	value, _ = putModel.CalOptionValue(kMaxVolatility)
	t.Logf("put otm optionvalue2: %v", value)
	assertFloatEqual(t, 32000, value, 0.0001)

	putModel.Setup(30000, 32000)
	value, _ = putModel.CalOptionValue(kDefaultAccuracy)
	t.Logf("put itm optionvalue: %v", value)

	value, _ = putModel.CalOptionValue(kMaxVolatility)
	t.Logf("put itm optionvalue2: %v", value)
	assertFloatEqual(t, 32000, value, 0.0001)
}

func TestPrice4(t *testing.T) {
	callModel.CalTimeRate(getTime("2021/06/09 10:13:25"), getTime("2021/06/18 08:00:00"))
	putModel.CalTimeRate(getTime("2021/06/09 10:13:25"), getTime("2021/06/18 08:00:00"))

	price, _ := putModel.Setup(34070.88, 36000).CalOptionValue(1.0237)
	price2, _ := callModel.Setup(34070.88, 36000).CalOptionValue(1.0237)
	t.Logf("put sample4 price: %v, %v", price, price2)
	assertFloatEqual(t, 3328.19, price, 0.01)
	assertFloatEqual(t, 1399.07, price2, 0.01)
}

func TestIV64000A(t *testing.T) {
	//测试极端情况下的迭代收敛
	expire := getTime("2021/07/24 08:00:00")
	callModel.CalTimeRate(expire-(int64)(0.0244496131*365*float64(kOneDaySec)), expire)
	callModel.Setup(64000, 1)
	iv1, _ := callModel.CalImpliedVolatility(63999.1)
	t.Logf("call sample iv64000 lower: %v", iv1)
	value1, _ := callModel.CalOptionValue(iv1)
	assertFloatEqual(t, value1, 63999.1, 0.00001)

	// TODO 不一致
	iv2, _ := callModel.CalImpliedVolatility(63999.9)
	t.Logf("call sample iv64000 upper: %v", iv2)
	value2, _ := callModel.CalOptionValue(iv2)
	assertFloatEqual(t, value2, 63999.9, 0.00001)

	putModel.CalTimeRate(expire-(int64)(0.0244496131*365*float64(kOneDaySec)), expire)
	putModel.Setup(64000, 1)
	iv1, _ = putModel.CalImpliedVolatility(0.1)
	t.Logf("put sample iv64000 lower: %v", iv1)
	value1, _ = putModel.CalOptionValue(iv1)
	assertFloatEqual(t, value1, 0.1, 0.00001)

	iv2, _ = putModel.CalImpliedVolatility(0.9)
	t.Logf("put sample iv64000 upper: %v", iv2)
	value2, _ = putModel.CalOptionValue(iv2)
	assertFloatEqual(t, value2, 0.9, 0.00001)
}

func TestIV64000B(t *testing.T) {
	//测试极端情况下的迭代收敛
	expire := getTime("2021/07/24 08:00:00")
	callModel.CalTimeRate(expire-(int64)(0.0244496131*365*float64(kOneDaySec)), expire)
	callModel.Setup(64000, 4000)
	iv1, _ := callModel.CalImpliedVolatility(60000.1)
	t.Logf("call sample iv64000 lower: %v", iv1)
	value1, _ := callModel.CalOptionValue(iv1)
	assertFloatEqual(t, value1, 60000.1, 0.00001)

	iv2, _ := callModel.CalImpliedVolatility(63999.9)
	t.Logf("call sample iv64000 upper: %v", iv2)
	value2, _ := callModel.CalOptionValue(iv2)
	assertFloatEqual(t, value2, 63999.9, 0.00001)

	putModel.CalTimeRate(expire-(int64)(0.0244496131*365*float64(kOneDaySec)), expire)
	putModel.Setup(64000, 4000)
	iv1, _ = putModel.CalImpliedVolatility(0.1)
	t.Logf("put sample iv64000 lower: %v", iv1)
	value1, _ = putModel.CalOptionValue(iv1)
	assertFloatEqual(t, value1, 0.1, 0.00001)

	iv2, _ = putModel.CalImpliedVolatility(3999.9)
	t.Logf("put sample iv64000 upper: %v", iv2)
	value2, _ = putModel.CalOptionValue(iv2)
	assertFloatEqual(t, value2, 3999.9, 0.00001)
}

func TestIV64000C(t *testing.T) {
	//测试极端情况下的迭代收敛
	expire := getTime("2021/07/24 08:00:00")
	callModel.CalTimeRate(expire-(int64)(0.0244496131*365*float64(kOneDaySec)), expire)
	callModel.Setup(64000, 63999)
	iv1, _ := callModel.CalImpliedVolatility(1.1)
	t.Log("call sample iv64000 lower:", iv1)
	value1, _ := callModel.CalOptionValue(iv1)
	assertFloatEqual(t, value1, 1.1, 0.00001)

	iv2, _ := callModel.CalImpliedVolatility(63999.9)
	t.Log("call sample iv64000 upper:", iv2)
	value2, _ := callModel.CalOptionValue(iv2)
	assertFloatEqual(t, value2, 63999.9, 0.00001)

	iv3, _ := callModel.CalImpliedVolatility(32000)
	t.Log("call sample iv64000 mid:", iv3)
	value3, _ := callModel.CalOptionValue(iv3)
	assertFloatEqual(t, value3, 32000, 0.00001)

	putModel.CalTimeRate(expire-(int64)(0.0244496131*365*float64(kOneDaySec)), expire)
	putModel.Setup(64000, 63999)
	iv1, _ = putModel.CalImpliedVolatility(0.1)
	t.Log("put sample iv64000 lower:", iv1)
	value1, _ = putModel.CalOptionValue(iv1)
	assertFloatEqual(t, value1, 0.1, 0.00001)

	iv2, _ = putModel.CalImpliedVolatility(63998.9)
	t.Logf("put sample iv64000 upper: %v", iv2)
	value2, _ = putModel.CalOptionValue(iv2)
	assertFloatEqual(t, value2, 63998.9, 0.00001)

	iv3, _ = putModel.CalImpliedVolatility(32000)
	t.Log("put sample iv64000 mid:", iv3)
	value3, _ = putModel.CalOptionValue(iv3)
	assertFloatEqual(t, value3, 32000, 0.00001)
}

func TestIV1x(t *testing.T) {
	//测试极端情况下的迭代收敛
	expire := getTime("2021/07/24 08:00:00")
	callModel.CalTimeRate(expire-(int64)(0.0244496131*365*float64(kOneDaySec)), expire)
	callModel.Setup(1, 0.9)
	iv1, _ := callModel.CalImpliedVolatility(0.11)
	t.Log("call sample iv64000 lower:", iv1)
	value1, _ := callModel.CalOptionValue(iv1)
	assertFloatEqual(t, value1, 0.11, 0.00001)

	iv2, _ := callModel.CalImpliedVolatility(0.99)
	t.Log("call sample iv64000 upper:", iv2)
	value2, _ := callModel.CalOptionValue(iv2)
	assertFloatEqual(t, value2, 0.99, 0.00001)

	iv3, _ := callModel.CalImpliedVolatility(0.5)
	t.Log("call sample iv64000 mid:", iv3)
	value3, _ := callModel.CalOptionValue(iv3)
	assertFloatEqual(t, value3, 0.5, 0.00001)

	putModel.CalTimeRate(expire-(int64)(0.0244496131*365*float64(kOneDaySec)), expire)
	putModel.Setup(1, 0.9)
	iv1, _ = putModel.CalImpliedVolatility(0.11)
	t.Log("put sample iv64000 lower:", iv1)
	value1, _ = putModel.CalOptionValue(iv1)
	assertFloatEqual(t, value1, 0.11, 0.00001)

	iv2, _ = putModel.CalImpliedVolatility(0.89)
	t.Log("put sample iv64000 upper:", iv2)
	value2, _ = putModel.CalOptionValue(iv2)
	assertFloatEqual(t, value2, 0.89, 0.00001)

	iv3, _ = putModel.CalImpliedVolatility(0.5)
	t.Log("put sample iv64000 mid:", iv3)
	value3, _ = putModel.CalOptionValue(iv3)
	assertFloatEqual(t, value3, 0.5, 0.00001)
}

func TestIV2x(t *testing.T) {
	//测试极端情况下的迭代收敛
	expire := getTime("2021/07/24 08:00:00")
	callModel.CalTimeRate(expire-(int64)(0.0244496131*365*float64(kOneDaySec)), expire)
	callModel.Setup(1, 0.1)
	iv1, _ := callModel.CalImpliedVolatility(0.91)
	t.Log("call sample iv64000 lower:", iv1)
	value1, _ := callModel.CalOptionValue(iv1)
	assertFloatEqual(t, value1, 0.91, 0.00001)

	iv2, _ := callModel.CalImpliedVolatility(0.99)
	t.Log("call sample iv64000 upper:", iv2)
	value2, _ := callModel.CalOptionValue(iv2)
	assertFloatEqual(t, value2, 0.99, 0.00001)

	iv3, _ := callModel.CalImpliedVolatility(0.95)
	t.Log("call sample iv64000 mid:", iv3)
	value3, _ := callModel.CalOptionValue(iv3)
	assertFloatEqual(t, value3, 0.95, 0.00001)

	iv4, _ := callModel.CalImpliedVolatility(0.9)
	t.Log("call sample iv 4:", iv4)
	assertFloatEqual(t, 0, iv4, 0.00001)

	putModel.CalTimeRate(expire-(int64)(0.0244496131*365*float64(kOneDaySec)), expire)
	putModel.Setup(1, 0.1)
	iv1, _ = putModel.CalImpliedVolatility(0.09)
	t.Log("put sample iv64000 lower:", iv1)
	value1, _ = putModel.CalOptionValue(iv1)
	assertFloatEqual(t, value1, 0.09, 0.00001)

	iv2, _ = putModel.CalImpliedVolatility(0.1)
	t.Log("put sample iv64000 upper:", iv2)
	assertFloatEqual(t, kMaxVolatility, iv2, 0.00001)

	iv3, _ = putModel.CalImpliedVolatility(0.01)
	t.Log("put sample iv64000 mid:", iv3)
	value3, _ = putModel.CalOptionValue(iv3)
	assertFloatEqual(t, value3, 0.01, 0.00001)
}

func TestTT(t *testing.T) {
	callModel.CalTimeRate(0, 86400*10)
	callModel.Setup(30000, 32000)
	v1, _ := callModel.CalOptionValue(1.2)
	t.Log(v1)

	putModel.CalTimeRate(0, 86400*10)
	putModel.Setup(30000, 32000)
	iv2, _ := putModel.CalOptionValue(1.2)
	t.Log(iv2)
}

func TestAdjust(t *testing.T) {
	callModel.Setup(32000, 32000)
	callModel.CalTimeRate(getTime("2021/07/24 07:00:00"), getTime("2021/07/24 08:00:00"))
	t.Log(callModel.Greeks(1))
	assertFalse(t, callModel.IsDecayAdjusted())

	closeTime := "2021/07/24 07:30:01"

	callModel.CalTimeRate(getTime(closeTime), getTime("2021/07/24 08:00:00"))
	assertTrue(t, callModel.IsDecayAdjusted())

	delta, _ := callModel.Delta(1)
	t.Log(delta)

	callModel.CalTimeRate(getTime(closeTime), getTime("2021/07/24 08:00:00"))
	callModel.Setup(32000, 32000)
	delta1, _ := callModel.Delta(1)
	assertFloatEqual(t, delta1, delta, 0.000001)

	t.Log(callModel.CalOptionValue(1))

	// GreekResult result = null;
	// try {
	// 	result = callModel.greeks(1);
	// } catch (Exception e) {
	// 	Assert.assertTrue(e instanceof FunctionStateException);
	// }
	if _, err := callModel.Greeks(1); err != nil {
		t.Error(err)
	}

	// GreekResult greekResult = new GreekResult();
	// greekResult.setGamma(0.1234);
	result, _ := callModel.Greeks(1)
	t.Log("diff gamma:", result)

	// Assert.assertNotEquals(result.getGamma(), greekResult.getGamma())
	//        result = callModel.greeks(1, new FunctionState(greekResult, 32000));
	//        t.Log("same gamma:" + result);
	//        assertFloatEqual(t,result.getGamma(), greekResult.getGamma(), 0.00001);

	// greekResult.setDelta(0.54)
	callModel.CalTimeRate(getTime("2021/07/24 07:30:01"), getTime("2021/07/24 08:00:00"))
	callModel.SetUnderlying(50000)
	callModel.SetStrikePrice(50000)
	t.Log(callModel.CalOptionValue(0.7957))
	t.Log(callModel.Greeks(0.7957))
}

func TestAtm1(t *testing.T) {
	callModel.SetUnderlying(32000)
	callModel.SetStrikePrice(32000)
	value, _ := callModel.CalOptionValue(kDefaultAccuracy)
	assertFloatEqual(t, 0, value, 0.1)
	t.Log("call atm min optionvalue:", value)

	value, _ = callModel.CalOptionValue(kMaxVolatility)
	t.Log("call atm max optionvalue1:", value)
	assertFloatEqual(t, 32000, value, 0.0001)
	iv, _ := callModel.CalImpliedVolatility(2000)
	t.Log("call atm iv: ", iv)
	value, _ = callModel.CalOptionValue(iv)
	assertFloatEqual(t, 2000, value, 0.0001)

	putModel.Setup(32000, 32000)
	value, _ = putModel.CalOptionValue(kDefaultAccuracy)
	assertFloatEqual(t, 0, value, 0.1)
	t.Log("put atm min optionvalue:", value)

	value, _ = putModel.CalOptionValue(kMaxVolatility)
	t.Log("put atm max optionvalue1:", value)
	assertFloatEqual(t, 32000, value, 0.0001)
	iv, _ = putModel.CalImpliedVolatility(2000)
	t.Log("put atm iv: ", iv)
	value, _ = putModel.CalOptionValue(iv)
	assertFloatEqual(t, 2000, value, 0.0001)
}

func TestIV6x(t *testing.T) {
	curTimeSec := getTime("2021/06/09 10:13:25")
	callModel.CalTimeRate(curTimeSec, curTimeSec+(int64)(4.3*3600))
	//callModel.getOptionFunction().setTimeToExpiration(4.3/24/365);
	iv, _ := callModel.Setup(34441, 31000).CalImpliedVolatility(3444.2)
	t.Log("put sample iv6:", iv)
	assertFloatEqual(t, 1.9335, iv, 0.001)
}

func TestStrikePrice(t *testing.T) {
	curTimeSec := getTime("2021/06/09 10:13:25")
	callModel.CalTimeRate(curTimeSec, curTimeSec+(int64)(4.3*3600))
	strikePrice := 31234.5
	iv, _ := callModel.Setup(34441, strikePrice).CalImpliedVolatility(5432.1)
	t.Log("iv:", iv)
	delta, _ := callModel.Delta(iv)
	calStrikePrice, _ := callModel.CalStrikePrice(delta, iv)
	t.Log("calStrikePrice:", calStrikePrice)
	assertFloatEqual(t, strikePrice, calStrikePrice, 0.00001)

	putModel.CalTimeRate(curTimeSec, curTimeSec+(int64)(4.3*3600))
	iv, _ = putModel.Setup(34441, strikePrice).CalImpliedVolatility(5432.1)
	t.Log("put iv:", iv)
	delta, _ = putModel.Delta(iv)
	calStrikePrice, _ = putModel.CalStrikePrice(delta, iv)
	t.Log("put calStrikePrice:", calStrikePrice)
	assertFloatEqual(t, strikePrice, calStrikePrice, 0.00001)

	if _, err := callModel.CalStrikePrice(1, 1.1); err == nil {
		t.Error("should return error")
	} else {
		t.Log(err)
	}

	if _, err := callModel.CalStrikePrice(2, 1.1); err == nil {
		t.Error("should return error")
	} else {
		t.Log(err)
	}
}

func TestStrikePrice2(t *testing.T) {
	callModel.CalTimeRate(getTime("2021/07/01 00:00:00"), getTime("2021/07/08 00:00:00"))
	callModel.SetUnderlying(39600)
	k1, _ := callModel.CalStrikePrice(0.9, 0.801)
	assertFloatEqual(t, 34564, k1, 0.5)
	k2, _ := callModel.CalStrikePrice(0.8, 0.801)
	assertFloatEqual(t, 36293, k2, 0.5)
	k3, _ := callModel.CalStrikePrice(0.6, 0.801)
	assertFloatEqual(t, 38740, k3, 0.5)
	k4, _ := callModel.CalStrikePrice(0.4, 0.801)
	assertFloatEqual(t, 40980, k4, 0.5)
	k5, _ := callModel.CalStrikePrice(0.2, 0.801)
	assertFloatEqual(t, 43743, k5, 0.5)
	k6, _ := callModel.CalStrikePrice(0.1, 0.801)
	assertFloatEqual(t, 45931, k6, 0.5)

	t.Logf("%f, %f, %f, %f, %f, %f", k1, k2, k3, k4, k5, k6)
}

func TestGreek1(t *testing.T) {
	callModel.CalTimeRate(getTime("2021/06/09 09:49:17"), getTime("2021/06/18 08:00:00"))
	callModel.Setup(34277.23, 30000)
	result, _ := callModel.Greeks(2.943)

	delta, _ := callModel.Delta(2.943)
	gamma, _ := callModel.Gamma(2.943)
	theta, _ := callModel.Theta(2.943)
	vega, _ := callModel.Vega(2.943)
	rho, _ := callModel.Rho(2.943)

	err := kDefaultAccuracy
	assertFloatEqual(t, result.Delta, delta, err)
	assertFloatEqual(t, result.Gamma, gamma, err)
	assertFloatEqual(t, result.Theta, theta, err)
	assertFloatEqual(t, result.Vega, vega, err)
	assertFloatEqual(t, result.Rho, rho, err)
}

func TestGreek2(t *testing.T) {
	callModel.CalTimeRate(getTime("2021/06/09 09:49:17"), getTime("2021/06/18 08:00:00"))
	result, _ := callModel.Setup(34277.23, 30000).Greeks(2.943)
	t.Log(result)

	err := kDefaultAccuracy
	assertFloatEqual(t, 0.6983721905, result.Delta, err)
	assertFloatEqual(t, 0.0000220965097, result.Gamma, err)
	assertFloatEqual(t, -308.0298331, result.Theta, err)
	assertFloatEqual(t, 18.68088171, result.Vega, err)
	// assertFloatEqual(t, 0.6983721905, result.Rho, err)
}

func TestGreek3(t *testing.T) {
	putModel.CalTimeRate(getTime("2021/06/09 09:49:17"), getTime("2021/06/18 08:00:00"))
	putModel.Setup(34277.23, 30000)
	result, _ := putModel.Greeks(2.943)

	delta, _ := putModel.Delta(2.943)
	gamma, _ := putModel.Gamma(2.943)
	theta, _ := putModel.Theta(2.943)
	vega, _ := putModel.Vega(2.943)
	rho, _ := putModel.Rho(2.943)

	err := kDefaultAccuracy
	assertFloatEqual(t, result.Delta, delta, err)
	assertFloatEqual(t, result.Gamma, gamma, err)
	assertFloatEqual(t, result.Theta, theta, err)
	assertFloatEqual(t, result.Vega, vega, err)
	assertFloatEqual(t, result.Rho, rho, err)
}

func TestGreek4(t *testing.T) {
	putModel.CalTimeRate(getTime("2021/06/09 09:49:17"), getTime("2021/06/18 08:00:00"))
	putModel.Setup(34277.23, 30000)
	result, _ := putModel.Greeks(2.943)
	t.Log(result)

	err := kDefaultAccuracy
	assertFloatEqual(t, -0.3016278095, result.Delta, err)
	assertFloatEqual(t, 0.0000220965097, result.Gamma, err)
	assertFloatEqual(t, -308.0298331, result.Theta, err)
	assertFloatEqual(t, 18.68088171, result.Vega, err)
	assertFloatEqual(t, -3.4933014, result.Rho, err)
}

func TestHV(t *testing.T) {
	err := kDefaultAccuracy

	hv, _ := CalHistoricalVolatility(array)
	t.Log(hv)
	assertFloatEqual(t, 0.894676, hv, err)

	if _, err := CalHistoricalVolatility(nil); err == nil {
		t.Error("should return error")
	}

	if _, err := CalHistoricalVolatility([]float64{1.0}); err == nil {
		t.Error("should return error")
	}
	hv, _ = CalHistoricalVolatility([]float64{37625.25, 37210.03, 36860.05})
	t.Log(hv)
}

func TestOptionValue2(t *testing.T) {
	model1 := NewBSModelWithTime(OptionTypeCall, 0.0521, 0, (int64)(0.0959/kYearFactor*float64(kOneDaySec)))
	model1.Setup(164, 165)
	value, _ := model1.CalOptionValue(0.29)
	t.Log(value)
	assertFloatEqual(t, 5.78853, value, kDefaultAccuracy)

	// System.out.println(Arrays.toString(BigInteger.valueOf(36).divideAndRemainder(BigInteger.valueOf(30))))
	// System.out.println(BigInteger.valueOf(36).mod(BigInteger.valueOf(30)))
}

// func TestIvCurve(t *testing.T) {
// 	// AbstractOptionFunction f = new CallOptionFunction(0);
// 	f := newOptionFunction(OptionTypeCall, 0)
// 	f.CalTimeRate((int64)(21.57 * 24 * 3600))
// 	f.SetUnderlying(32979.56)
// 	f.SetStrikePrice(30000)

// 	cnt := 1000
// 	max := 16
// 	for i := 0; i <= cnt; i++ {
// 		v := max*i/cnt
// 		t.Logf("%v,%v,%v", i, v, f.OptionValue())
// 	}
// 	// int cnt = 1000;
// 	// double max = 16;
// 	// for(int i = 0; i <= cnt; i++ ) {
// 	// 	double v = (max)*i/cnt;
// 	// 	System.out.println(i + "," + v + "," + f.value(v));
// 	// }
// }

func TestTimeCurve(t *testing.T) {
	// AbstractOptionFunction f = new PutOptionFunction(0);
	f := newOptionFunction(OptionTypePut, 0)

	f.SetUnderlying(30000)
	f.SetStrikePrice(32979.56)
	cnt := 1000
	// max := 16.0;
	for i := 1; i <= cnt; i++ {
		f.CalTimeRate((int64)(i * 3600))
		t.Logf("%v,%v", i, f.CalOptionPrice(1))
	}
}

func TestNegative(t *testing.T) {
	// DateFormat dateFormat = new SimpleDateFormat("yyyyMMdd:HHmmss");
	// long time = dateFormat.parse("20211016:182646").getTime();
	// System.out.println(time);
	tt, _ := time.Parse("20060102:150405", "20211016:182646")
	x := tt.Unix()
	t.Log(x)

	bs := NewBSModelWithTime(OptionTypePut, 0, x, 1634889600)
	bs.Setup(61811.78, 40000)
	price, _ := bs.CalOptionValue(0.42999999999999905)
	t.Log(price)
	assertFloatEqual(t, price, 0, 1e-20)

	// double price = new BSModel(CallPutFlag.P, 0,
	//         time/1000,
	//         1634889600).setup(61811.78, 40000)
	//         .calOptionValue(0.42999999999999905);
	// System.out.println("price:" + price);
	// System.out.println(Double.compare(price, 0));
	// Assert.assertEquals(price, 0, 1e-20);
}

func TestBSModel(t *testing.T) {
	t.Run("recovery", func(t *testing.T) {
		defer func() {
			if err := recover(); err == nil {
				t.Error("no panic")
			}
		}()
		NewBSModelWithTime(OptionTypePut, 0, 1634889600, 0)
	})

	t.Run("recovery func", func(t *testing.T) {
		defer func() {
			if err := recover(); err == nil {
				t.Error("no panic")
			}
		}()
		_ = newOptionFunction(3, 0)
	})

	t.Run("recovery setup", func(t *testing.T) {
		defer func() {
			if err := recover(); err == nil {
				t.Error("no panic")
			}
		}()
		bs := NewBSModel(OptionTypePut, 0)
		_ = bs.Setup(0, 0)
	})

	t.Run("basic", func(t *testing.T) {
		bs := NewBSModel(OptionTypePut, 0)
		bs.SetMaxIterations(1)
		bs.SetMaxIterations(0)
		t.Log(bs.MaxIterations())
		bs.SetAccuracy(1)
		t.Log(bs.Accuracy())
		t.Log(bs.StrikePrice())
		bs.SetUnderlying(0)
		bs.SetStrikePrice(0)
		_, _ = bs.Delta(0)
		_, _ = bs.Gamma(0)
		_, _ = bs.Theta(0)
		_, _ = bs.Vega(0)
		_, _ = bs.Rho(0)
		_, _ = bs.Greeks(0)
		_ = bs.CalTimeRate(1, 0)
		_, _ = bs.CalOptionValue(0)
		_, _ = bs.CalStrikePrice(0, 1)
		_, _ = bs.CalStrikePrice(1, 0)
		_, _ = bs.CalImpliedVolatility(0)
	})

	t.Run("baseFunction", func(t *testing.T) {
		pf := newPutFunction(1).(*putFunction)
		t.Log(pf.Underlying())
		t.Log(pf.StrikePrice())
		t.Log(pf.Derivative(0))
		t.Log(pf.vTheta(0, 0, 0))
		cf := newCallFunction(1).(*callFunction)
		t.Log(cf.vTheta(0, 0, 0))
		cf.SetStrikePrice(1)
		cf.SetUnderlying(0)
		t.Log(cf.vCalOptionValueBound())
		cf.SetStrikePrice(0)
		cf.SetUnderlying(1)
		t.Log(cf.vCalOptionValueBound())
	})

	t.Run("math", func(t *testing.T) {
		_ = maxInt64(0, 1)
		_ = maxInt64(1, 0)
		_ = sndEvaluate(nil)
		_ = sndCDN(0)
	})
}

// BenchmarkXxx/erfInverseCND
//
// BenchmarkXxx/erfInverseCND-8         	270443966	         4.442 ns/op	       0 B/op	       0 allocs/op
// BenchmarkXxx/evaluate
// BenchmarkXxx/evaluate-8              	 1626426	      1021 ns/op	       0 B/op	       0 allocs/op
// BenchmarkXxx/put_optionvalue_time
// BenchmarkXxx/put_optionvalue_time-8  	36497690	        32.19 ns/op	       0 B/op	       0 allocs/op
// BenchmarkXxx/put_iv_time
// BenchmarkXxx/put_iv_time-8           	 3781230	       318.1 ns/op	       0 B/op	       0 allocs/op
// BenchmarkXxx/put_greeks_time
// BenchmarkXxx/put_greeks_time-8       	21847526	        53.75 ns/op	      48 B/op	       1 allocs/op
// BenchmarkXxx/call_optionvalue_time
// BenchmarkXxx/call_optionvalue_time-8 	38216661	        32.87 ns/op	       0 B/op	       0 allocs/op
// BenchmarkXxx/call_iv_time
// BenchmarkXxx/call_iv_time-8          	 3724519	       317.8 ns/op	       0 B/op	       0 allocs/op
// BenchmarkXxx/call_greeks_time
// BenchmarkXxx/call_greeks_time-8      	22469121	        53.00 ns/op	      48 B/op	       1 allocs/op
// BenchmarkXxx/hv_time
// BenchmarkXxx/hv_time-8               	  931677	      1256 ns/op	    3072 B/op	       1 allocs/op
func BenchmarkXxx(b *testing.B) {
	b.Run("erfInverseCND", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sndErfInverseCND(0.5197244846780518)
		}
	})

	b.Run("evaluate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sndEvaluate(array)
		}
	})

	putModel.CalTimeRate(getTime("2021/06/09 09:49:17"), getTime("2021/06/18 08:00:00"))
	putModel.Setup(34277.23, 30000)

	b.Run("put optionvalue time", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = putModel.CalOptionValue(2.943)
		}
	})

	b.Run("put iv time", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = putModel.CalImpliedVolatility(3948.79)
		}
	})

	b.Run("put greeks time", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = putModel.Greeks(2.943)
		}
	})

	// callModel
	callModel.CalTimeRate(getTime("2021/06/09 09:49:17"), getTime("2021/06/18 08:00:00"))
	callModel.Setup(34277.23, 30000)

	b.Run("call optionvalue time", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = callModel.CalOptionValue(2.943)
		}
	})

	b.Run("call iv time", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = callModel.CalImpliedVolatility(8226.02)
		}
	})

	b.Run("call greeks time", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = callModel.Greeks(2.943)
		}
	})

	//
	b.Run("hv time", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = CalHistoricalVolatility(array)
		}
	})
}
