package groute

import (
	"encoding/json"
	"strings"
)

const (
	idxMethodInvaid = iota
	idxMethodGet
	idxMethodPost
	idxMethodHead
	idxMethodPut
	idxMethodPatch
	idxMethodDelete
	idxMethodConnect
	idxMethodOptions
	idxMethodTrace
	idxMethodAny
	idxMethodMax
)

// methodIndexMap for http request, eg: GET
var methodIndexMap = map[string]int{
	"GET":     idxMethodGet,
	"HEAD":    idxMethodHead,
	"POST":    idxMethodPost,
	"PUT":     idxMethodPut,
	"PATCH":   idxMethodPatch,
	"DELETE":  idxMethodDelete,
	"CONNECT": idxMethodConnect,
	"OPTIONS": idxMethodOptions,
	"TRACE":   idxMethodTrace,
}

// AccountType 账户类型,对于一个用户,账户类型只能是一种,但对于配置,允许支持多种账户类型,例如unify,代表uma和uta
type AccountType uint8

const (
	AccountTypeUnknown        = AccountType(0)    // 未知类型
	AccountTypeNormal         = AccountType(0x01) // 普通用户
	AccountTypeUnifiedMargin  = AccountType(0x02) // 统一保证金账户
	AccountTypeUnifiedTrading = AccountType(0x04) // 统一交易账户

	// special account type flag
	AccountTypeUnified = AccountTypeUnifiedMargin | AccountTypeUnifiedTrading
	// AccountTypeAll 标识任意合法的account类型
	AccountTypeAll = AccountTypeUnknown
)

func (at AccountType) String() string {
	switch at {
	case AccountTypeNormal:
		return "normal"
	case AccountTypeUnifiedMargin:
		return "unified_margin"
	case AccountTypeUnifiedTrading:
		return "unified_trading"
	case AccountTypeUnified:
		return "unified"
	default:
		return "unknown"
	}
}

func (at AccountType) Is(target AccountType) bool {
	return at == target || (at&target != 0)
}

func ToAccountType(s string) AccountType {
	switch strings.ToLower(s) {
	case "normal":
		return AccountTypeNormal
	case "unified_margin", "uma":
		return AccountTypeUnifiedMargin
	case "unified_trading", "uta":
		return AccountTypeUnifiedTrading
	case "unified": // 任意统一账户
		return AccountTypeUnified
	default:
		return AccountTypeUnknown
	}
}

type PathType uint8

const (
	PATH_TYPE_UNKNOWN PathType = iota // 未知类型
	PATH_TYPE_STATIC                  // 静态路由
	PATH_TYPE_PRIFIX                  // 前缀路由,/*{name}结尾
	PATH_TYPE_PARAMS                  // 参数路由,/:{name}格式
)

// RouteType route type
type RouteType uint8

const (
	ROUTE_TYPE_UNKNOWN      RouteType = iota // 未知类型
	ROUTE_TYPE_DEFAULT                       // 独占路由exclusive
	ROUTE_TYPE_CATEGORY                      // 基于category路由
	ROUTE_TYPE_ACCOUNT_TYPE                  // 基于account type路由,逻辑同category,区别是从accountType字段提取数据
	ROUTE_TYPE_ALL_IN_ONE                    // all in one路由,需要兼容default和category路由
	ROUTE_TYPE_MAX                           //
)

// String to string
func (rt RouteType) String() string {
	switch rt {
	case ROUTE_TYPE_DEFAULT:
		return "default"
	case ROUTE_TYPE_CATEGORY:
		return "category"
	case ROUTE_TYPE_ACCOUNT_TYPE:
		return "account_type"
	case ROUTE_TYPE_ALL_IN_ONE:
		return "all_in_one"
	default:
		return "UNKNOWN"
	}
}

func (rt RouteType) MarshalJSON() ([]byte, error) {
	return json.Marshal(rt.String())
}

// ParseRouteType 从字符串中解析RouteType
func ParseRouteType(v string) RouteType {
	switch v {
	case "default":
		return ROUTE_TYPE_DEFAULT
	case "category":
		return ROUTE_TYPE_CATEGORY
	case "account_type":
		return ROUTE_TYPE_ACCOUNT_TYPE
	case "all_in_one":
		return ROUTE_TYPE_ALL_IN_ONE
	default:
		return ROUTE_TYPE_UNKNOWN
	}
}
