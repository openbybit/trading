package response

import (
	"bytes"
	"errors"
	"fmt"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"git.bybit.com/svc/mod/pkg/bplatform"
	"github.com/jhump/protoreflect/dynamic"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/bhttp"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/server/filter/response/version"
	"bgw/pkg/server/metadata"
)

var (
	errInvalidAnyResult  = errors.New("invalid any result")
	errInvalidResultType = errors.New("invalid result type")
	errInvalidResult     = errors.New("invalid result, get nil result")
	errMetaInBodyData    = berror.NewUpStreamErr(berror.UpstreamErrResponseInvalid, "metaInBody invalid data")
)

const anyResultKey = "result"

type handler func(ctx *types.Ctx, source resultCarrier, target Target) (err error)

// source is grpc response, related to target result
// process: grpc response -> get grpc result by field(1) -> set into target result
func handleAny(ctx *types.Ctx, source resultCarrier, target Target) (err error) {
	m, ok := source.(messager)
	if !ok {
		return errInvalidResultType
	}

	// !NOTE: get result by field name: result
	dm, ok := m.GetMessage().(*dynamic.Message)
	if !ok {
		return errInvalidResultType
	}
	value, err1 := dm.TryGetFieldByName(anyResultKey)
	if err1 != nil {
		msg := fmt.Sprintf("any response defined error, err = %s", err1.Error())
		galert.Error(ctx, msg)
		value = emptyJSON
	}

	switch t := value.(type) {
	case []byte:
		target.SetResult(t)
	case string:
		target.SetResult(cast.UnsafeStringToBytes(t))
	default:
		return errInvalidAnyResult
	}

	return
}

// compatible with: any + passThrough
// change into passThrough response version
func handlePassthrough(source resultCarrier) (target Target) {
	target = version.NewPassthroughResponse()
	raw, err := source.GetData()
	if err != nil {
		return nil
	}

	// set raw data
	target.SetResult(raw)
	return
}

// handle default response, source is grpc response
// target is (v1,v2) response version
// process: grpc response -> marshal into json -> set into target
func handleDefault(ctx *types.Ctx, source resultCarrier, target Target) (err error) {
	result, err := source.GetData()
	if err != nil {
		glog.Error(ctx, "marshal message failed", glog.String("err", err.Error()))
		return berror.NewUpStreamErr(berror.UpstreamErrResponseInvalid, "response marshal message failed", err.Error())
	}

	target.SetResult(result)
	if len(result) == 0 {
		target.SetResult(emptyJSON)
	}
	return
}

// handleConvert convert upstream V2Response to V1Response
func handleConvert(ctx *types.Ctx, source resultCarrier, target Target) (err error) {
	// only convert v2 to v1
	if target.Version() != version.VersionV1 {
		return errMetaInBodyData
	}

	raw, err := source.GetData()
	if err != nil {
		glog.Error(ctx, "marshal message failed", glog.String("err", err.Error()))
		return berror.NewUpStreamErr(berror.UpstreamErrResponseInvalid, "response marshal message failed", err.Error())
	}

	// get code from upstream response body
	resp := &version.V2Response{}
	if err := util.JsonUnmarshal(raw, resp); err != nil {
		glog.Error(ctx, "handleDefault metaInBody Unmarshal error", glog.String("error", err.Error()),
			glog.String("raw-data", cast.UnsafeBytesToString(raw)))
		return errMetaInBodyData
	}

	target.SetCode(resp.GetCode())
	target.SetMessage(resp.GetMessage())
	target.SetResult(resp.GetResult())
	target.SetExtInfo(resp.GetExtInfo())
	target.SetExtMap(resp.GetExtMap())

	return
}

const (
	rejected     = "Rejected"
	blockTradeId = "blockTradeId"
)

type blockTradeInfo struct {
	BlockTradeId string `json:"blockTradeId"`
	Status       string `json:"status"`
	RejectParty  string `json:"rejectParty"`
}

func handleBlockTradeError(ctx *types.Ctx) []byte {
	role, _ := ctx.UserValue(constant.BlockTradeKey).(string)
	if role == "" {
		role = constant.BlockTradeTaker
	}
	info := &blockTradeInfo{
		Status:      rejected,
		RejectParty: role,
	}
	switch {
	case ctx.IsGet():
		info.BlockTradeId = string(ctx.QueryArgs().Peek(blockTradeId))
	case bytes.Equal(ctx.Request.Header.ContentType(), bhttp.ContentTypePostForm):
		info.BlockTradeId = string(ctx.PostArgs().Peek(blockTradeId))
	default:
		info.BlockTradeId = util.JsonGetString(ctx.Request.Body(), blockTradeId)
	}

	d, err := util.JsonMarshal(info)
	if err != nil {
		glog.Info(ctx, "blockTradeInfo marshal error", glog.String("err", err.Error()))
		return emptyJSON
	}
	return d
}

func isOpenAPI(md *metadata.Metadata) bool {
	return bplatform.Client(md.Extension.Platform) == bplatform.OpenAPI
}
