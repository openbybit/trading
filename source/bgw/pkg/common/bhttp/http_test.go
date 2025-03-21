package bhttp

import (
	"testing"

	"bgw/pkg/common/types"

	"github.com/tj/assert"
	"github.com/valyala/fasthttp"
)

func TestGetRemoteIP(t *testing.T) {
	a := assert.New(t)
	c := newReq()
	c.Request.Header.Set("X-Bybit-Forwarded-For", "23.45.67.89, 13.45.67.89")

	ip := GetRemoteIP(c)
	t.Log(ip)
	a.Equal("23.45.67.89", ip)

	c = newReq()
	c.Request.Header.Set("X-Forwarded-For", "23.45.67.89, 13.45.67.89")
	ip = GetRemoteIP(c)
	t.Log(ip)
	a.Equal("23.45.67.89", ip)

	c = newReq()
	c.Request.Header.Set("X-Bybit-Forwarded-For", "23.45.67.89, 13.45.67.89")
	c.Request.Header.Set("X-Forwarded-For", "43.45.67.89, 13.45.67.89")
	ip = GetRemoteIP(c)
	t.Log(ip)
	a.Equal("23.45.67.89", ip)
}

func newReq() *types.Ctx {
	return &types.Ctx{Request: fasthttp.Request{Header: fasthttp.RequestHeader{}}}
}
