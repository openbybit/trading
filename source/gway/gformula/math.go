package gformula

import "math"

func maxInt64(x, y int64) int64 {
	if x < y {
		return y
	}
	return x
}

func minInt64(x, y int64) int64 {
	if x > y {
		return y
	}
	return x
}

// 使用NewtonRaphson方法计算期权的隐含波动率
func solveNewtonRaphson(optionFn optionFunction, startValue float64, absoluteAccuracy float64, maxIterations int) float64 {
	x0 := startValue
	x1 := float64(0.0)

	for i := 0; i < maxIterations; i++ {
		derivative := optionFn.BlackSholesVega(x0)
		if math.Abs(derivative) < absoluteAccuracy {
			return 0
		}

		lastOptionValue := optionFn.OptionValue()
		x1 = x0 - (optionFn.CalOptionPrice(x0)-lastOptionValue)/derivative

		if math.Abs(x1-x0) <= absoluteAccuracy {
			return x1
		}
		x0 = x1
	}

	return x0
}

// standard normal distribution
/**
* LaTeX书写标准差公式如下
* \sigma = \sqrt{\frac{1}{N}\sum_{i=1}^{N}(x_i-\mu)^2}
* $\sigma$ 表示标准差
* $N$ 表示样本数量
* $x_i$ 表示第 $i$ 个观测值
* $\mu$ 表示样本的均值
* 对所有差值的平方求和，得到 $\sum_{i=1}^{N}(x_i-\mu)^2$。
* frac{1}{N} ：输出的结果将是一个垂直居中的分数，其中分子为 1，分母为 N
* @param data  需要计算标准差的数组
* @param size  数组元素个数
* @return 标准差
 */
func sndEvaluate(profitRateVector []float64) float64 {
	size := len(profitRateVector)
	if size <= 1 {
		return 0
	}

	var sum float64
	for i := 0; i < size; i++ {
		sum += profitRateVector[i]
	}

	mean := sum / float64(size)
	bias := float64(0.0)
	var diff float64
	for i := 0; i < size; i++ {
		// 对于每个数据点 $x_i$，计算其与样本均值 $\mu$ 的差值 $(x_i-\mu)$。
		diff = profitRateVector[i] - mean
		// 对所有差值的平方求和，得到 $\sum_{i=1}^{N}(x_i-\mu)^2$。
		bias += diff * diff
	}

	// 通过将自由度减去 $1$，可以让样本方差的计算结果更加准确，更接近于总体方差的真实值
	bias /= (float64(size) - 1)
	// 返回标准差
	return math.Sqrt(bias)
}

func sndCDN(X float64) float64 {
	a1 := 0.31938153
	a2 := -0.356563782
	a3 := 1.781477937
	a4 := -1.821255978
	a5 := 1.330274429

	var L, K, w float64

	L = math.Abs(X)
	K = 1.0 / (1.0 + 0.2316419*L)
	w = 1.0 - kDivPi2*math.Exp(-L*L/2)*(a1*K+a2*K*K+a3*math.Pow(K, 3)+a4*math.Pow(K, 4)+a5*math.Pow(K, 5))

	if X < 0.0 {
		w = 1.0 - w
	}
	return w
}

// 求标准正态分布累计函数，精度最高，速度最慢
func sndErfCND(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt2))
}

// 根据标准正态分布累计函数结果反向求解输入值
func sndErfInverseCND(x float64) float64 {
	return math.Sqrt2 * math.Erfinv(2*x-1)
}

// 累计函数，用自然指数与多项式拟合.精度不够,速度最快
func sndPageCND(x float64) float64 {
	const (
		a1 = 0.070565992
		a2 = 1.5976
	)

	return 1.0 - 1.0/(1+math.Exp(a1*math.Pow(x, 3)+a2*x))
}
