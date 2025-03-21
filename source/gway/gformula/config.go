package gformula

var globalConfig = Config{
	FormulaSettlePeriod:       int32(kHalfHourSecs),
	FormulaLastDate:           int64(kOneDaySec),
	FormulaMaxIterations:      int32(kDefaultIterations),
	FormulaAccuracy:           kDefaultAccuracy,
	FormulaDerivativeAccuracy: kDefaultDerivativeAccuracy,
}

type Config struct {
	FormulaSettlePeriod       int32
	FormulaLastDate           int64
	FormulaMaxIterations      int32
	FormulaAccuracy           float64
	FormulaDerivativeAccuracy float64
}
