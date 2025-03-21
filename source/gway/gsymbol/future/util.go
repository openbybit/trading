package future

import (
	"math"

	"code.bydev.io/lib/gopkg/bdecimal.git"
)

func stringFromEN(f int64, scale int32, friction int32) (string, error) {
	intDec := bdecimal.NewDecFromInt(f)
	resEN, err := bdecimal.NewDecFromFloat(0)
	if err != nil {
		return "", err
	}
	ENDec, err := bdecimal.NewDecFromFloat(math.Pow10(int(scale)))
	if err != nil {
		return "", err
	}
	err = bdecimal.DecimalDiv(intDec, ENDec, resEN, int(friction))
	if err != nil {
		return "", err
	}
	return resEN.String(), err
}

func stringToEN(s string, n int32) (int64, error) {
	strDec, err := bdecimal.NewDecFromString(s)
	if err != nil {
		return 0, err
	}
	err = strDec.Shift(int(n))
	if err != nil {
		return 0, err
	}
	intEN, err := strDec.ToInt()
	if err != nil {
		return 0, err
	}
	return intEN, nil
}

func int64ToEN(i int64, n int32) (int64, error) {
	intDec := bdecimal.NewDecFromInt(i)

	err := intDec.Shift(int(n))
	if err != nil {
		return 0, err
	}
	intEN, err := intDec.ToInt()
	if err != nil {
		return 0, err
	}
	return intEN, nil
}

func stringFromE8ForSymbolInfo(f int64) (string, error) {
	intDec := bdecimal.NewDecFromInt(f)
	resEN, err := bdecimal.NewDecFromFloat(0)
	if err != nil {
		return "", err
	}
	ENDec, err := bdecimal.NewDecFromFloat(math.Pow10(8))
	if err != nil {
		return "", err
	}
	err = bdecimal.DecimalDiv(intDec, ENDec, resEN, 5)
	if err != nil {
		return "", err
	}
	return resEN.String(), nil
}
