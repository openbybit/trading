package biz_limiter

import "bgw/pkg/server/filter"

func Init() {
	filter.Register(filter.BizRateLimitFilterKey, newLimiter)
	filter.Register(filter.BizRateLimitFilterMEMO, newLimiterMemo) // route filter
	filter.Register(filter.BizRateLimitFilterV2Key, newV2)
}
