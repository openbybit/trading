package gformula

import (
	"math"
	"testing"
)

func TestCND(t *testing.T) {
	v1 := sndErfCND(0.234)
	v2 := sndCDN(0.234)
	v3 := sndPageCND(0.234)
	t.Log("cdn:", v1, v2, v3)
	// cdn: 0.5925075106684191 0.5925074671039878 0.592604447995196
	if math.Abs(v1-v2) > 1e-7 {
		t.Errorf("v1, v2 not equal")
	}
	if math.Abs(v1-v3) > 1e-3 {
		t.Errorf("v1, v3 not equal")
	}

	l := sndErfInverseCND(v1)
	t.Logf("inv: %v", l)
	// inv: 0.23400000000000004

	if math.Abs(l-0.234) > 0.0001 {
		t.Errorf("invalid sndErfInverseCND")
	}

	stdev := sndEvaluate([]float64{-0.0001581, -0.0007234, 0.0003890})
	t.Logf("stdev: %v", stdev)
	// stdev: 0.0005562248136620061
}
