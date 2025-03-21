package gformula

import "math"

func newCallFunction(interestRate float64) optionFunction {
	f := &callFunction{}
	f.init(interestRate)
	f.vmt = f
	return f
}

type callFunction struct {
	baseFunction
}

func (f *callFunction) vDelta(d1 float64) float64 {
	return sndErfCND(d1)
}

func (f *callFunction) vTheta(volatility, expd1, cnd2 float64) float64 {
	t := -kYearFactor * 0.5 * f.underlying * volatility / f.timeFactor * expd1
	if math.Abs(f.interestRate) < f.globalAccuracy {
		return t
	}

	return (t - kYearFactor*f.interestRate*f.strikeFactor*cnd2)
}

func (f *callFunction) vRho(cnd2 float64) float64 {
	return kCenti * f.strikeFactor * f.timeToExpiration * cnd2
}

func (f *callFunction) vCalOptionPrice(volatility float64) float64 {
	return f.FormulaPrice(volatility)
}

func (f *callFunction) vCalOptionValueBound() [2]float64 {
	var res [2]float64
	if f.strikePrice > f.underlying {
		res[0] = 0
		res[1] = f.underlying
	} else {
		res[0] = f.underlying - f.strikePrice
		res[1] = f.underlying
	}

	return res
}
