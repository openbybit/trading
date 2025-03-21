package version

import (
	"encoding/json"
	"time"

	"bgw/pkg/common/util"
)

// VersionV2 version name
const VersionV2 = "v2"

var (
	emptyJSON = []byte("{}")
)

// NewV2Response new v2 response
func NewV2Response() *V2Response {
	return &V2Response{
		RetExtInfo: emptyJSON,
	}
}

// V2Response token and time will set in response header
type V2Response struct {
	Code       int64           `json:"retCode"`
	Message    string          `json:"retMsg"`
	Result     json.RawMessage `json:"result"`
	RetExtMap  json.RawMessage `json:"retExtMap,omitempty"` // for translate template
	RetExtInfo json.RawMessage `json:"retExtInfo"`          // for extension info to customer
	Time       int64           `json:"time"`
	Error      string          `json:"error,omitempty"`
}

// Version get version
func (r *V2Response) Version() string { return VersionV2 }

// GetCode get code
func (r *V2Response) GetCode() int64 { return r.Code }

// SetCode set code
func (r *V2Response) SetCode(code int64) { r.Code = code }

// GetMessage get message
func (r *V2Response) GetMessage() string { return r.Message }

// SetMessage set message
func (r *V2Response) SetMessage(msg string) { r.Message = msg }

// GetResult get result
func (r *V2Response) GetResult() []byte { return r.Result }

// SetResult set result
func (r *V2Response) SetResult(data []byte) { r.Result = data }

// GetExtInfo get ext info
func (r *V2Response) GetExtInfo() []byte { return r.RetExtInfo }

// SetExtInfo set ext info
func (r *V2Response) SetExtInfo(ext json.RawMessage) { r.RetExtInfo = ext }

// SetExtMap set ext map
func (r *V2Response) SetExtMap(ext json.RawMessage) { r.RetExtMap = ext }

// GetExtMap get ext map
func (r *V2Response) GetExtMap() []byte { return r.RetExtMap }

func (r *V2Response) SetExtCode(c string) {}

// SetError set error
func (r *V2Response) SetError(e string) { r.Error = e }

// Marshal do marshal response
func (r *V2Response) Marshal() ([]byte, error) {
	r.Time = time.Now().UnixNano() / 1e6
	return util.JsonMarshal(r)
}
