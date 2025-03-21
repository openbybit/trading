package util

import (
	"crypto/md5"
	"strconv"
	"testing"

	"github.com/tj/assert"
)

func TestDecodeHeaderValue(t *testing.T) {
	a := []byte(`\ufffd\u0002\u0000\u0000GET`)
	v := DecodeHeaderValue(a)
	t.Log(v)
}

func TestToMD5(t *testing.T) {
	str := "asfgdshfhfgjhd"
	s := ToMD5Byte([]byte(str))
	t.Log(s, len(s))

	s = ToMD5(str)
	t.Log(s, len(s))

	s1 := md5.Sum([]byte(str))
	t.Logf("%x", s1)
}

func TestAtoi(t *testing.T) {
	x, _ := strconv.ParseInt("09", 10, 64)
	t.Log(x)
}

func TestValidateGrpcHeader(t *testing.T) {
	msg := "Mozilla/5.0 (iPhone; CPU OS 10_15_5 (Ergänzendes Update) like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/12.1.1 Mobile/14E304 Safari/605.1.15"
	t.Log(ToValidateGrpcHeader("UserAgent", msg))
}

func TestJsonGetAllVal(t *testing.T) {
	var v = `{
	"category" : "spot",
	"category" : "future"
}
`
	vs := JsonGetAllVal([]byte(v), "category")
	assert.Equal(t, len(vs), 2)
}

var req = `{
    "category":"spot",
    "symbol":"BTCUSDT",
    "orderType":"Limit", 
    "side":"sell",
    "qty":"0.01",
    "price":"21000",
    "timeInForce":"GTC",
    "smpType":"CancelTaker",
	"category":"spot",
}
`

// 300ns
func BenchmarkJsonGetAllVal(b *testing.B) {
	d := []byte(req)
	for i := 0; i < b.N; i++ {
		_ = JsonGetAllVal(d, "category")
	}
}
