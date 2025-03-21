package filter

import (
	"context"
	"fmt"
	"strings"

	"bgw/pkg/common/types"
)

// Filter Keys
const (
	InitFilterKey               = "FILTER_INIT"               // init filter, inject route config
	AccessLogFilterKey          = "FILTER_ACCESSLOG"          // global filter, log
	AuthFilterKey               = "FILTER_AUTH"               // route filter, use auth
	ContextFilterKey            = "FILTER_CONTEXT"            // route filter, request & response header
	ContextFilterKeyGlobal      = "FILTER_CONTEXT_GLOBAL"     // global filter, request & response header
	OpenAPIFilterKey            = "FILTER_OPENAPI"            // route filter, openapi
	CorsFilterKey               = "FILTER_CORS"               // global filter, cors
	RecoveryFilterKey           = "FILTER_RECOVERY"           // global filter, recovery
	ResponseFilterKey           = "FILTER_RESPONSE"           // route filter, response data & http status
	ExecuteLimitFilterKey       = "FILTER_EXECUTE"            // route filter, execute limit
	MetricsFilterKey            = "FILTER_METRICS"            // global filter, metrics, must be the last filter
	BizRateLimitFilterKey       = "FILTER_BIZ_LIMITER"        // business rate limiter
	BizRateLimitFilterV2Key     = "FILTER_BIZ_LIMITER_V2"     // business rate limiter V2
	QPSRateLimitFilterKey       = "FILTER_QPS_LIMITER"        // qps limiter
	QPSRateLimitFilterKeyGlobal = "FILTER_QPS_LIMITER_GLOBAL" // global qps limiter
	TracingFilterKey            = "FILTER_TRACING"            // global filter, tracing filter
	GEOFilterKey                = "FILTER_GEO_IP"             // global filter, geo ip filter
	IPRateLimitFilterKey        = "FILTER_IP_LIMITER"         // route filter, ip rate limiter
	AntiReplayFilterKey         = "FILTER_ANTI_REPLAY"        // route filter, anti replay
	RequestFilterKey            = "FILTER_REQUEST"            // route filter, request data
	SignatureFilterKey          = "FILTER_SIGNATURE"          // route filter, signature
	APILimiterKey               = "FILTER_API_LIMITER"        // api limiter
	ComplianceWallFilterKey     = "FILTER_COMPLIANCE_WALL"    // compliance wall
	GrayFilterKey               = "FILTER_GRAY"               // gray filter
	OpenInterestFilterKey       = "FILTER_OPEN_INTEREST"      // openinterest filter
	BizRateLimitFilterMEMO      = "FILTER_BIZ_LIMITER_MEMO"
	CryptionFilterKey           = "FILTER_BIZ_CRYPTION"
	BanFilterKey                = "FILTER_BIZ_BAN"
	BspFilterKey                = "FILTER_BSP"
)

var (
	filters = make(map[string]interface{})
)

// Filter is a filter interface
type Filter interface {
	Do(types.Handler) types.Handler
	GetName() string
}

// NewFilter is a filter constructor
type NewFilter = func() Filter

// Initializer parse arguments
type Initializer interface {
	Init(ctx context.Context, args ...string) error
}

// Func is a filter function
type Func func(types.Handler) types.Handler

// Do will call next handler
func (f Func) Do(next types.Handler) types.Handler {
	return f(next)
}

// Register will store the @filter and @name
func Register(name string, filter interface{}) {
	filters[strings.ToLower(name)] = filter
}

// GetFilter get filter by name,
// and parse args if impl Parser interface
func GetFilter(ctx context.Context, name string, args ...string) (Filter, error) {
	f, ok := filters[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("filter name not register: %s", name)
	}

	var filter Filter
	switch t := f.(type) {
	case NewFilter: // route filter
		filter = t()
	case Filter: // global filter
		filter = t
	default:
		return nil, fmt.Errorf("filter type is invalid: %s -> %T", name, t)
	}

	if init, ok := filter.(Initializer); ok {
		err := init.Init(ctx, args...)
		if err != nil {
			return nil, fmt.Errorf("filter Init error: %s -> %w", name, err)
		}
	}

	return filter, nil
}
