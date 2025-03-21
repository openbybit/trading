package constant

type Event = string

const (
	// EventEffectAPIKeyCache event topic
	EventEffectAPIKeyCache      Event = "effect_apikey_cache"       // 用户apikey缓存事件通知
	EventMemberBanned           Event = "cht-ban-msg"               // 用户封禁事件通知 "member_banned"
	EventMemberTagChange        Event = "member_tag_change"         // 用户tag事件通知
	EventSpecialSubMemberCreate Event = "special_sub_member_create" // 特殊子用户创建通知（目前有：copytrade）
	EventRateLimitChange        Event = "sync_rate_limit"           // 期货rate limit数据变更
)
