package version

import "encoding/json"

// VersionPassthrough version name
const VersionPassthrough = "passthrough"

// NewPassthroughResponse new passthrough response
func NewPassthroughResponse() *PassthroughResponse {
	return &PassthroughResponse{}
}

// PassthroughResponse passthrough struct response
type PassthroughResponse struct {
	result []byte
}

// SetAPIInfo set api info

// Version get version
func (p *PassthroughResponse) Version() string { return VersionPassthrough }

// GetCode get code
func (p *PassthroughResponse) GetCode() int64 { return 0 }

// SetCode set code
func (p *PassthroughResponse) SetCode(code int64) {
	// do nothing, not support
}

// GetMessage get message
func (p *PassthroughResponse) GetMessage() string { return "" }

// SetMessage set message
func (p *PassthroughResponse) SetMessage(msg string) {
	// do nothing, not support
}

// GetExtInfo get ext info
func (p *PassthroughResponse) GetExtInfo() []byte { return nil }

// SetExtInfo set ext info
func (p *PassthroughResponse) SetExtInfo(ext json.RawMessage) {
	// do nothing, not support
}

// SetExtMap set ext map
func (p *PassthroughResponse) SetExtMap(ext json.RawMessage) {
	// do nothing, not support
}

// GetExtMap get ext map
func (p *PassthroughResponse) GetExtMap() []byte { return nil }

func (p *PassthroughResponse) SetExtCode(c string) {
	// do nothing, not support
}

// Marshal do marshal response
func (p *PassthroughResponse) Marshal() ([]byte, error) { return p.result, nil }

// SetResult set result
func (p *PassthroughResponse) SetResult(data []byte) { p.result = data }

// GetResult get result
func (p *PassthroughResponse) GetResult() []byte { return p.result }
