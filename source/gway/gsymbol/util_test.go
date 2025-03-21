package gsymbol

import "testing"

func TestCalcScale(t *testing.T) {
	x := []string{"0.01", "0.010000", "0.11", "0.", "1", "100"}
	for _, s := range x {
		t.Log(calcScale(s))
	}
}
