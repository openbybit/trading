package response

import (
	"bytes"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"google.golang.org/grpc/metadata"
)

const (
	apiError = "api error, can not get code or msg from context"
)

var (
	_ batchAPI    = &batchAPIResponseInfo{}
	_ apiResponse = &batchAPIResponseInfo{}
	_ apiResponse = &singleAPIResponseInfo{}
)

type apiResponse interface {
	GetCode() int64
	SetCode(code int64)
	GetMsg() string
	SetMsg(msg string)
	GetExtMap() []byte
	SetExtMap(ext []byte)
	GetExtInfo() []byte
	SetExtInfo(ext []byte)
}

type batchAPI interface {
	GetCodeByIdx(idx int) int64
	GetMessageByIdx(idx int) string
	GetExtMapByIdx(idx int) []byte
	GetBatchExtInfo() []byte
}

type singleAPIResponseInfo struct {
	Code    int64
	Message string
	ExtMap  []byte
	ExtInfo []byte
}

type codeInfo struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

type batchCodesInfo struct {
	List []codeInfo `json:"list,omitempty"`
}

func newSingleAPIResponseInfo() *singleAPIResponseInfo {
	return &singleAPIResponseInfo{Message: messageOK}
}

func (s *singleAPIResponseInfo) SetCode(code int64) {
	s.Code = code
}

func (s *singleAPIResponseInfo) SetMsg(msg string) {
	s.Message = msg
}

func (s *singleAPIResponseInfo) SetExtMap(ext []byte) {
	s.ExtMap = ext
}

func (s *singleAPIResponseInfo) SetExtInfo(ext []byte) {
	s.ExtInfo = ext
}

func (s *singleAPIResponseInfo) GetCode() int64 {
	return s.Code
}

func (s *singleAPIResponseInfo) GetMsg() string {
	return s.Message
}

func (s *singleAPIResponseInfo) GetExtMap() []byte {
	return s.ExtMap
}

func (s *singleAPIResponseInfo) GetExtInfo() []byte {
	if len(s.ExtInfo) == 0 {
		return nil
	}
	return s.ExtInfo
}

type batchAPIResponseInfo struct {
	singleAPIResponseInfo
	Codes    []int64
	Messages []string
	ExtMaps  [][]byte
	ExtInfos [][]byte
}

func newBatchAPIResponseInfo() *batchAPIResponseInfo {
	return &batchAPIResponseInfo{
		singleAPIResponseInfo: singleAPIResponseInfo{Message: messageOK},
	}
}

func (b *batchAPIResponseInfo) GetCode() int64 {
	return b.singleAPIResponseInfo.GetCode()
}

func (b *batchAPIResponseInfo) GetCodeByIdx(idx int) int64 {
	if idx >= len(b.Codes) {
		return 0
	}
	return b.Codes[idx]
}

func (b *batchAPIResponseInfo) GetMsg() string {
	return b.singleAPIResponseInfo.GetMsg()
}

func (b *batchAPIResponseInfo) GetMessageByIdx(idx int) string {
	if idx >= len(b.Messages) {
		return ""
	}
	return b.Messages[idx]
}

func (b *batchAPIResponseInfo) GetExtMap() []byte {
	return b.singleAPIResponseInfo.GetExtMap()
}

func (b *batchAPIResponseInfo) GetExtMapByIdx(idx int) []byte {
	if idx >= len(b.ExtMaps) {
		return nil
	}
	m := b.ExtMaps[idx]
	if bytes.Equal(m, []byte("{}")) {
		return nil
	}
	return m
}

func (b *batchAPIResponseInfo) GetExtInfo() []byte {
	return b.singleAPIResponseInfo.GetExtInfo()
}

func (b *batchAPIResponseInfo) GetBatchExtInfo() []byte {
	extInfo := emptyJSON
	var codesInfo batchCodesInfo
	for i := 0; i < len(b.Codes); i++ {
		item := codeInfo{
			Code: b.Codes[i],
			Msg:  b.Messages[i],
		}
		codesInfo.List = append(codesInfo.List, item)
	}
	if ext := util.ToJSON(codesInfo); len(ext) > 0 {
		extInfo = ext
	}
	return extInfo
}

func getCodeFromCtx(ctx *types.Ctx, md metadata.MD, batch bool) apiResponse {
	ar := parseCodeFromCtx(ctx, md, batch)
	if ar == nil {
		return &singleAPIResponseInfo{Message: apiError}
	}

	if a, ok := ar.(*batchAPIResponseInfo); ok {
		// batch api
		remain := len(a.Codes) - len(a.Messages)
		for i := 0; i < remain; i++ {
			a.Messages = append(a.Messages, messageOK)
		}
	}

	return ar
}

func parseCodeFromCtx(ctx *types.Ctx, md metadata.MD, batch bool) apiResponse {
	var ar apiResponse
	if batch {
		ar = newBatchAPIResponseInfo()
	} else {
		ar = newSingleAPIResponseInfo()
	}

	if md == nil {
		return ar
	}

	x := md[constant.BgwAPIResponseCodes]
	for i, code := range x {
		if i == 0 {
			ar.SetCode(cast.AtoInt64(code))
			continue
		}
		if !batch {
			break
		}
		a, ok := ar.(*batchAPIResponseInfo)
		if ok {
			a.Codes = append(a.Codes, cast.AtoInt64(code))
		}
	}

	m := md[constant.BgwAPIResponseMessages]
	for i, msg := range m {
		if i == 0 {
			ar.SetMsg(msg)
			continue
		}
		if !batch {
			break
		}
		a, ok := ar.(*batchAPIResponseInfo)
		if ok {
			a.Messages = append(a.Messages, msg)
		}
	}

	e := md[constant.BgwAPIResponseExtMaps]
	for i, extMap := range e {
		if i == 0 {
			ar.SetExtMap([]byte(extMap))
			continue
		}
		if !batch {
			break
		}
		a, ok := ar.(*batchAPIResponseInfo)
		if ok {
			a.ExtMaps = append(a.ExtMaps, []byte(extMap))
		}
	}

	ins := md[constant.BgwAPIResponseExtInfos]
	for i, in := range ins {
		if i == 0 {
			ar.SetExtInfo([]byte(in))
			continue
		}
		if !batch {
			break
		}
		a, ok := ar.(*batchAPIResponseInfo)
		if ok {
			a.ExtInfos = append(a.ExtInfos, []byte(in))
		}
	}

	glog.Debug(ctx, "api code and message, extra", glog.String("path", cast.UnsafeBytesToString(ctx.Path())), glog.Any("codes", x),
		glog.Any("messages", m), glog.Any("extras-map", e), glog.Any("extras-info", ins))
	return ar
}
