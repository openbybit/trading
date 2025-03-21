package response

import (
	"bgw/pkg/server/core/grpc"
	"bgw/pkg/server/filter/response/version"
	"errors"
	"git.bybit.com/svc/stub/pkg/svc/masquerade"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/tj/assert"
	"google.golang.org/grpc/metadata"
	"testing"

	"bgw/pkg/common/bhttp"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
)

func TestBlockTrade(t *testing.T) {
	ctx := &types.Ctx{}
	ctx.SetUserValue(constant.BlockTradeKey, "Maker")
	d := handleBlockTradeError(ctx)
	t.Log(string(d))

	ctx = &types.Ctx{}
	d = handleBlockTradeError(ctx)
	t.Log(string(d))

	// query
	ctx.Request.URI().SetQueryString("blockTradeId=abc&coin=USDT")
	d = handleBlockTradeError(ctx)
	t.Log(string(d))

	// json
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetBodyRaw([]byte(`{"blockTradeId":"xyz","coin":"USDT"}`))
	d = handleBlockTradeError(ctx)
	t.Log(string(d))

	// form
	ctx.Request.SetBodyRaw([]byte("blockTradeId=ght&coin=USDT"))
	ctx.Request.Header.SetContentTypeBytes(bhttp.ContentTypePostForm)
	d = handleBlockTradeError(ctx)
	t.Log(string(d))
}

func TestHandle(t *testing.T) {
	t.Run("handleDefault", func(t *testing.T) {
		rctx := makeReqCtx()
		source := &mockC{
			e: errors.New("11"),
		}
		target := version.NewV2Response()
		err := handleDefault(rctx, source, target)
		assert.Equal(t, "response marshal message failed, 11", err.Error())

		source = &mockC{
			r: "123",
		}
		err = handleDefault(rctx, source, target)
		assert.NoError(t, err)
		assert.Equal(t, "123", string(target.GetResult()))

		source = &mockC{
			r: "",
		}
		err = handleDefault(rctx, source, target)
		assert.NoError(t, err)
		assert.Equal(t, "{}", string(target.GetResult()))
	})
	t.Run("handleConvert", func(t *testing.T) {
		rctx := makeReqCtx()
		source := &mockC{
			e: errors.New("11"),
		}
		targetV2 := version.NewV2Response()
		err := handleConvert(rctx, source, targetV2)
		assert.Equal(t, errMetaInBodyData, err)

		targetV1 := version.NewV1Response()
		err = handleConvert(rctx, source, targetV1)
		assert.Equal(t, "response marshal message failed, 11", err.Error())

		source = &mockC{
			r: "123",
		}
		targetV1 = version.NewV1Response()
		err = handleConvert(rctx, source, targetV1)
		assert.Equal(t, errMetaInBodyData, err)

		source = &mockC{
			r: "{}",
		}
		targetV1 = version.NewV1Response()
		err = handleConvert(rctx, source, targetV1)
		assert.NoError(t, err)

	})
	t.Run("handleAny", func(t *testing.T) {
		rctx := makeReqCtx()
		source := &mockC{
			e: errors.New("11"),
		}
		targetV2 := version.NewV2Response()
		err := handleAny(rctx, source, targetV2)
		assert.Equal(t, errInvalidResultType, err)

		sourceRpc := grpc.NewResult()
		sourceRpc.SetMessage(&masquerade.AuthResponse{})
		targetV1 := version.NewV1Response()
		err = handleAny(rctx, sourceRpc, targetV1)
		assert.Equal(t, errInvalidResultType, err)

		sourceRpc = grpc.NewResult()
		// fixme 构造正确的message
		msg, e := dynamic.AsDynamicMessage(&masquerade.AuthResponse{})
		assert.NoError(t, e)
		sourceRpc.SetMessage(msg)
		targetV1 = version.NewV1Response()
		err = handleAny(rctx, sourceRpc, targetV1)
		assert.Equal(t, "{}", string(targetV1.GetResult()))
	})
	t.Run("handlePassthrough", func(t *testing.T) {
		source := &mockC{
			e: errors.New("11"),
		}
		tt := handlePassthrough(source)
		assert.Nil(t, tt)

		source = &mockC{
			r: "121221",
		}
		tt = handlePassthrough(source)
		assert.NotNil(t, tt)
		assert.Equal(t, "121221", string(tt.GetResult()))
	})
}

type mockC struct {
	r string
	e error
}

func (m mockC) GetStatus() int {
	return 0
}

func (m mockC) GetData() ([]byte, error) {
	return []byte(m.r), m.e
}

func (m mockC) Metadata() metadata.MD {
	return nil
}

func (m mockC) Close() error {
	return nil
}
