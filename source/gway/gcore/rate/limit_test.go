package rate

import (
	"testing"
	"time"
)

func TestRateLimit(t *testing.T) {
	r := New(0, 0)
	t.Log(r.Allow(), "should true")

	r = New(1, 0)
	t.Log(r.Allow(), "should true")
	t.Log(r.Allow(), "should false")

	time.Sleep(time.Second)
	t.Log(r.Allow(), "should true")
	t.Log(r.Allow(), "should false")
}
