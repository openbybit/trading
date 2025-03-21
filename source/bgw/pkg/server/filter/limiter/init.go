package limiter

import "bgw/pkg/server/filter"

func Init() {
	filter.Register(filter.QPSRateLimitFilterKeyGlobal, newGlobalLimiter()) // global filter
	filter.Register(filter.QPSRateLimitFilterKey, new)                      // route filter
	filter.Register(filter.IPRateLimitFilterKey, newIPLimiter)              // route filter
}
