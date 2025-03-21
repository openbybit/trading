package gformula

import (
	"fmt"
	"math"
)

func newOptionFunction(typ OptionType, interestRate float64) optionFunction {
	switch typ {
	case OptionTypePut:
		return newPutFunction(interestRate)
	case OptionTypeCall:
		return newCallFunction(interestRate)
	default:
		panic(fmt.Errorf("invalid option type: %v", typ))
	}
}

type optionFunction interface {
	DeltaDecayCoefficient() float64
	OptionValue() float64
	SetOptionValue(v float64)
	Underlying() float64
	SetUnderlying(underlying float64)
	StrikePrice() float64
	SetStrikePrice(strikePrice float64)

	CalDelta(volatility float64) float64
	CalGamma(volatility float64) float64
	CalTheta(volatility float64) float64
	CalVega(volatility float64) float64
	CalRho(volatility float64) float64
	CalGreeks(volatility float64) *GreekResult

	// BlackSholesVega 期权定价公式的导数
	BlackSholesVega(volatility float64) float64
	/**
	 * <p>计算期权价格的最大最小范围.</p>
	 * <p>call期权价格最大不超过标的价格</p>
	 * <p>put期权价格最大不超过行权价格</p>
	 * <p>虚值/平值期权最低价格为0，实值期权最小价格为abs(underlying_-strkeprice)</p>
	 * */
	CalOptionValueBound() [2]float64
	/**
	 * 期权价格.
	 *
	 * @param volatility the volatility
	 * @return the double
	 */
	CalOptionPrice(volatility float64) float64
	/**
	 * 根据delta/volatility计算行权价格
	 *
	 * @param delta      the delta
	 * @param volatility the volatility
	 * @return the double
	 */
	CalStrikePrice(delta, volatility float64) float64
	/**
	 * <p>计算时间相关参数.</p>
	 * <p>最小时间跨度为1秒</p>
	 *
	 * @param time_to_expire 当前时间与到期时间相差秒数
	 */
	CalTimeRate(timeToExpire int64)

	/**
	 * 预估波动率初始值.
	 * 按照bs模型特性，起始位置选择过小或过大，都会导致iv无解；
	 * 用二分查找寻找起始点，
	 * 合理的起始位置应该满足以下条件：
	 *   1. 对应点方程的导数不能为零，即斜率不能太小
	 *   2. 导数斜线与X轴的交点应该为正，负数在bs模型中没有意义，并有可能导致牛顿迭代无法收敛.
	 * @param min_option 最新期权价格
	 * @param max_option 最大期权价格
	 * @param max_iterations 最大尝试次数
	 * @return 预估的起始值
	 */
	EstimateVolatilityStart(minOption, maxOption float64, maxIterations int) float64
}

type virtualMethodTable interface {
	vDelta(d1 float64) float64
	vTheta(volatility, expd1, cnd2 float64) float64
	vRho(cnd2 float64) float64
	vCalOptionPrice(volatility float64) float64
	vCalOptionValueBound() [2]float64
}

type baseFunction struct {
	optionValue      float64 // 期权价格
	underlying       float64 // 标的价格
	strikePrice      float64 // 行权价
	strikeFactor     float64 // 行权价参数
	interestRate     float64 // 连续复合无风险利率，小数
	timeToExpiration float64 // 年化到期时间
	timeFactor       float64 // 到期时间参数
	//
	deltaDecayCoefficient float64 // delta随到期日临近的衰减系数,(0,1], UTC 07:30:00~08:00:00 期间 delta修正计算
	thetaDecayCoefficient float64 // theta随到期日临近的衰减系数，(0, 1], 默认到期日24小时内theta修正计算
	payoffUnit            float64 // 收益单位

	//
	globalAccuracy     float64
	derivativeAccuracy float64

	vmt virtualMethodTable
}

func (f *baseFunction) init(interestRate float64) {
	f.interestRate = interestRate
	f.globalAccuracy = globalConfig.FormulaAccuracy
	f.derivativeAccuracy = globalConfig.FormulaDerivativeAccuracy
	f.deltaDecayCoefficient = 1
	f.thetaDecayCoefficient = 1
	f.payoffUnit = 1
}

func (f *baseFunction) DeltaDecayCoefficient() float64 {
	return f.deltaDecayCoefficient
}

func (f *baseFunction) OptionValue() float64 {
	return f.optionValue
}

func (f *baseFunction) SetOptionValue(v float64) {
	f.optionValue = v
}

func (f *baseFunction) Underlying() float64 {
	return f.underlying
}

func (f *baseFunction) SetUnderlying(underlying float64) {
	f.underlying = underlying
}

func (f *baseFunction) StrikePrice() float64 {
	return f.strikePrice
}

func (f *baseFunction) SetStrikePrice(strikePrice float64) {
	f.strikePrice = strikePrice
	f.strikeFactor = strikePrice * float64(f.payoffUnit)
}

func (f *baseFunction) FormulaPrice(volatility float64) float64 {
	d := f.DPlusMinus(volatility)
	return f.underlying*sndErfCND(d[0]) - f.strikeFactor*sndErfCND(d[1])
}

// DPlusMinus 计算d1和d2
func (f *baseFunction) DPlusMinus(volatility float64) [2]float64 {
	var res [2]float64
	factor := f.CalFactor(volatility)
	d1 := f.CalD1(volatility, factor)
	d2 := f.CalD2(d1, factor)

	res[0] = d1
	res[1] = d2
	return res
}

// Derivative BS公式对波动率的一阶导数
func (f *baseFunction) Derivative(d1 float64) float64 {
	return f.underlying * f.timeFactor * f.ExpD1(d1)
}

func (f *baseFunction) ExpD1(d1 float64) float64 {
	return kDivPi2 * math.Exp(-0.5*d1*d1)
}

func (f *baseFunction) Delta(d1 float64) float64 {
	return f.vmt.vDelta(d1)
}

func (f *baseFunction) Gamma(volatility, expd1 float64) float64 {
	return 1.0 / (f.underlying * volatility * f.timeFactor) * expd1
}

func (f *baseFunction) Theta(volatility, expd1, cnd2 float64) float64 {
	return f.vmt.vTheta(volatility, expd1, cnd2)
}

func (f *baseFunction) Vega(expd1 float64) float64 {
	return kCenti * f.underlying * f.timeFactor * expd1
}

func (f *baseFunction) Rho(cnd2 float64) float64 {
	return f.vmt.vRho(cnd2)
}

func (f *baseFunction) AdjustDelta(delta float64) float64 {
	return delta * f.deltaDecayCoefficient
}

func (f *baseFunction) AdjustGamma(gamma float64) float64 {
	return f.thetaDecayCoefficient * gamma
}

func (f *baseFunction) AdjustTheta(theta float64) float64 {
	return f.thetaDecayCoefficient * theta
}

/**
 * <p>计算d1.</p>
 * <p>d1描述期权对标的价格的敏感程度，N(d1)是在风险中性测度下，按标的价格加权得到的期权被执行概率</p>
 *
 * @param underlying  标的价格
 * @param strikePrice 行权价
 * @param volatility  波动率
 * @param factor      系数来自calFactor(volatility)
 * @return the double d1
 */
func (f *baseFunction) CalD1(volatility, factor float64) float64 {
	return (math.Log(f.underlying/f.strikePrice) + (f.interestRate+0.5*volatility*volatility)*f.timeToExpiration) / (factor)
}

/**
 * <p>计算d2.</p>
 * <p>d2描述期权被执行的可能性，N(d2)是在风险中性条件下，不按标的价格加权得到的期权被执行概率</p>
 *
 * @param d1     the d1
 * @param factor the factor
 * @return the double d2
 */
func (f *baseFunction) CalD2(d1, factor float64) float64 {
	return d1 - factor
}

/**
 * Cal factor.
 *
 * @param volatility the volatility
 * @return the double
 */
func (f *baseFunction) CalFactor(volatility float64) float64 {
	return volatility * f.timeFactor
}

func (f *baseFunction) CalRho(volatility float64) float64 {
	factor := f.CalFactor(volatility)
	d1 := f.CalD1(volatility, factor)
	return f.Rho(sndErfCND(f.CalD2(d1, factor)))
}

/**
 * 计算delta
 *
 * @param volatility the volatility
 * @return the double
 */
func (f *baseFunction) CalDelta(volatility float64) float64 {
	return f.AdjustDelta(f.Delta(f.CalD1(volatility, f.CalFactor(volatility))))
}

/**
* 计算gamma.
*
* @param volatility    the volatility
* @return the double
 */
func (f *baseFunction) CalGamma(volatility float64) float64 {
	return f.AdjustGamma(f.Gamma(volatility, f.ExpD1(f.CalD1(volatility, f.CalFactor(volatility)))))
}

/**
* 计算theta
*
* @param volatility the volatility
* @return the double
 */
func (f *baseFunction) CalTheta(volatility float64) float64 {
	d := f.DPlusMinus(volatility)
	return f.AdjustTheta(f.Theta(volatility, f.ExpD1(d[0]), sndErfCND(d[1])))
}

func (f *baseFunction) CalVega(volatility float64) float64 {
	return f.Vega(f.ExpD1(f.CalD1(volatility, f.CalFactor(volatility))))
}

func (f *baseFunction) CalOptionPrice(volatility float64) float64 {
	return f.vmt.vCalOptionPrice(volatility)
}

func (f *baseFunction) CalStrikePrice(delta, volatility float64) float64 {
	return f.underlying / math.Exp(sndErfInverseCND(delta)*volatility*f.timeFactor-f.timeToExpiration*(f.interestRate+0.5*volatility*volatility))
}

func (f *baseFunction) CalOptionValueBound() [2]float64 {
	return f.vmt.vCalOptionValueBound()
}

func (f *baseFunction) CalTimeRate(timeToExpire int64) {
	timeSpan := maxInt64(timeToExpire, 1)
	f.timeToExpiration = float64(timeSpan) * float64(kYearFactor) / float64(kOneDaySec)
	// exp(x)函数返回e的x次幂,此处计算连续复利
	f.payoffUnit = math.Exp(-f.interestRate * f.timeToExpiration)
	f.timeFactor = math.Sqrt(f.timeToExpiration)
	// 交割日最后结算周期，默认为最后半小时，即1800秒
	globalSpan := globalConfig.FormulaSettlePeriod
	// delta随到期日临近的衰减系数,(0,1]
	f.deltaDecayCoefficient = float64(minInt64(timeSpan, int64(globalSpan))) * 1.0 / float64(globalSpan)
	// 交割日当天，默认为最后24小时，即86400秒
	oneDay := globalConfig.FormulaLastDate
	// theta随到期日临近的衰减系数，(0, 1]
	f.thetaDecayCoefficient = float64(minInt64(timeSpan, oneDay)) * 1.0 / float64(oneDay)
	// 行权价参数
	f.strikeFactor = f.strikePrice * float64(f.payoffUnit)
}

func (f *baseFunction) CalGreeks(volatility float64) *GreekResult {
	d := f.DPlusMinus(volatility)

	var result GreekResult
	result.Delta = f.AdjustDelta(f.Delta(d[0]))

	/**正常用公式gamma*/
	expd1 := f.ExpD1(d[0])
	result.Gamma = f.AdjustGamma(f.Gamma(volatility, expd1))

	cnd2 := sndErfCND(d[1])
	result.Theta = f.AdjustTheta(f.Theta(volatility, expd1, cnd2))
	result.Vega = f.Vega(expd1)
	result.Rho = f.Rho(cnd2)

	return &result
}

func (f *baseFunction) BlackSholesVega(volatility float64) float64 {
	factor := f.CalFactor(volatility)
	d1 := f.CalD1(volatility, factor)
	nd1 := math.Exp(-d1*d1/2) / kSqrtPi2
	vega := f.underlying * nd1 * f.timeFactor
	return vega
}

func (f *baseFunction) EstimateVolatilityStart(minOption, maxOption float64, maxIterations int) float64 {
	forward := f.underlying / float64(f.payoffUnit)
	q := 2.0 * math.Abs(math.Log(forward/f.strikePrice)) / f.timeToExpiration
	// 判断是否需要使用精度修正,小于精度，则将其修正为1。
	if q < f.globalAccuracy {
		q = 1
	}
	// 计算起始值
	start := 0.5 * math.Sqrt(q)
	// 计算中间值 minOption和maxOption分别是期权价格的最大最小范围
	midOption := 0.5*(minOption+maxOption) - f.optionValue
	for cnt := 0; cnt < maxIterations; cnt++ {
		// 计算期权价格误差 CalOptionPrice表示使用BS公式计算期权价格
		startValue := f.CalOptionPrice(start) - f.optionValue
		// 使用BS公式计算期权价格对波动率的一阶导数，即Vega
		// start 表示隐含波动率。derivative 表示Vega
		derivative := f.BlackSholesVega(start)

		next := start - startValue/derivative
		// 如果计算的导数>全局初始求导精度,说明计算的结果可以参与下一轮的计算
		// 尝试寻找下一个候选初始点，它应该为正数，同时对应导数不能为0
		if derivative > f.derivativeAccuracy && (next >= 0 && f.BlackSholesVega(next) > f.derivativeAccuracy) {
			return next
		} else if startValue > midOption {
			start *= 0.5
		} else {
			start *= 1.5
		}
	}
	//   LOG_WARN("start derivative too many iterations {}, return {},info[{},{},{},{},{}]", cnt, start, underlying_,strike_price_, option_value_, min_option, max_option);
	return start
}
