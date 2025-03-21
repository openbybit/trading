package gquote

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randString() string {
	b := strings.Builder{}
	for i := 0; i < 10; i++ {
		idx := rand.Intn(26)
		b.WriteRune(rune('a' + idx))
	}

	return b.String()
}

func toDecimal(x int64, scale int32) decimal.Decimal {
	return decimal.NewFromInt(x).Shift(-scale)
}

func parseSymbolDeliveryTime(s string) (time.Time, error) {
	tokens := strings.Split(s, "-")
	if len(tokens) < 2 {
		return time.Time{}, fmt.Errorf("invalid symbol format: %s, tokens: %v", s, tokens)
	}

	dt, err := parseTime(tokens[1])
	if err != nil {
		return time.Time{}, err
	}
	return dt.Add(time.Hour * 8), nil
}

func parseTime(timeStr string) (time.Time, error) {
	layout := "02Jan06"
	if len(timeStr) == 6 {
		layout = "2Jan06"
	}

	return time.Parse(layout, timeStr)
}
