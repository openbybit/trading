package berror

// UpStreamErr is error of upstream
type UpStreamErr struct {
	baseErr
}

// NewUpStreamErr new upstream error
func NewUpStreamErr(code int64, msg ...string) UpStreamErr {
	if code == 0 {
		code = 6000
	}
	be := newBaseErr(code, msg...)
	return UpStreamErr{be}
}

// 目前对6000～6999之间的错误按照百位数进行了划分
const (
	// 6000～6099 上游业务报错
	UpstreamErrInstanceNotFound     int64 = 6001
	UpstreamErrInstanceConnFailed   int64 = 6002
	UpstreamErrInvokerFailed        int64 = 6003
	UpstreamErrResponseInvalid      int64 = 6004
	UpstreamErrDemoInstanceNotFound int64 = 6005
	// UpstreamErrInvokerBreaker for errors need to be broken
	UpstreamErrInvokerBreaker int64 = 6006

	// 6100~6199 masq服务调用报错
	UpstreamErrMasqInvokeFailed  int64 = 6100
	UpstreamErrOauthInvokeFailed int64 = 6101

	// 6200~6299 user service调用报错
	UpstreamErrUserServiceInvokeFailed int64 = 6200

	// 6300～6399 封禁服务调用报错
	UpstreamErrBanServiceInvokeFailed int64 = 6300
)

// 内部错误
const (
	// 熔断错误码
	InternalErrBreaker = 5001
)
