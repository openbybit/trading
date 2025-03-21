package gformula

import "math"

var (
	kSqrt2                     = math.Sqrt(2.0)
	kSqrtPi2                   = math.Sqrt(2 * math.Pi)
	kDivPi2                    = 1.0 / math.Sqrt(2*math.Pi)
	kCenti                     = 0.01
	kOneDaySec                 = 24 * 3600
	kYearFactor                = float64(1.0 / 365)
	kSqrtDay                   = math.Sqrt(24)
	kYearDayFactor             = kSqrtDay * math.Sqrt(365)
	kDefaultIterations         = 100
	kDefaultAccuracy           = 1e-6
	kDefaultDerivativeAccuracy = 0.001
	kMaxVolatility             = 1e9
	kHalfHourSecs              = 1800
)
