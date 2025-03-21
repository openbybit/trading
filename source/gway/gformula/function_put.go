package gformula

import "math"

func newPutFunction(interestRate float64) optionFunction {
	f := &putFunction{}
	f.init(interestRate)
	f.vmt = f
	return f
}

type putFunction struct {
	baseFunction
}

func (f *putFunction) vDelta(d1 float64) float64 {
	return sndErfCND(d1) - 1
}

func (f *putFunction) vTheta(volatility, expd1, cnd2 float64) float64 {
	t := -kYearFactor * 0.5 * f.underlying * volatility / f.timeFactor * expd1

	if math.Abs(f.interestRate) < f.globalAccuracy {
		return t
	}

	return t + kYearFactor*f.interestRate*f.strikeFactor*cnd2
}

func (f *putFunction) vRho(cnd2 float64) float64 {
	return kCenti * f.strikeFactor * f.timeToExpiration * (cnd2 - 1)
}

func (f *putFunction) vCalOptionPrice(volatility float64) float64 {
	return f.strikeFactor - f.underlying + f.FormulaPrice(volatility)
}

func (f *putFunction) vCalOptionValueBound() [2]float64 {
	var res [2]float64
	if f.strikePrice < f.underlying {
		res[0] = 0
		res[1] = f.strikePrice
	} else {
		res[0] = f.strikePrice - f.underlying
		res[1] = f.strikePrice
	}

	return res
}
