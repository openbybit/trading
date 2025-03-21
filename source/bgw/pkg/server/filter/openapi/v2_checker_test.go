package openapi

import (
	"bgw/pkg/common/bhttp"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/test"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gcore/sign"
	"errors"
	"github.com/agiledragon/gomonkey/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/tj/assert"
	"github.com/valyala/fasthttp"
	"reflect"
	"testing"
)

func TestNewV2Checker(t *testing.T) {
	rctx, _ := test.NewReqCtx()
	rctx.Request.Header.SetMethod("GET")
	uri := &fasthttp.URI{}
	uri.SetQueryString("xx=123&bb=321&symbol=5&timestamp=2222&recvWindow=333&api_key=123&sign=121212")
	rctx.Request.SetURI(uri)

	// GET & allowGuest = true
	ret, err := newV2Checker(rctx, "129.0.0.1", true, false)
	param := rctx.UserValue(constant.BgwRequestParsed).(map[string]interface{})

	assert.Equal(t, "5", param["symbol"].(string))

	assert.Equal(t, "123", string(rctx.Request.Header.Peek(constant.HeaderAPIKey)))
	assert.Equal(t, "123", ret[0].GetAPIKey())
	assert.Equal(t, "121212", ret[0].GetAPISign())
	assert.Equal(t, "129.0.0.1", ret[0].GetClientIP())
	assert.Equal(t, "", ret[0].GetAPIRecvWindow())
	assert.Equal(t, "", ret[0].GetAPITimestamp())
	assert.Equal(t, nil, ret[1])
	assert.NoError(t, err)

	//apiTimestamp == "" || apiKey == "" || apiSignature == ""
	rctx.Request.Header.Set(constant.HeaderAPIKey, "")
	uri.SetQueryString("xx=123&bb=321&symbol=5&timestamp=&recvWindow=333&api_key=&sign=")
	rctx.Request.SetURI(uri)
	ret, err = newV2Checker(rctx, "129.0.0.1", false, false)
	param = rctx.UserValue(constant.BgwRequestParsed).(map[string]interface{})

	assert.Equal(t, "5", param["symbol"].(string))

	assert.Equal(t, nil, ret[0])
	assert.Equal(t, nil, ret[1])
	assert.EqualError(t, err, "empty value: apiTimestamp[] apiKey[] apiSignature[]")

	//makerSign != "" && makerApiKey != ""
	rctx.Request.Header.Set(constant.HeaderAPIKey, "")
	uri.SetQueryString("symbol=5&timestamp=2222&recvWindow=333&api_key=123&sign=121212&maker_api_key=xx&maker_sign=de")
	rctx.Request.SetURI(uri)
	ret, err = newV2Checker(rctx, "129.0.0.1", false, false)
	assert.Equal(t, "123", string(rctx.Request.Header.Peek(constant.HeaderAPIKey)))
	assert.Equal(t, "123", ret[0].GetAPIKey())
	assert.Equal(t, "121212", ret[0].GetAPISign())
	assert.Equal(t, "129.0.0.1", ret[0].GetClientIP())
	assert.Equal(t, "333", ret[0].GetAPIRecvWindow())
	assert.Equal(t, "2222", ret[0].GetAPITimestamp())

	assert.Equal(t, "5", ret[0].(*v2Checker).params["symbol"])
	assert.Equal(t, "2222", ret[0].(*v2Checker).params["timestamp"])
	assert.Equal(t, "333", ret[0].(*v2Checker).params["recvWindow"])
	assert.Equal(t, "123", ret[0].(*v2Checker).params["api_key"])
	assert.Equal(t, "121212", ret[0].(*v2Checker).params["sign"])
	assert.Equal(t, "xx", ret[0].(*v2Checker).params["maker_api_key"])
	assert.Equal(t, "de", ret[0].(*v2Checker).params["maker_sign"])
	assert.Equal(t, false, ret[0].(*v2Checker).postQueryFlag)
	assert.Equal(t, 0, len(ret[0].(*v2Checker).postQueryData))

	assert.Equal(t, "xx", ret[1].GetAPIKey())
	assert.Equal(t, "de", ret[1].GetAPISign())
	assert.Equal(t, "129.0.0.1", ret[1].GetClientIP())
	assert.Equal(t, "333", ret[1].GetAPIRecvWindow())
	assert.Equal(t, "2222", ret[1].GetAPITimestamp())

	assert.Equal(t, "5", ret[1].(*v2Checker).params["symbol"])
	assert.Equal(t, "2222", ret[1].(*v2Checker).params["timestamp"])
	assert.Equal(t, "333", ret[1].(*v2Checker).params["recvWindow"])
	assert.Equal(t, "123", ret[1].(*v2Checker).params["api_key"])
	assert.Equal(t, "121212", ret[1].(*v2Checker).params["sign"])
	assert.Equal(t, "xx", ret[1].(*v2Checker).params["maker_api_key"])
	assert.Equal(t, "de", ret[1].(*v2Checker).params["maker_sign"])
	assert.Equal(t, false, ret[1].(*v2Checker).postQueryFlag)
	assert.Equal(t, 0, len(ret[1].(*v2Checker).postQueryData))
	assert.NoError(t, err)
}

func TestNewV2CheckerParseParam(t *testing.T) {
	rctx, _ := test.NewReqCtx()
	rctx.Request.Header.SetMethod("POST")
	uri := &fasthttp.URI{}
	// body len is 0
	rctx.Request.Header.Set(constant.HeaderAPIKey, "")
	uri.SetQueryString("symbol=5&timestamp=2222&recvWindow=333&api_key=123&sign=121212&maker_api_key=xx&maker_sign=de")
	rctx.Request.SetURI(uri)
	p := gomonkey.ApplyFunc(jsoniter.Marshal, func(v interface{}) ([]byte, error) {
		return nil, errors.New("xxxx")
	})
	ret, err := newV2Checker(rctx, "129.0.0.1", false, false)
	assert.EqualError(t, err, "invalid params")
	p.Reset()
	p.ApplyFunc(jsoniter.Marshal, func(v interface{}) ([]byte, error) {
		return []byte("{\"symbol\":\"5\",\"timestamp\":\"2222\",\"maker_api_key\":\"xx\",\"recvWindow\":\"333\",\"api_key\":\"123\",\"sign\":\"121212\",\"maker_sign\":\"de\"}"), nil
	})
	ret, err = newV2Checker(rctx, "129.0.0.1", false, false)
	p.Reset()
	assert.Equal(t, "{\"symbol\":\"5\",\"timestamp\":\"2222\",\"maker_api_key\":\"xx\",\"recvWindow\":\"333\",\"api_key\":\"123\",\"sign\":\"121212\",\"maker_sign\":\"de\"}", string(rctx.UserValue("request-handled-body").([]byte)))
	assert.Equal(t, true, rctx.UserValue(constant.BgwRequestHandled).(bool))
	assert.Equal(t, "123", string(rctx.Request.Header.Peek(constant.HeaderAPIKey)))
	assert.Equal(t, "123", ret[0].GetAPIKey())
	assert.Equal(t, "121212", ret[0].GetAPISign())
	assert.Equal(t, "129.0.0.1", ret[0].GetClientIP())
	assert.Equal(t, "333", ret[0].GetAPIRecvWindow())
	assert.Equal(t, "2222", ret[0].GetAPITimestamp())

	assert.Equal(t, "5", ret[0].(*v2Checker).params["symbol"])
	assert.Equal(t, "2222", ret[0].(*v2Checker).params["timestamp"])
	assert.Equal(t, "333", ret[0].(*v2Checker).params["recvWindow"])
	assert.Equal(t, "123", ret[0].(*v2Checker).params["api_key"])
	assert.Equal(t, "121212", ret[0].(*v2Checker).params["sign"])
	assert.Equal(t, "xx", ret[0].(*v2Checker).params["maker_api_key"])
	assert.Equal(t, "de", ret[0].(*v2Checker).params["maker_sign"])
	assert.Equal(t, true, ret[0].(*v2Checker).postQueryFlag)
	assert.Equal(t, 7, len(ret[0].(*v2Checker).postQueryData))
	assert.Equal(t, "5", ret[0].(*v2Checker).postQueryData["symbol"])
	assert.Equal(t, "2222", ret[0].(*v2Checker).postQueryData["timestamp"])
	assert.Equal(t, "333", ret[0].(*v2Checker).postQueryData["recvWindow"])
	assert.Equal(t, "123", ret[0].(*v2Checker).postQueryData["api_key"])
	assert.Equal(t, "121212", ret[0].(*v2Checker).postQueryData["sign"])
	assert.Equal(t, "xx", ret[0].(*v2Checker).postQueryData["maker_api_key"])
	assert.Equal(t, "de", ret[0].(*v2Checker).postQueryData["maker_sign"])
	assert.NoError(t, err)
}

func TestNewV2CheckerParseParam2(t *testing.T) {
	//bytes.HasPrefix(ctx.Request.Header.ContentType(), bhttp.ContentTypePostForm):
	rctx, _ := test.NewReqCtx()
	rctx.Request.Header.SetMethod("POST")
	uri := &fasthttp.URI{}

	rctx.Request.AppendBodyString("ss")
	rctx.Request.Header.SetContentType(string(bhttp.ContentTypePostForm))
	rctx.Request.Header.Set(constant.HeaderAPIKey, "")
	uri.SetQueryString("symbol=5&timestamp=2222&recvWindow=333&api_key=123&sign=121212&maker_api_key=xx&maker_sign=de")
	rctx.Request.SetURI(uri)
	p := gomonkey.ApplyFunc(jsoniter.Marshal, func(v interface{}) ([]byte, error) {
		return nil, errors.New("xxxx")
	})
	ret, err := newV2Checker(rctx, "129.0.0.1", false, true)
	assert.EqualError(t, err, "invalid params")
	p.Reset()
	p.ApplyFunc(jsoniter.Marshal, func(v interface{}) ([]byte, error) {
		return []byte("{\"symbol\":\"5\",\"timestamp\":\"2222\",\"maker_api_key\":\"xx\",\"recvWindow\":\"333\",\"api_key\":\"123\",\"sign\":\"121212\",\"maker_sign\":\"de\"}"), nil
	})
	ret, err = newV2Checker(rctx, "129.0.0.1", false, true)
	p.Reset()
	assert.Equal(t, "{\"symbol\":\"5\",\"timestamp\":\"2222\",\"maker_api_key\":\"xx\",\"recvWindow\":\"333\",\"api_key\":\"123\",\"sign\":\"121212\",\"maker_sign\":\"de\"}", string(rctx.UserValue("request-handled-body").([]byte)))
	assert.Equal(t, true, rctx.UserValue(constant.BgwRequestHandled).(bool))
	assert.Equal(t, "123", string(rctx.Request.Header.Peek(constant.HeaderAPIKey)))
	assert.Equal(t, "123", ret[0].GetAPIKey())
	assert.Equal(t, "121212", ret[0].GetAPISign())
	assert.Equal(t, "129.0.0.1", ret[0].GetClientIP())
	assert.Equal(t, "333", ret[0].GetAPIRecvWindow())
	assert.Equal(t, "2222", ret[0].GetAPITimestamp())

	assert.Equal(t, "5", ret[0].(*v2Checker).params["symbol"])
	assert.Equal(t, "2222", ret[0].(*v2Checker).params["timestamp"])
	assert.Equal(t, "333", ret[0].(*v2Checker).params["recvWindow"])
	assert.Equal(t, "123", ret[0].(*v2Checker).params["api_key"])
	assert.Equal(t, "121212", ret[0].(*v2Checker).params["sign"])
	assert.Equal(t, "xx", ret[0].(*v2Checker).params["maker_api_key"])
	assert.Equal(t, "de", ret[0].(*v2Checker).params["maker_sign"])
	assert.Equal(t, false, ret[0].(*v2Checker).postQueryFlag)
	assert.Equal(t, 0, len(ret[0].(*v2Checker).postQueryData))
	assert.NoError(t, err)
}

func TestNewV2CheckerParseParam3(t *testing.T) {
	//queryFallback && len(ctx.URI().QueryString()) > 0
	rctx, _ := test.NewReqCtx()
	rctx.Request.Header.SetMethod("POST")
	uri := &fasthttp.URI{}
	rctx.Request.AppendBodyString("x")
	rctx.Request.Header.Set(constant.HeaderAPIKey, "")
	uri.SetQueryString("symbol=5&timestamp=2222&recvWindow=333&api_key=123&sign=121212&maker_api_key=xx&maker_sign=de")
	rctx.Request.SetURI(uri)
	p := gomonkey.ApplyFunc(jsoniter.Marshal, func(v interface{}) ([]byte, error) {
		return []byte("{\"symbol\":\"5\",\"timestamp\":\"2222\",\"maker_api_key\":\"xx\",\"recvWindow\":\"333\",\"api_key\":\"123\",\"sign\":\"121212\",\"maker_sign\":\"de\"}"), nil
	})
	ret, err := newV2Checker(rctx, "129.0.0.1", false, true)
	p.Reset()
	assert.Equal(t, "123", string(rctx.Request.Header.Peek(constant.HeaderAPIKey)))
	assert.Equal(t, "123", ret[0].GetAPIKey())
	assert.Equal(t, "121212", ret[0].GetAPISign())
	assert.Equal(t, "129.0.0.1", ret[0].GetClientIP())
	assert.Equal(t, "333", ret[0].GetAPIRecvWindow())
	assert.Equal(t, "2222", ret[0].GetAPITimestamp())

	assert.Equal(t, "5", ret[0].(*v2Checker).params["symbol"])
	assert.Equal(t, "2222", ret[0].(*v2Checker).params["timestamp"])
	assert.Equal(t, "333", ret[0].(*v2Checker).params["recvWindow"])
	assert.Equal(t, "123", ret[0].(*v2Checker).params["api_key"])
	assert.Equal(t, "121212", ret[0].(*v2Checker).params["sign"])
	assert.Equal(t, "xx", ret[0].(*v2Checker).params["maker_api_key"])
	assert.Equal(t, "de", ret[0].(*v2Checker).params["maker_sign"])
	assert.Equal(t, false, ret[0].(*v2Checker).postQueryFlag)
	assert.Equal(t, 0, len(ret[0].(*v2Checker).postQueryData))
	assert.NoError(t, err)
}

func TestNewV2CheckerParseParam4(t *testing.T) {
	//default
	rctx, _ := test.NewReqCtx()
	rctx.Request.Header.SetMethod("POST")
	rctx.Request.Header.Set(constant.HeaderAPIKey, "")
	p := gomonkey.ApplyFunc(jsoniter.Marshal, func(v interface{}) ([]byte, error) {
		return []byte("{\"symbol\":\"5\",\"timestamp\":\"2222\",\"maker_api_key\":\"xx\",\"recvWindow\":\"333\",\"api_key\":\"123\",\"sign\":\"121212\",\"maker_sign\":\"de\"}"), nil
	})
	rctx.Request.AppendBodyString("xxx")
	ret, err := newV2Checker(rctx, "129.0.0.1", false, true)

	assert.EqualError(t, err, "body not json")
	rctx.Request.Reset()
	rctx.Request.Header.SetMethod("POST")
	rctx.Request.Header.Set(constant.HeaderAPIKey, "")
	rctx.Request.AppendBodyString("{\"symbol\":\"5\",\"timestamp\":\"2222\",\"maker_api_key\":\"xx\",\"recvWindow\":\"333\",\"api_key\":\"123\",\"sign\":\"121212\",\"maker_sign\":\"de\"}")

	ret, err = newV2Checker(rctx, "129.0.0.1", false, true)
	p.Reset()
	assert.Equal(t, "123", string(rctx.Request.Header.Peek(constant.HeaderAPIKey)))
	assert.Equal(t, "123", ret[0].GetAPIKey())
	assert.Equal(t, "121212", ret[0].GetAPISign())
	assert.Equal(t, "129.0.0.1", ret[0].GetClientIP())
	assert.Equal(t, "333", ret[0].GetAPIRecvWindow())
	assert.Equal(t, "2222", ret[0].GetAPITimestamp())

	assert.Equal(t, "5", ret[0].(*v2Checker).params["symbol"])
	assert.Equal(t, "2222", ret[0].(*v2Checker).params["timestamp"])
	assert.Equal(t, "333", ret[0].(*v2Checker).params["recvWindow"])
	assert.Equal(t, "123", ret[0].(*v2Checker).params["api_key"])
	assert.Equal(t, "121212", ret[0].(*v2Checker).params["sign"])
	assert.Equal(t, "xx", ret[0].(*v2Checker).params["maker_api_key"])
	assert.Equal(t, "de", ret[0].(*v2Checker).params["maker_sign"])
	assert.Equal(t, false, ret[0].(*v2Checker).postQueryFlag)
	assert.Equal(t, 0, len(ret[0].(*v2Checker).postQueryData))
	assert.NoError(t, err)
}

func TestV2Checker_VerifySign(t *testing.T) {
	rctx, _ := test.NewReqCtx()
	rctx.Request.Header.SetMethod("POST")
	rctx.Request.Header.Set(constant.HeaderAPIKey, "")
	rctx.Request.AppendBodyString("{\"symbol\":\"5\",\"timestamp\":\"2222\",\"maker_api_key\":\"xx\",\"recvWindow\":\"333\",\"api_key\":\"123\",\"sign\":\"121212\",\"maker_sign\":\"de\"}")

	ret, _ := newV2Checker(rctx, "129.0.0.1", false, true)
	assert.Equal(t, "v2", ret[0].GetVersion())
	v2 := ret[0].(*v2Checker)
	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(ret[0]), "checkSign", func(ctx *types.Ctx, signTyp sign.Type, secret string, params map[string]string) error {
		return nil
	})
	err := v2.VerifySign(rctx, sign.TypeHmac, "xxx")
	assert.NoError(t, err)
	p.Reset()
	//!ctx.IsGet() && !v2.postQueryFlag
	rctx.Request.Header.SetMethod("POST")
	p.ApplyPrivateMethod(reflect.TypeOf(ret[0]), "checkSign", func(ctx *types.Ctx, signTyp sign.Type, secret string, params map[string]string) error {
		return errors.New("xxx")
	})
	err = v2.VerifySign(rctx, sign.TypeHmac, "xxx")
	assert.EqualError(t, err, "xxx")
	p.Reset()
	//v2.postQueryFlag = true
	v2.postQueryFlag = true
	p.ApplyPrivateMethod(reflect.TypeOf(ret[0]), "checkSign", func(ctx *types.Ctx, signTyp sign.Type, secret string, params map[string]string) error {
		return errors.New("xxx")
	})
	err = v2.VerifySign(rctx, sign.TypeHmac, "xxx")
	assert.EqualError(t, err, "xxx")
	p.ApplyPrivateMethod(reflect.TypeOf(ret[0]), "checkSign", func(ctx *types.Ctx, signTyp sign.Type, secret string, params map[string]string) error {
		return nil
	})
	err = v2.VerifySign(rctx, sign.TypeHmac, "xxx")
	assert.NoError(t, err)
	p.Reset()
	//v2.postQueryFlag = false & method = get
	rctx.Request.Header.SetMethod("GET")
	v2.postQueryFlag = false
	uri := &fasthttp.URI{}
	uri.SetQueryString("symbol=5&timestamp=2222&recvWindow=333&api_key=123&sign=121212&maker_api_key=xx&maker_sign=de")
	rctx.Request.SetURI(uri)
	p.ApplyPrivateMethod(reflect.TypeOf(ret[0]), "checkSign", func(ctx *types.Ctx, signTyp sign.Type, secret string, params map[string]string) error {
		return errors.New("xxx")
	})
	err = v2.VerifySign(rctx, sign.TypeHmac, "xxx")
	assert.EqualError(t, err, "xxx")
	p.Reset()
	p.ApplyPrivateMethod(reflect.TypeOf(ret[0]), "checkSign", func(ctx *types.Ctx, signTyp sign.Type, secret string, params map[string]string) error {
		return nil
	})
	err = v2.VerifySign(rctx, sign.TypeHmac, "xxx")
	assert.NoError(t, err)
	p.Reset()

}

func TestCheckSign(t *testing.T) {
	rctx, _ := test.NewReqCtx()
	rctx.Request.Header.SetMethod("POST")
	rctx.Request.Header.Set(constant.HeaderAPIKey, "")
	rctx.Request.AppendBodyString("{\"symbol\":\"5\",\"timestamp\":\"2222\",\"maker_api_key\":\"xx\",\"recvWindow\":\"333\",\"api_key\":\"123\",\"sign\":\"121212\",\"maker_sign\":\"de\"}")

	ret, _ := newV2Checker(rctx, "129.0.0.1", false, true)

	v2 := ret[0].(*v2Checker)

	p := gomonkey.ApplyFunc(sign.Verify, func(typ sign.Type, secret, content []byte, s string) error {
		assert.Equal(t, sign.TypeHmac, typ)
		assert.Equal(t, "xxx", string(secret))
		assert.Equal(t, "api_key=123&maker_api_key=xx&maker_sign=de&recvWindow=333&symbol=5&timestamp=2222", string(content))
		assert.Equal(t, "121212", s)
		return errors.New("xxxx")
	})
	p.ApplyFunc(env.IsProduction, func() bool {
		return false
	})
	err := v2.checkSign(rctx, sign.TypeHmac, "xxx", v2.params)
	assert.EqualError(t, err, "error sign! origin_string[api_key=123&maker_api_key=xx&maker_sign=de&recvWindow=333&symbol=5&timestamp=2222], sign=121212")

	p.ApplyFunc(env.IsProduction, func() bool {
		return true
	})
	v2.params["sign"] = "121212"
	err = v2.checkSign(rctx, sign.TypeHmac, "xxx", v2.params)
	assert.EqualError(t, err, "error sign! origin_string[api_key=123&maker_api_key=xx&maker_sign=de&recvWindow=333&symbol=5&timestamp=2222]")

	p.Reset()
}
