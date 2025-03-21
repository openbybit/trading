package constant

type BGWCtxKey = string

const (
	NAMESPACE_KEY              = "namespace"
	GROUP_KEY                  = "group"
	DEFAULT_GROUP              = "DEFAULT_GROUP"
	BGW_GROUP                  = "BGW_GROUP"
	DEFAULT_NAMESPACE          = "public"
	VERSION_KEY                = "version"
	INTERFACE_KEY              = "interface"
	PATH_KEY                   = "path"
	METHOD_KEY                 = "method"
	SERVICE_KEY                = "service"
	METHODS_KEY                = "methods"
	TIMEOUT_KEY                = "timeout"
	ENABLED_KEY                = "enabled"
	TOKEN_KEY                  = "token"
	RELEASE_KEY                = "release"
	DESCRIPTOR_KEY             = "descriptor"
	PORT_KEY                   = "port"
	PROTOCOL_KEY               = "protocol"
	LOADBALANCE_KEY            = "loadbalance"
	SERVICE_INSTANCE_ENDPOINTS = "endpoints"
	SERVICE_FILTER_KEY         = "service.filter"
	REFERENCE_FILTER_KEY       = "reference.filter"
	TIMESTAMP_KEY              = "timestamp"
	REMOTE_TIMESTAMP_KEY       = "remote.timestamp"
	CLUSTER_KEY                = "cluster"
	WEIGHT_KEY                 = "weight"
	WARMUP_KEY                 = "warmup"
	RETRIES_KEY                = "retries"
	STICKY_KEY                 = "sticky"
	DEFAULT_EXECUTE_LIMIT      = "-1"
	PID_KEY                    = "pid"
	CATEGORY_KEY               = "category"
	REGISTRY_TIMEOUT_KEY       = "registry.timeout"
	REPORTER_PROMETHEUS        = "prometheus"
	SquidProxy                 = "SQUID_PROXY" // http proxy
	BGWNamespace               = "BGW_NAMESPACE"
	BGWConfigNamespace         = "bgw"
	BGWS                       = "bgws"
	ContextKey                 = "trace-context"
	BDOTEnv                    = "MY_ENV_NAME"
	BDOTProjectEnv             = "MY_PROJECT_ENV_NAME"
	EnvStage                   = "BSM_SERVICE_STAGE"
	EnableController           = "BGW_CONTROLLER"

	METADATA_CTX           = BGWCtxKey("metadata.ctx")          // context metadata key
	MetadataResponseCtx    = BGWCtxKey("metadata.response.ctx") // response context metadata key
	ContextFromFastHttpCtx = "general.ctx"                      // general context.Context from fasthttp.RequestCtx
	BgwResponseHandled     = "next-response-handled"            // grpc or http response had handled
	BgwRequestHandled      = "next-request-handled"             // grpc or http request had handled
	BgwRequestParsed       = "pre-request-parse"                // grpc or http request params had parsed
	BgwSelectMetas         = "ctx.select.metas"                 // select meta data
	BgwRateLimitInfo       = "ctx.limit.info"                   // rate limit info
	RiskSignBin            = "risk-sign-bin"

	BgwUpstreamCost      = "upstream-cost-time"       // upstream invoke cost time(ms)
	BgwUserAccountCost   = "user-account-cost-time"   // user account invoke cost time(ms)
	BgwUserStatusCost    = "user-status-cost-time"    // user status invoke cost time(ms)
	BgwUserApikeyCost    = "user-apikey-cost-time"    // user apikey invoke cost time(ms)
	BgwMasqCost          = "masq-cost-time"           // masq invoke cost time(ms)
	BgwUserRelationCost  = "user-relation-cost-time"  // user query relation invoke cost time(ms)
	BgwUserMemberTagCost = "user-membertag-cost-time" // user query member tag invoke cost time(ms)
	BgwRedisReqCost      = "redis-cost-time"          // redis request cost time(ms)
	BgwUpstreamCodes     = "upstream-codes"           // grpc response codes to observe
	BgwLimitDataCost     = "limit-data-cost-time"     // limit data invoke cost time(ms)

	BgwAPIResponseCodes      = "next-response-codes"       // grpc response codes
	BgwAPIResponseMessages   = "next-response-messages"    // grpc response messages
	BgwAPIResponseExtMaps    = "next-response-extras"      // grpc response extras
	BgwAPIResponseExtInfos   = "next-response-extra-infos" // grpc response extra infos
	BgwAPIResponseStatusCode = "next-response-status-code" // grpc response status code
	BgwAPIResponseExtCode    = "next-response-ext-code"
	BgwAPIResponseFlag       = "x-pt-next-response-flag" // grpc response flag 	// grpc response flag

	BgwgUpstreamCost = "bgwg_upstream_cost_time" // bgwg upstream invoke cost time(ms)
	BgwgMethod       = "bgwg_grpc_method"        // bgwg upstream invoke cost time(ms)
	BgwgRedisReqCost = "bgwg_redis_cost_time"    // bgwg redis request cost time(ms)
	BgwgUserReqCost  = "bgwg_user_cost_time"     // bgwg user req cost time(ms)

	BgwAPILimiterAllCost = "api-limiter-cost-time"         // bgw call openapi limiter cost time(ms)
	BgwComplianceCost    = "bgw_compliance_wall_cost_time" // compliance server rpc cost(ms)

	BlockTradeKey   = "blockTradeKey"
	BlockTradeTaker = "Taker"
	BlockTradeMaker = "Maker"
)
