package berror

import (
	"strings"
)

// baseErr is base error
type baseErr struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

func newBaseErr(c int64, msg ...string) baseErr {
	return baseErr{
		Code:    c,
		Message: strings.Join(msg, ", "),
	}
}

// Error error
func (b baseErr) Error() string {
	return b.Message
}

// GetCode get code
func (b baseErr) GetCode() int64 {
	return b.Code
}

// String string
func (b baseErr) String() string {
	return b.Error()
}

// BizErr is error of business
type BizErr struct {
	baseErr
}

// NewBizErr new biz error
func NewBizErr(code int64, msg ...string) error {
	be := newBaseErr(code, msg...)
	return BizErr{be}
}

// WithMessage can only use to process biz error.
func WithMessage(err error, msg string) error {
	be, _ := err.(BizErr)
	be.Message = msg + ": " + err.Error()
	return be
}

// GetErrCode get error code
func GetErrCode(err error) (code int64) {
	switch e := err.(type) {
	case BizErr:
		code = e.GetCode()
	case InterErr:
		code = e.GetCode()
	case UpStreamErr:
		code = e.GetCode()
	default:
		code = 5000
	}
	return
}

// InterErr is error of bgw
type InterErr struct {
	baseErr
}

// NewInterErr new inter error
func NewInterErr(msg ...string) InterErr {
	be := newBaseErr(5000, msg...)
	return InterErr{be}
}
