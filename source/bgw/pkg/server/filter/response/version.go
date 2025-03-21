package response

import (
	"encoding/json"

	"bgw/pkg/server/filter/response/version"
	"bgw/pkg/server/metadata"
)

const (
	formatKey = "_sp_response_format"
	retCode   = "Ret_Code"
	messageOK = "OK"
)

var (
	_ Target        = &version.V1Response{}
	_ Target        = &version.V2Response{}
	_ Target        = &version.PassthroughResponse{}
	_ withRateLimit = &version.V1Response{}
	_ withtoken     = &version.V1Response{}
	_ replaceTime   = &version.V1Response{}
	_ withError     = &version.V1Response{}
	_ withError     = &version.V2Response{}

	hump      = []byte("hump")
	portugal  = []byte("portugal")
	emptyJSON = []byte("{}")
)

type withRateLimit interface {
	SetLimit(info metadata.RateLimitInfo)
}

type withtoken interface {
	SetToken(token *string)
}

type replaceTime interface {
	SetTime(t interface{})
}

type withError interface {
	SetError(string)
}

// Target is the interface for response
type Target interface {
	Version() string
	GetCode() int64
	SetCode(code int64)
	GetMessage() string
	SetMessage(msg string)
	GetExtInfo() []byte
	SetExtInfo(ext json.RawMessage)
	GetExtMap() []byte
	SetExtMap(ext json.RawMessage)
	SetExtCode(c string)
	SetResult([]byte)
	GetResult() []byte
	Marshal() ([]byte, error)
}
