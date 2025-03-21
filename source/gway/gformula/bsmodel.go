package gformula

import (
	"errors"
	"fmt"
	"math"
)

var (
	ErrMustBePositive = errors.New("must be positive")
)

type OptionType uint8

const (
	OptionTypeCall = OptionType(1)
	OptionTypePut  = OptionType(2)
)

func NewBSModel(typ OptionType, interestRate float64) BSModel {
	f := newOptionFunction(typ, interestRate)
	bs := BSModel{
		function:      f,
		maxIterations: int(globalConfig.FormulaMaxIterations),
		accuracy:      globalConfig.FormulaAccuracy,
	}
	return bs
}

func NewBSModelWithTime(typ OptionType, interestRate float64, currentTimeSec, expirationSec int64) BSModel {
	bs := NewBSModel(typ, interestRate)
	if err := bs.CalTimeRate(currentTimeSec, expirationSec); err != nil {
		panic(err)
	}

	return bs
}

/**
 * <p>提供期权核心指标计算入口</p>
 * <ul>
 * <li>1. 期权理论价格</li>
 * <li>2. 隐含波动率</li>
 * <li>3. 历史波动率</li>
 * <li>4. 希腊字母（delta/gamma/vega/theta/rho）</li>
 * <li>5. 行权价。</li>
 * </ul>
 * <p>BS模型基于以下要素构建：</p>
 * <ul>
 * <li>a. 到期时间</li>
 * <li>b. 标的价格</li>
 * <li>c. 行权价格</li>
 * <li>d. 连续复合无风险利率</li>
 * <li>e. 波动率</li>
 * <li>f. 期权价格</li>
 * </ul>
 * <p>通常到期时间、标的价格、行权价格、利率是确定的，通过波动率可计算期权价格，反之由期权价格可计算波动率；
 * 特殊情况下，确定到期时间、标的价格、利率，通过波动率和希腊字母反向求解行权价格。</p>
 */
type BSModel struct {
	function      optionFunction
	maxIterations int
	accuracy      float64
}

func (bsm *BSModel) MaxIterations() int {
	return bsm.maxIterations
}

func (bsm *BSModel) SetMaxIterations(maxIterations int) {
	if maxIterations > 0 {
		bsm.maxIterations = maxIterations
	} else {
		bsm.maxIterations = 1
	}
}

func (bsm *BSModel) Accuracy() float64 {
	return bsm.accuracy
}

func (bsm *BSModel) SetAccuracy(accuracy float64) {
	bsm.accuracy = accuracy
}

func (bsm *BSModel) Underlying() float64 {
	return bsm.function.Underlying()
}

func (bsm *BSModel) SetUnderlying(underlying float64) error {
	if err := bsm.checkPositive(underlying, "underlying"); err != nil {
		return err
	}
	bsm.function.SetUnderlying(underlying)
	return nil
}

func (bsm *BSModel) StrikePrice() float64 {
	return bsm.function.StrikePrice()
}

func (bsm *BSModel) SetStrikePrice(strikePrice float64) error {
	if err := bsm.checkPositive(strikePrice, "strikePrice"); err != nil {
		return err
	}
	bsm.function.SetStrikePrice(strikePrice)
	return nil
}

func (bsm *BSModel) Setup(underlying float64, strikePrice float64) *BSModel {
	err1 := bsm.SetUnderlying(underlying)
	err2 := bsm.SetStrikePrice(strikePrice)
	if err1 != nil || err2 != nil {
		panic(fmt.Errorf("has some error, err1: %v, err2 : %v", err1, err2))
	}

	return bsm
}

/**
 * <p>计算delta。</p>
 * <p>delta又称对冲值，是衡量标的资产价格变动时期权价格的变化幅度，取值范围(-1,1)。</p>
 * <p>临近到期时间时，delta在理论值的基础上会做进一步修正</p>
 * <p>call delta一定为正，put delta一定为负</p>
 * <p>volatility需要为正数，否则抛IllegalArgumentException异常</p>
 * <p>前提：设置underlying和strike price</p>
 *
 * @param volatility 波动率，正数
 * @return the double
 */
func (bsm *BSModel) Delta(volatility float64) (float64, error) {
	if err := bsm.checkPositive(volatility, "volatility"); err != nil {
		return 0, err
	}

	delta := bsm.function.CalDelta(volatility)
	return delta, nil
}

/**
 * <p>计算gamma.</p>
 * <p>衡量标的价格对delta值的影响程度，gamma值越大，delta变化越快，取值范围(0,1).</p>
 * <p>isDecayAdjusted判断是否临近到期时间，</p>
 * <p>临近到期时间(默认最后半小时，可通过GlobalConfig修改)时gamma采用特殊计算方式，依赖上一次计算的delta,
 * gamma和underlying。</p> <p>正常时候functionState可以为null.</p>
 * <p>通过functionState把依赖的数据传入方程,临近到期时间的functionState==null时，gamma计算将抛异常.</p>
 * <p>前提：设置underlying和strike price</p>
 *
 * @param volatility    波动率，正数
 * @return the double
 */
func (bsm *BSModel) Gamma(volatility float64) (float64, error) {
	if err := bsm.checkPositive(volatility, "volatility"); err != nil {
		return 0, err
	}

	gamma := bsm.function.CalGamma(volatility)
	return gamma, nil
}

/**
 * <p>计算theta.</p>
 * <p>theta衡量时间变化对期权价格的影响程度，表示每过一天期权价格的减少量。为负数。</p>
 * <p>到期时间越长，期权价格越高，反之，随着时间流逝期权价格不断下降，越临近到期日，价格损失越大。</p>
 * <p>前提：设置underlying和strike price</p>
 *
 * @param volatility 波动率，正数
 * @return the double
 */
func (bsm *BSModel) Theta(volatility float64) (float64, error) {
	if err := bsm.checkPositive(volatility, "volatility"); err != nil {
		return 0, err
	}

	theta := bsm.function.CalTheta(volatility)
	return theta, nil
}

/**
 * <p>计算vega.</p>
 * <p>vega衡量标的价格波动率的变化对期权价格的影响，波动率变化一个单位为1%</p>
 * <p>为正值，值越大，表示对波动率变化的风险越大。</p>
 *
 * <p>前提：设置underlying和strike price</p>
 *
 * @param volatility 波动率，正数
 * @return the double
 */
func (bsm *BSModel) Vega(volatility float64) (float64, error) {
	if err := bsm.checkPositive(volatility, "volatility"); err != nil {
		return 0, err
	}

	vega := bsm.function.CalVega(volatility)
	return vega, nil
}

/**
 * <p>计算rho.</p>
 * <p>rho衡量无风险利率变化对期权价格的影响，利率变化一个单位为1%</p>
 * <p>call rho为正，put rho为负</p>
 *
 * <p>前提：设置underlying和strike price</p>
 *
 * @param volatility 波动率，正数
 * @return the double
 */
func (bsm *BSModel) Rho(volatility float64) (float64, error) {
	if err := bsm.checkPositive(volatility, "volatility"); err != nil {
		return 0, err
	}

	rho := bsm.function.CalRho(volatility)
	return rho, nil
}

/**
 * <p>计算全量希腊字母</p>
 * <p>volatility需要为正数，否则抛IllegalArgumentException异常。</p>
 * <p>isDecayAdjusted判断是否临近到期时间，</p>
 * <p>临近到期时间(默认最后半小时，可通过GlobalConfig修改)时，delta会做修正，gamma采用特殊计算方式，依赖上一次计算的delta,
 * gamma和underlying。</p> <p>正常时候functionState可以为null.</p>
 * <p>通过functionState把依赖的数据传入方程,临近到期时间的functionState==null时，gamma计算将抛异常.</p>
 *
 * <p>前提：设置underlying和strike price</p>
 *
 * @param volatility 波动率，正数
 * @return 全量希腊字母
 */
func (bsm *BSModel) Greeks(volatility float64) (*GreekResult, error) {
	if err := bsm.checkPositive(volatility, "volatility"); err != nil {
		return nil, err
	}

	greeks := bsm.function.CalGreeks(volatility)
	return greeks, nil
}

/**
 * <p>bs模型是否要做时间衰减校正，BSModel创建或timeRate执行后可判断。</p>
 * <p>临近到期时间，即与到期时间之差小于GlobalConfig.FORMULA_SETTLE_PERIOD的设置时，greeks需做校正，</p>
 * <p>校正时，对gamma的求解需要设置lastState（通常为上一秒）的FunctionState.</p>
 *
 * @return the boolean
 */
func (bsm *BSModel) IsDecayAdjusted() bool {
	return bsm.function.DeltaDecayCoefficient() < 1
}

func (bsm *BSModel) CalTimeRate(currentTimeSec, expirationSec int64) error {
	if currentTimeSec > expirationSec {
		return fmt.Errorf("current time is greater than expiration[%d,%d]", currentTimeSec, expirationSec)
	}
	bsm.function.CalTimeRate(expirationSec - currentTimeSec)
	return nil
}

/**
 * <p>理论期权价格</p>
 * <p>volatility需要正数，否则抛IllegalArgumentException异常</p>
 * <p>前提：设置underlying和strike price</p>
 *
 * @param volatility 波动率，正数
 * @return the double
 */
func (bsm *BSModel) CalOptionValue(volatility float64) (float64, error) {
	if err := bsm.checkPositive(volatility, "volatility"); err != nil {
		return 0, err
	}
	return math.Max(bsm.function.CalOptionPrice(volatility), 0.0), nil
}

func (bsm *BSModel) CalStrikePrice(delta, volatility float64) (float64, error) {
	if err := bsm.checkPositive(volatility, "volatility"); err != nil {
		return 0, err
	}

	if delta < 0 {
		delta += 1
	}

	if err := bsm.checkPositive(delta, "delta"); err != nil {
		return 0, err
	}

	if delta >= 1 {
		return 0, fmt.Errorf("delta must be less than 1")
	}

	return bsm.function.CalStrikePrice(delta, volatility), nil
}

/**
 * <p>计算隐含波动率.</p>
 * <p>没有精确解，使用牛顿法迭代求数值解，</p>
 * <p>最大迭代次数maxIterations和计算精度accuracy将影响计算结果，</p>
 * <p>精度要求越高，需要迭代的次数越多。但牛顿法的收敛速度很快，通常迭代20次以内就有很高精度的结果</p>
 * <p>当输入的期权价格低于理论最低价时，隐含波动率无解，返回0；高于理论最高价时，隐含波动率无解，返回1e9</p>
 * <p>当前价格需要正数，否则抛IllegalArgumentException异常</p>
 * <p>前提：设置underlying和strike price</p>
 *
 * @param currentPrice 当前期权价格，正数
 * @return 隐含波动率，非负数
 */
func (bsm *BSModel) CalImpliedVolatility(currentPrice float64) (float64, error) {
	if err := bsm.checkPositive(currentPrice, "option price"); err != nil {
		return 0, err
	}
	fn := bsm.function

	optionBound := fn.CalOptionValueBound()
	if currentPrice < optionBound[0]+bsm.accuracy {
		return 0, nil
	}

	if currentPrice > optionBound[1]-bsm.accuracy {
		return kMaxVolatility, nil
	}

	fn.SetOptionValue(currentPrice)
	start := fn.EstimateVolatilityStart(optionBound[0], optionBound[1], bsm.maxIterations)
	iv := math.Max(solveNewtonRaphson(fn, start, bsm.accuracy, bsm.maxIterations), 0)
	return iv, nil
}

func (bsm *BSModel) checkPositive(x float64, msg string) error {
	if x <= 0 {
		return fmt.Errorf("%s must be positive", msg)
	}

	return nil
}

/**
 * <p>计算年化历史波动率</p>
 * <p>indexArray是一个长度至少为2，且元素不能有零值的数组</p>
 * <p>否则抛InvalidArrayException异常</p>
 *
 * @param index_array 历史指数价格，按时间从旧到新排序
 * @return 历史波动率，非负数
 */
func CalHistoricalVolatility(indexArray []float64) (float64, error) {
	if len(indexArray) < 2 {
		return 0, fmt.Errorf("The length must be greater than 2")
	}

	kProfitRateLength := len(indexArray) - 1
	profitRateVector := make([]float64, 0, kProfitRateLength)

	for i := 0; i < kProfitRateLength; i++ {
		if indexArray[i] <= 0 {
			return 0, fmt.Errorf("index data must be greater than or equal to 0")
		}

		profitRateVector = append(profitRateVector, indexArray[i+1]/indexArray[i]-1)
	}

	result := sndEvaluate(profitRateVector) * kYearDayFactor

	return result, nil
}
