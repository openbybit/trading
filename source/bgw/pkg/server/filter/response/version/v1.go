package version

import (
	"encoding/json"

	"bgw/pkg/common/util"
	"bgw/pkg/server/metadata"
)

// VersionV1 version name
const VersionV1 = "v1"

// V1Response v1 response
type V1Response struct {
	Code    int64           `json:"ret_code"`
	Message string          `json:"ret_msg"`
	Result  json.RawMessage `json:"result"`
	ExtCode string          `json:"ext_code"`
	ExtInfo json.RawMessage `json:"ext_info"`
	ExtMap  json.RawMessage `json:"ext_map,omitempty"`
	Time    interface{}     `json:"time_now"`
	Token   *string         `json:"token,omitempty"`
	Error   string          `json:"error,omitempty"`
	*metadata.RateLimitInfo
}

// NewV1Response new v1 response
func NewV1Response() *V1Response {
	return &V1Response{}
}

// SetToken set token
func (r *V1Response) SetToken(token *string) { r.Token = token }

// SetLimit set limit
func (r *V1Response) SetLimit(info metadata.RateLimitInfo) { r.RateLimitInfo = &info }

// GetExtMap get ext map
func (r *V1Response) GetExtMap() []byte { return r.ExtMap }

// Version get version
func (r *V1Response) Version() string { return VersionV1 }

// GetCode get code
func (r *V1Response) GetCode() int64 { return r.Code }

// SetCode set code
func (r *V1Response) SetCode(code int64) { r.Code = code }

// GetMessage get message
func (r *V1Response) GetMessage() string { return r.Message }

// SetMessage set message
func (r *V1Response) SetMessage(msg string) { r.Message = msg }

// GetResult get result
func (r *V1Response) GetResult() []byte { return r.Result }

// SetResult set result
func (r *V1Response) SetResult(data []byte) { r.Result = data }

// GetExtInfo get ext info
func (r *V1Response) GetExtInfo() []byte { return r.ExtInfo }

// SetExtInfo set ext info
func (r *V1Response) SetExtInfo(ext json.RawMessage) { r.ExtInfo = ext }

// SetExtMap set ext map
func (r *V1Response) SetExtMap(ext json.RawMessage) { r.ExtMap = ext }

// SetTime set time
func (r *V1Response) SetTime(t interface{}) { r.Time = t }

// SetError set error
func (r *V1Response) SetError(e string) { r.Error = e }

func (r *V1Response) SetExtCode(c string) { r.ExtCode = c }

// Marshal do marshal response
func (r *V1Response) Marshal() ([]byte, error) {
	return util.JsonMarshal(r)
}
