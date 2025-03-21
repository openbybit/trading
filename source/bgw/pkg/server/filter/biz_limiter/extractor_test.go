package biz_limiter

import (
	"fmt"
	"strings"
	"testing"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/util"
	"bgw/pkg/server/metadata"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"github.com/tj/assert"
	"github.com/valyala/fasthttp"
)

func TestGetOptionKey(t *testing.T) {
	assert.Equal(t, "option:12:100", getOptionKey(100, "12"))
}

func TestValues(t *testing.T) {
	a := assert.New(t)
	app := "spot"
	// var uidRates [][2]int64
	uidRates := [][2]int64{
		{
			19156331,
			100,
		},
		{
			20775677,
			100,
		},
	}
	// nolint
	type quota struct {
		UID    string `json:"uid"`
		URI    string `json:"uri"`
		Method string `json:"method"`
		Any    string `json:"any"`
		Limit  string `json:"limit"`
	}

	//t.Log("user", len(uidRates))

	methPaths := [][2]string{
		{
			"GET",
			"/spot/v1/order",
		},
		{
			"POST",
			"/spot/v1/order",
		},
		{
			"DELETE",
			"/spot/v1/order",
		},
		{
			"DELETE",
			"/spot/v1/order/fast",
		},
		{
			"GET",
			"/spot/v1/history-orders",
		},
		{
			"POST",
			"/spot/v1/history-orders",
		},
		{
			"DELETE",
			"/spot/order/batch-cancel",
		},
		{
			"DELETE",
			"/spot/order/batch-fast-cancel",
		},
		{
			"DELETE",
			"/spot/order/batch-cancel-by-ids",
		},
		{
			"GET",
			"/spot/v1/open-orders",
		},
		{
			"POST",
			"/spot/v1/open-orders",
		},
		{
			"GET",
			"/spot/v1/myTrades",
		},
		{
			"POST",
			"/spot/v1/myTrades",
		},
		{
			"GET",
			"/spot/v1/account",
		},
		{
			"POST",
			"/spot/v1/account",
		},
		{
			"GET",
			"/spot/v3/private/account",
		},
		{
			"POST",
			"/spot/v3/private/order",
		},
		{
			"GET",
			"/spot/v3/private/order",
		},
		{
			"POST",
			"/spot/v3/private/cancel-order",
		},
		{
			"POST",
			"/spot/v3/private/cancel-orders",
		},
		{
			"POST",
			"/spot/v3/private/cancel-orders-by-ids",
		},
		{
			"GET",
			"/spot/v3/private/open-orders",
		},
		{
			"GET",
			"/spot/v3/private/history-orders",
		},
		{
			"GET",
			"/spot/v3/private/my-trades",
		},
	}

	r := extractor{
		UID:    true,
		Path:   true,
		Method: true,
	}
	c := &fasthttp.RequestCtx{}
	//t.Log("user", len(uidRates), "api", len(methPaths))
	count := 0
	result := ""
	for _, uids := range uidRates {
		uid := uids[0]
		rate := uids[1]
		for _, methPath := range methPaths {
			md := metadata.MDFromContext(c)
			md.UID = uid
			md.Method = methPath[0]
			md.Path = methPath[1]
			metadata.ContextWithMD(c, md)
			kv := r.Values(c, BTCUSD)
			vs := append([]string{app}, kv...)
			key := strings.Join(vs, ":")
			d := fmt.Sprintf("etcdctl put %s/%s/%s %d\n", constant.RootDataPath, constant.LimiterQuotaV2, key, rate)
			// fmt.Print(d)
			result += d
			count++

			c.ResetUserValues()
			a.NotNil(c)
		}
		vs := append([]string{app}, cast.Int64toa(uid))
		key := strings.Join(vs, ":")
		qv := &quotaValue{
			UID:   uid,
			Quota: int(rate),
		}
		quote, err := util.JsonMarshal(qv)
		a.NoError(err)
		d := fmt.Sprintf("etcdctl put %s/%s/%s '%s'\n", constant.RootDataPath, constant.LimiterQuotaV2, key, quote)
		result += d
	}
	//t.Log("------------")
	//fmt.Print(result)
	// err = util.WriteFile("/Users/SH88189ML/uid+path+method-result.yaml", []byte(result))
	// a.NoError(err)
	//t.Log("etcd key count", count)
}
