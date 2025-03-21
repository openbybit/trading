package openapi

import (
	"bgw/pkg/common/constant"
	"bgw/pkg/test"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gcore/sign"
	"errors"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/tj/assert"
	"github.com/valyala/fasthttp"
	"testing"
)

func TestNewV3Checker(t *testing.T) {
	rctx, _ := test.NewReqCtx()
	rctx.Request.Header.SetMethod("xxxx")
	ret, err := newV3Checker(rctx, "zas", "129.0.0.1", true, false)
	assert.Equal(t, [2]Checker{nil, nil}, ret)
	assert.EqualError(t, err, "method error: Request parameter error.")

	rctx.Request.Header.SetMethod("POST")
	rctx.Request.Header.Set(constant.HeaderAPISign, "121212")
	ret, err = newV3Checker(rctx, "zas", "129.0.0.1", true, false)
	assert.Equal(t, "zas", ret[0].GetAPIKey())
	assert.Equal(t, "121212", ret[0].GetAPISign())
	assert.Equal(t, "129.0.0.1", ret[0].GetClientIP())
	assert.Equal(t, nil, ret[1])
	assert.NoError(t, err)

	// apiTimestamp == "" || apiKey == ""
	ret, err = newV3Checker(rctx, "", "129.0.0.1", false, false)
	assert.Equal(t, [2]Checker{Checker(nil), Checker(nil)}, ret)
	assert.EqualError(t, err, "empty value: apiTimestamp[] apiKey[]: Request parameter error.")

	//!isWss && apiSignature == ""
	ret, err = newV3Checker(rctx, "123", "129.0.0.1", false, true)
	assert.Equal(t, [2]Checker{Checker(nil), Checker(nil)}, ret)
	assert.EqualError(t, err, "empty value: apiTimestamp[] apiKey[123]: Request parameter error.")

	//get
	uri := &fasthttp.URI{}
	uri.SetQueryString("asas")
	rctx.Request.SetURI(uri)
	rctx.Request.Header.SetMethod("GET")
	rctx.Request.Header.Set(constant.HeaderAPITimeStamp, "2222")
	rctx.Request.Header.Set(constant.HeaderAPIRecvWindow, "333")
	ret, err = newV3Checker(rctx, "123", "129.0.0.1", false, false)

	assert.Equal(t, "v3", ret[0].GetVersion())
	assert.Equal(t, "123", ret[0].GetAPIKey())
	assert.Equal(t, "121212", ret[0].GetAPISign())
	assert.Equal(t, "129.0.0.1", ret[0].GetClientIP())
	assert.Equal(t, "333", ret[0].GetAPIRecvWindow())
	assert.Equal(t, "2222", ret[0].GetAPITimestamp())
	assert.Equal(t, "asas", string(ret[0].(*v3Checker).apiPayload))
	assert.Equal(t, nil, ret[1])
	assert.NoError(t, err)

	//post & maker apikey

	rctx.Request.SetURI(uri)
	rctx.Request.AppendBodyString("body")
	rctx.Request.Header.SetMethod("POST")
	rctx.Request.Header.Set(constant.HeaderAPITimeStamp, "2222")
	rctx.Request.Header.Set(constant.HeaderAPIRecvWindow, "333")
	rctx.Request.Header.Set(constant.HeaderAPIMAKERSIGN, "444")
	rctx.Request.Header.Set(constant.HeaderAPIMAKERAPIKEY, "555")
	ret, err = newV3Checker(rctx, "123", "129.0.0.1", false, false)
	assert.Equal(t, "123", ret[0].GetAPIKey())
	assert.Equal(t, "121212", ret[0].GetAPISign())
	assert.Equal(t, "129.0.0.1", ret[0].GetClientIP())
	assert.Equal(t, "333", ret[0].GetAPIRecvWindow())
	assert.Equal(t, "2222", ret[0].GetAPITimestamp())
	assert.Equal(t, "body", string(ret[0].(*v3Checker).apiPayload))

	assert.Equal(t, "555", ret[1].GetAPIKey())
	assert.Equal(t, "444", ret[1].GetAPISign())
	assert.Equal(t, "129.0.0.1", ret[1].GetClientIP())
	assert.Equal(t, "333", ret[1].GetAPIRecvWindow())
	assert.Equal(t, "2222", ret[1].GetAPITimestamp())
	assert.Equal(t, "body", string(ret[1].(*v3Checker).apiPayload))

	assert.NoError(t, err)

	// VerifySign err
	p := gomonkey.ApplyFunc(sign.Verify, func(typ sign.Type, secret, content []byte, s string) error {
		assert.Equal(t, sign.TypeHmac, typ)
		assert.Equal(t, "===", string(secret))
		assert.Equal(t, "2222123333body", string(content))
		assert.Equal(t, "121212", s)
		return errors.New("xxxx")
	})

	p.ApplyFunc(env.IsProduction, func() bool {
		return true
	})

	err = ret[0].VerifySign(rctx, sign.TypeHmac, "===")
	assert.EqualError(t, err, "error sign! origin_string[2222123333body]")
	p.ApplyFunc(env.IsProduction, func() bool {
		return false
	})
	err = ret[0].VerifySign(rctx, sign.TypeHmac, "===")
	assert.EqualError(t, err, "error sign! origin_string[2222123333body], sign=121212")

	p.ApplyFunc(sign.Verify, func(typ sign.Type, secret, content []byte, s string) error {
		return nil
	})
	err = ret[0].VerifySign(rctx, sign.TypeHmac, "===")
	assert.NoError(t, err)
	p.Reset()
}
