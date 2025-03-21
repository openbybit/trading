package response

import (
	"bgw/pkg/common/constant"
	"github.com/tj/assert"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestCodeManager(t *testing.T) {

	t.Run("init single empty ext info", func(t *testing.T) {
		b := newSingleAPIResponseInfo()
		b.SetExtInfo([]byte{})
		assert.Nil(t, b.GetExtInfo())
	})
	t.Run("init with ctx", func(t *testing.T) {
		rctx := makeReqCtx()
		m := make(map[string]string)
		md := metadata.New(m)
		md.Append(constant.BgwAPIResponseCodes, "0", "10016", "10010", "222", "11")
		md.Append(constant.BgwAPIResponseMessages, "1", "2", "3", "4")
		md.Append(constant.BgwAPIResponseExtMaps, "1", "2", "3", "{}")
		md.Append(constant.BgwAPIResponseExtInfos, "1", "2", "3", "4")
		bb := getCodeFromCtx(rctx, nil, true)
		assert.IsType(t, newBatchAPIResponseInfo(), bb)
		bb = getCodeFromCtx(rctx, nil, false)
		assert.IsType(t, newSingleAPIResponseInfo(), bb)

		bb = getCodeFromCtx(rctx, md, false)
		assert.IsType(t, newSingleAPIResponseInfo(), bb)
		assert.Equal(t, int64(0), bb.GetCode())
		assert.Equal(t, "1", bb.GetMsg())
		assert.Equal(t, "1", string(bb.GetExtMap()))
		assert.Equal(t, "1", string(bb.GetExtInfo()))

		bb = getCodeFromCtx(rctx, md, true)
		bbb := bb.(*batchAPIResponseInfo)
		assert.IsType(t, newBatchAPIResponseInfo(), bbb)
		assert.Equal(t, int64(0), bbb.GetCode())
		assert.Equal(t, "1", bbb.GetMsg())
		assert.Equal(t, "1", string(bbb.GetExtMap()))
		assert.Equal(t, "1", string(bb.GetExtInfo()))

		assert.Equal(t, int64(10016), bbb.GetCodeByIdx(0))
		assert.Equal(t, "2", bbb.GetMessageByIdx(0))
		assert.Equal(t, "2", string(bbb.GetExtMapByIdx(0)))

		assert.Equal(t, int64(10010), bbb.GetCodeByIdx(1))
		assert.Equal(t, "3", bbb.GetMessageByIdx(1))
		assert.Equal(t, "3", string(bbb.GetExtMapByIdx(1)))

		assert.Equal(t, int64(222), bbb.GetCodeByIdx(2))
		assert.Equal(t, "4", bbb.GetMessageByIdx(2))

		assert.Equal(t, int64(11), bbb.GetCodeByIdx(3))
		assert.Equal(t, "OK", bbb.GetMessageByIdx(3))
		assert.Equal(t, []byte(nil), bbb.GetExtMapByIdx(3))

		assert.Nil(t, bbb.GetExtMapByIdx(4))
		assert.Equal(t, "", bbb.GetMessageByIdx(4))
		assert.Equal(t, int64(0), bbb.GetCodeByIdx(4))

		assert.Equal(t, "{\"list\":[{\"code\":10016,\"msg\":\"2\"},{\"code\":10010,\"msg\":\"3\"},{\"code\":222,\"msg\":\"4\"},{\"code\":11,\"msg\":\"OK\"}]}", string(bbb.GetBatchExtInfo()))
	})

}
