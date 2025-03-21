package berror

const (
	HttpStatusOK    = 200
	HttpNotFound    = 404
	HttpServerError = 500
)

const (
	ErrCodeInvalidRequest  = 10001
	ErrCodeOpenAPIApiKey   = 10003
	ErrCodeUserTradeBanned = 11108
)

const (
	TimeoutErr          int64 = 10000
	SystemInternalError int64 = 10016
)

const userBanMsg = "User has been banned."

var (
	ErrDefault                    = NewInterErr("Internal System Error.")
	ErrRouteKeyInvalid            = NewInterErr("illegle route config, route key is invalid")
	ErrTimeout                    = NewBizErr(10000, "Server Timeout")
	ErrParams                     = NewBizErr(10001, "Request parameter error.")
	ErrRouteMissing               = NewBizErr(10001, "Request route does not exist.")
	ErrInvalidSymbol              = NewBizErr(10001, "The requested symbol is invalid.")
	ErrInvalidRequest             = NewBizErr(10001, "Request parameter error.")
	ErrOpenAPITimestamp           = NewBizErr(10002, "The request time exceeds the time window range.")
	ErrOpenAPIApiKey              = NewBizErr(10003, "API key is invalid.")
	ErrOpenAPISign                = NewBizErr(10004, "Error sign, please check your signature generation algorithm.")
	ErrOpenAPIPermission          = NewBizErr(10005, "Permission denied, please check your API key permissions.")
	ErrVisitsLimit                = NewBizErr(10006, "Too many visits. Exceeded the API Rate Limit.")
	ErrAuthVerifyFailed           = NewBizErr(10007, "User authentication failed.")
	ErrOpenAPIUserLoginBanned     = NewBizErr(10008, userBanMsg)                                                                                                // login ban
	ErrOpenAPIUserUsdtAllBanned   = NewBizErr(11008, userBanMsg)                                                                                                // usdt ban
	ErrOpenAPIUserAllBanned       = NewBizErr(11108, userBanMsg)                                                                                                // all ban
	ErrTradeCheckUTAProcessBanned = NewBizErr(10030, "Your positions cannot be traded until the account upgrade process is completed. Please try again later.") // uta process ban in trade check
	ErrCountryBanned              = NewBizErr(10009, "IP has been banned.")
	ErrInvalidIP                  = NewBizErr(10010, "Unmatched IP, please check your API key's bound IP addresses.")
	ErrRouteNotFound              = NewBizErr(10017, "Route not found.")
	ErrAuthenticationAccess       = NewBizErr(10019, "This feature is still in beta test.")
	ErrUnifiedMarginAccess        = NewBizErr(10020, "The API can only be accessed by unified account users.")
	ErrZoneVIPSelectorInvalidUID  = NewBizErr(10021, "Your account is not in vip list.")
	ErrComplianceRuleTriggered    = NewBizErr(10024, "The product or service you are seeking to access is not available to you due to regulatory restrictions.If you believe you are a permitted customer of this product or service,please reach out to us at support@bybit.com")
	ErrSymbolLimited              = NewBizErr(10029, "The requested symbol is not whitelisted.")
	ErrBadSign                    = NewBizErr(10031, "cryption sign is invalid")
	ErrOpenAPIApiKeyExpire        = NewBizErr(33004, "Your api key has expired.")
)
