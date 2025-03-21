package constant

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
