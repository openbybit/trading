package response

import (
	"bgw/pkg/common/constant"
	"bgw/pkg/server/core/http"
	"bgw/pkg/server/filter/response/version"
	gmetadata "bgw/pkg/server/metadata"
	"github.com/tj/assert"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestTranslate(t *testing.T) {

	t.Run("do", func(t *testing.T) {

		tr := &translate{
			msgSource: MsgSource_Unknown,
		}

		md := metadata.New(make(map[string]string))
		md.Append(constant.BgwAPIResponseCodes, "12053")

		rctx := makeReqCtx()
		gmd := gmetadata.NewMetadata()
		gmd.Route.AppName = "futures2"
		gmd.Route.AppCfg.Mapping = false
		gmd.Extension.Language = "zh-my"
		rctx.SetUserValue(constant.METADATA_CTX, gmd)

		source := http.NewResult()
		source.SetMetadata(md)
		target := version.NewV2Response()
		target.SetMessage("msg")
		target.SetCode(12053)
		target.SetExtInfo([]byte("!2"))
		tr.do(rctx, source, target, true)
		assert.Equal(t, int64(12053), target.GetCode())
		assert.Equal(t, "OK", target.GetMessage())
		assert.Equal(t, "{}", string(target.GetExtInfo()))

		target.SetMessage("msg")
		target.SetCode(12053)
		target.SetExtInfo([]byte("!2"))
		md.Delete(constant.BgwAPIResponseCodes)
		tr.do(rctx, source, target, false)
		assert.Equal(t, int64(12053), target.GetCode())
		assert.Equal(t, "msg", target.GetMessage())
		assert.Equal(t, "!2", string(target.GetExtInfo()))

		tr.msgSource = MsgSource_BackEnd
		target.SetMessage("msg")
		target.SetCode(12053)
		target.SetExtInfo([]byte("!2"))
		tr.do(rctx, source, target, true)

		assert.Equal(t, int64(12053), target.GetCode())
		assert.Equal(t, "msg", target.GetMessage())
		assert.Equal(t, "!2", string(target.GetExtInfo()))

		gmd.Route.AppName = "futures"
		setCodeLoader("futures")
		target.SetMessage("msg")
		target.SetCode(12053)
		target.SetExtInfo([]byte("!2"))
		tr.do(rctx, source, target, false)
		assert.Equal(t, int64(12053), target.GetCode())
		assert.Equal(t, "请登录主账户后再继续下一步操作。", target.GetMessage())
		assert.Equal(t, "!2", string(target.GetExtInfo()))

		target.SetMessage("msg")
		target.SetCode(12053)
		target.SetExtInfo([]byte("!2"))
		md.Delete(constant.BgwAPIResponseCodes)
		tr.do(rctx, source, target, true)
		assert.Equal(t, int64(12053), target.GetCode())
		assert.Equal(t, "请登录主账户后再继续下一步操作。", target.GetMessage())
		assert.Equal(t, "!2", string(target.GetExtInfo()))

		target.SetMessage("msg")
		target.SetCode(12053)
		target.SetExtInfo([]byte("!2"))
		md.Append(constant.BgwAPIResponseCodes, "12053", "22")
		tr.do(rctx, source, target, true)
		assert.Equal(t, int64(12053), target.GetCode())
		assert.Equal(t, "请登录主账户后再继续下一步操作。", target.GetMessage())
		assert.Equal(t, "{\"list\":[{\"code\":22,\"msg\":\"OK\"}]}", string(target.GetExtInfo()))
	})
}
