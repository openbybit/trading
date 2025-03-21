package response

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"git.bybit.com/svc/mod/pkg/bplatform"
	"github.com/tj/assert"
	"github.com/valyala/fasthttp"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/server/core/http"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/filter/response/version"
	gmetadata "bgw/pkg/server/metadata"
)

func TestResponse(t *testing.T) {
	t.Run("response init 0 args", func(t *testing.T) {
		Init()
		resp := &response{}
		assert.Equal(t, filter.ResponseFilterKey, resp.GetName())

		err := resp.Init(nil)
		assert.Nil(t, err)
	})
	t.Run("response init 1", func(t *testing.T) {
		resp := &response{}
		assert.Equal(t, filter.ResponseFilterKey, resp.GetName())

		err := resp.Init(nil, "response", "-any true")
		assert.NotNil(t, err)

		err = resp.Init(nil, "response.1.2.3.4.5.true")
		assert.Nil(t, err)
		if reflect.ValueOf(resp.handler).Pointer() != reflect.ValueOf(handleDefault).Pointer() {
			t.Fail()
		}

		err = resp.Init(nil, "response", "--any=true")
		assert.Nil(t, err)
		if reflect.ValueOf(resp.handler).Pointer() != reflect.ValueOf(handleAny).Pointer() {
			t.Fail()
		}
		err = resp.Init(nil, "response", "--any=false", "--metaInBody=true")
		assert.Nil(t, err)
		if reflect.ValueOf(resp.handler).Pointer() != reflect.ValueOf(handleConvert).Pointer() {
			t.Fail()
		}

	})
	t.Run("response skip option", func(t *testing.T) {
		resp := new()
		assert.Equal(t, filter.ResponseFilterKey, resp.GetName())

		rctx := makeReqCtx()
		rctx.Request.Header.SetMethod(fasthttp.MethodOptions)
		err := resp.Do(func(rctx *fasthttp.RequestCtx) error {
			return errors.New("xxx")
		})(rctx)
		assert.NotNil(t, err)
		err = resp.Do(func(rctx *fasthttp.RequestCtx) error {
			return nil
		})(rctx)
		assert.Nil(t, err)
	})
	t.Run("response do and recovery", func(t *testing.T) {
		resp := &response{}
		assert.Equal(t, filter.ResponseFilterKey, resp.GetName())
		rctx := makeReqCtx()
		rctx.Request.Header.SetMethod(fasthttp.MethodOptions)
		resp.recover(rctx, "12")
		assert.Equal(t, berror.HttpServerError, rctx.Response.StatusCode())

		rctx.Request.Header.SetMethod(fasthttp.MethodOptions)
		resp.recover(rctx, errors.New("12333"))
		assert.Equal(t, berror.HttpServerError, rctx.Response.StatusCode())

		err := resp.Do(nil)(rctx)
		assert.Nil(t, err)
		rctx.Request.Header.SetMethod(fasthttp.MethodGet)

		err = resp.Do(func(rctx *fasthttp.RequestCtx) error {
			return nil
		})(rctx)
		assert.Equal(t, errInvalidResult, err)

		err = resp.Do(func(rctx *fasthttp.RequestCtx) error {
			return errors.New("xxx")
		})(rctx)
		assert.NotNil(t, err)

		err = resp.Do(func(rctx *fasthttp.RequestCtx) error {
			return errors.New("errInvalidAnyResult")
		})(rctx)
		assert.NotNil(t, err)

		c := http.NewResult()
		rctx.SetUserValue(constant.CtxInvokeResult, c)
		resp.handler = func(ctx *types.Ctx, source resultCarrier, target Target) (err error) {
			return nil
		}
		err = resp.Do(func(rctx *fasthttp.RequestCtx) error {
			return nil
		})(rctx)
		assert.Nil(t, err)
		resp.handler = func(ctx *types.Ctx, source resultCarrier, target Target) (err error) {
			return errors.New("3323")
		}
		err = resp.Do(func(rctx *fasthttp.RequestCtx) error {
			return nil
		})(rctx)
		assert.Equal(t, "3323", err.Error())

	})
	t.Run("response finally", func(t *testing.T) {
		resp := &response{}
		resp.flags.translator = &translate{}
		assert.Equal(t, filter.ResponseFilterKey, resp.GetName())
		rctx := makeReqCtx()
		rctx.Request.Header.SetMethod(fasthttp.MethodOptions)

		tar := version.NewV2Response()
		tar.SetMessage("xzxzx")
		tar.SetCode(0)
		tar.SetResult([]byte("222"))
		r := http.NewResult()
		r.SetStatus(400)

		resp.finally(rctx, r, tar)
		assert.Equal(t, 400, rctx.Response.StatusCode())

		sss := rctx.Response.Body()
		rrrr := version.NewV2Response()
		err := json.Unmarshal(sss, rrrr)
		assert.Nil(t, err)
		assert.Equal(t, []byte("222"), rrrr.GetResult())
		assert.Equal(t, "xzxzx", rrrr.GetMessage())
		assert.Equal(t, "{}", string(rrrr.GetExtInfo()))

		v1 := version.NewV1Response()
		source := http.NewResult()
		source.SetMetadata(map[string][]string{constant.BgwAPIResponseExtCode: {"ext"}})
		resp.finally(rctx, source, v1)
		assert.Equal(t, v1.ExtCode, "ext")

	})
	t.Run("response get target", func(t *testing.T) {
		resp := &response{}
		assert.Equal(t, filter.ResponseFilterKey, resp.GetName())
		rctx := makeReqCtx()
		resp.flags.version = "1xxx2"
		tar := resp.getTarget(rctx)
		assert.IsType(t, version.NewV2Response(), tar)
		resp.flags.version = version.VersionV1
		tar = resp.getTarget(rctx)
		assert.IsType(t, version.NewV1Response(), tar)
		resp.flags.version = version.VersionV2
		tar = resp.getTarget(rctx)
		assert.IsType(t, version.NewV2Response(), tar)

		resp.flags.version = version.VersionV1
		tar = resp.getTarget(rctx)
		assert.IsType(t, version.NewV1Response(), tar)

		resp.flags.passthrough = true
		tar = resp.getTarget(rctx)
		assert.IsType(t, version.NewPassthroughResponse(), tar)

		rctx = makeReqCtx()
		uri := &fasthttp.URI{}
		uri.SetQueryString("_sp_response_format=hump")
		rctx.Request.SetURI(uri)
		resp.flags.passthrough = false
		tar = resp.getTarget(rctx)
		assert.IsType(t, version.NewV2Response(), tar)

		uri = &fasthttp.URI{}
		uri.SetQueryString("_sp_response_format=portugal")
		rctx.Request.SetURI(uri)
		resp.flags.passthrough = false
		tar = resp.getTarget(rctx)
		assert.IsType(t, version.NewV1Response(), tar)

	})
	t.Run("response getErrorTarget", func(t *testing.T) {
		resp := &response{}
		assert.Equal(t, filter.ResponseFilterKey, resp.GetName())

		tar := resp.getErrorTarget(version.VersionV1)
		assert.IsType(t, version.NewV1Response(), tar)
		tar = resp.getErrorTarget(version.VersionPassthrough)
		assert.IsType(t, version.NewV2Response(), tar)
		tar = resp.getErrorTarget("asas")
		assert.IsType(t, version.NewV2Response(), tar)

		resp.flags.any = true
		tar = resp.getErrorTarget(version.VersionPassthrough)
		assert.IsType(t, version.NewV2Response(), tar)

		resp.flags.any = false
		resp.flags.version = version.VersionV1
		tar = resp.getErrorTarget(version.VersionPassthrough)
		assert.IsType(t, version.NewV1Response(), tar)

		resp.flags.version = "12121212"
		tar = resp.getErrorTarget(version.VersionPassthrough)
		assert.IsType(t, version.NewV2Response(), tar)
	})
	t.Run("response handleError", func(t *testing.T) {
		resp := &response{}
		assert.Equal(t, filter.ResponseFilterKey, resp.GetName())

		rctx := makeReqCtx()
		rctx.Request.Header.SetMethod(fasthttp.MethodOptions)

		tar := resp.handleError(rctx, version.VersionV2, berror.NewBizErr(100, "12121"))

		assert.Equal(t, "12121", tar.GetMessage())
		assert.Equal(t, int64(100), tar.GetCode())

		tar = resp.handleError(rctx, version.VersionV2, errors.New("xxxxx"))

		assert.Equal(t, "Internal System Error.", tar.GetMessage())
		assert.Equal(t, berror.SystemInternalError, tar.GetCode())

		md := gmetadata.NewMetadata()
		rctx.SetUserValue(constant.METADATA_CTX, md)
		md.Route.ACL.Group = constant.ResourceGroupBlockTrade

		tar = resp.handleError(rctx, version.VersionV2, errors.New("xxxxx"))

		assert.Equal(t, "Internal System Error.", tar.GetMessage())
		assert.Equal(t, berror.SystemInternalError, tar.GetCode())
		assert.Equal(t, "{\"blockTradeId\":\"\",\"status\":\"Rejected\",\"rejectParty\":\"Taker\"}", string(tar.GetExtInfo()))

		env.SetEnvName("mainnet")

		tar = resp.handleError(rctx, version.VersionV2, errors.New("xxxxx"))

		assert.Equal(t, "Internal System Error.", tar.GetMessage())
		assert.Equal(t, berror.SystemInternalError, tar.GetCode())

		env.SetEnvName("qa")

		tar = resp.handleError(rctx, version.VersionV2, errors.New("xxxxx"))

		assert.Equal(t, "Internal System Error.", tar.GetMessage())
		assert.Equal(t, berror.SystemInternalError, tar.GetCode())
	})

	t.Run("response extend V2", func(t *testing.T) {
		resp := &response{}
		assert.Equal(t, filter.ResponseFilterKey, resp.GetName())
		resp.flags.rateLimit = true
		rctx := makeReqCtx()
		md := gmetadata.NewMetadata()
		rctx.SetUserValue(constant.METADATA_CTX, md)

		tar := version.NewV2Response()
		tar.SetMessage("xzxzx")
		tar.SetCode(20)
		tar.SetResult([]byte("222"))
		resp.extend(rctx, tar)

		assert.Equal(t, "20", string(rctx.Response.Header.Peek(retCode)))
		assert.Equal(t, "no-store", string(rctx.Response.Header.Peek("Cache-Control")))
		assert.Equal(t, "222", string(tar.Result))

		rctx = makeReqCtx()
		rctx.SetUserValue(constant.METADATA_CTX, md)
		md.Extension.Platform = string(bplatform.OpenAPI)
		resp.extend(rctx, tar)

		assert.Equal(t, "20", string(rctx.Response.Header.Peek(retCode)))
		assert.Equal(t, "no-store", string(rctx.Response.Header.Peek("Cache-Control")))
		assert.Equal(t, "{}", string(tar.Result))

		rctx = makeReqCtx()
		rctx.SetUserValue(constant.METADATA_CTX, md)
		md.Extension.Platform = string(bplatform.AndroidAPP)
		resp.extend(rctx, tar)

		assert.Equal(t, "20", string(rctx.Response.Header.Peek(retCode)))
		assert.Equal(t, "no-store", string(rctx.Response.Header.Peek("Cache-Control")))
		assert.Equal(t, "{}", string(tar.Result))

		rctx = makeReqCtx()
		rctx.SetUserValue(constant.METADATA_CTX, md)
		rctx.Response.AppendBodyString("kkkk")
		md.Extension.Platform = string(bplatform.OpenAPI)
		resp.extend(rctx, tar)

		assert.Equal(t, "20", string(rctx.Response.Header.Peek(retCode)))
		assert.Equal(t, "no-store", string(rctx.Response.Header.Peek("Cache-Control")))
		assert.Equal(t, "kkkk", string(tar.Result))

		rctx = makeReqCtx()
		rctx.SetUserValue(constant.METADATA_CTX, md)
		rctx.Response.AppendBodyString("kkkk")
		md.Extension.Platform = string(bplatform.AndroidAPP)
		tar.SetExtInfo([]byte{})
		resp.extend(rctx, tar)

		assert.Equal(t, "20", string(rctx.Response.Header.Peek(retCode)))
		assert.Equal(t, "no-store", string(rctx.Response.Header.Peek("Cache-Control")))
		assert.Equal(t, "kkkk", string(tar.Result))
		assert.Equal(t, "{}", string(tar.GetExtInfo()))
	})

	t.Run("response extend V1", func(t *testing.T) {
		resp := &response{}
		assert.Equal(t, filter.ResponseFilterKey, resp.GetName())
		resp.flags.rateLimit = true
		rctx := makeReqCtx()
		md := gmetadata.NewMetadata()
		rctx.SetUserValue(constant.METADATA_CTX, md)

		tar := version.NewV1Response()
		tar.SetMessage("xzxzx")
		tar.SetCode(20)
		tar.SetResult([]byte("222"))
		resp.extend(rctx, tar)

		assert.Equal(t, "20", string(rctx.Response.Header.Peek(retCode)))
		assert.Equal(t, "no-store", string(rctx.Response.Header.Peek("Cache-Control")))
		assert.Equal(t, "222", string(tar.Result))

		rctx = makeReqCtx()
		rctx.SetUserValue(constant.METADATA_CTX, md)

		md.Extension.Platform = string(bplatform.OpenAPI)
		resp.extend(rctx, tar)

		assert.Equal(t, "20", string(rctx.Response.Header.Peek(retCode)))
		assert.Equal(t, "no-store", string(rctx.Response.Header.Peek("Cache-Control")))
		assert.Equal(t, "{}", string(tar.Result))

		rctx = makeReqCtx()
		rctx.SetUserValue(constant.METADATA_CTX, md)
		md.Extension.Platform = string(bplatform.AndroidAPP)
		ss := "as"
		md.Intermediate.WeakToken = &ss
		resp.extend(rctx, tar)

		assert.Equal(t, "20", string(rctx.Response.Header.Peek(retCode)))
		assert.Equal(t, "no-store", string(rctx.Response.Header.Peek("Cache-Control")))
		assert.Equal(t, "{}", string(tar.Result))
		assert.Equal(t, ss, *tar.Token)

		rctx = makeReqCtx()
		rctx.SetUserValue(constant.METADATA_CTX, md)
		rctx.Response.AppendBodyString("kkkk")
		resp.flags.timeInt = true
		md.Extension.Platform = string(bplatform.OpenAPI)
		resp.extend(rctx, tar)

		assert.Equal(t, "20", string(rctx.Response.Header.Peek(retCode)))
		assert.Equal(t, "no-store", string(rctx.Response.Header.Peek("Cache-Control")))
		assert.Equal(t, "kkkk", string(tar.Result))

		rctx = makeReqCtx()
		rctx.SetUserValue(constant.METADATA_CTX, md)
		md.Extension.Platform = string(bplatform.AndroidAPP)
		tar.SetExtInfo([]byte{})
		resp.flags.extInfoStr = true
		resp.extend(rctx, tar)

		assert.Equal(t, "20", string(rctx.Response.Header.Peek(retCode)))
		assert.Equal(t, "no-store", string(rctx.Response.Header.Peek("Cache-Control")))
		assert.Equal(t, "kkkk", string(tar.Result))
		assert.Equal(t, "\"\"", string(tar.GetExtInfo()))
	})
}

func makeReqCtx() *fasthttp.RequestCtx {
	return &fasthttp.RequestCtx{
		Request:  fasthttp.Request{Header: fasthttp.RequestHeader{}},
		Response: fasthttp.Response{Header: fasthttp.ResponseHeader{}},
	}
}
