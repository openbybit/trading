package gquote

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestParseSymbolDeliveryTime(t *testing.T) {
	// 30JUN23 2JUN23
	// 02JUN23
	list := []string{"BTC-30JUN23-8000-C", "SOL-2JUN23-22-P"}
	for _, s := range list {
		dt, err := parseSymbolDeliveryTime(s)
		if err != nil {
			t.Errorf("parse fail err: %v", err)
		} else {
			t.Log(dt, time.Now().UTC(), err)
		}
	}

	month := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	for _, m := range month {
		m = strings.ToUpper(m)
		s := fmt.Sprintf("2%s23", m)
		x, err := parseTime(s)
		if err != nil {
			t.Error(err)
		} else {
			t.Log(s, x)
		}
	}
}

func TestDecimal(t *testing.T) {
	d := toDecimal(2804005000000, 8)
	t.Log(d)
	// 28040.05
}
