package metadata

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"time"

	"bgw/pkg/common/bhttp"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"google.golang.org/grpc/metadata"
)

type MetadataOption func(*Metadata)

type Metadata struct {
	ctx             context.Context
	UID             int64
	AccountID       int64
	UnifiedID       int64
	UaID            int64
	UnifiedMargin   bool // unifiedMargin
	UatID           int64
	UnifiedTrading  bool
	UaTag           string
	ReqInitAtE9     string        // tgw init time
	ReqInitTime     time.Time     // bgw init time
	ReqTime         time.Time     // bgw invoke time
	ReqCost         time.Duration // bgw invoke cost
	InvokeAddr      string
	InvokeService   string
	InvokeNamespace string
	InvokeGroup     string
	Version         string
	TraceID         string
	AKMTrace        string // akm trace id
	BrokerID        int32
	SiteID          string // 站点ID，区别于brokerID，brokerID表示用户归属，siteID表示流量入口
	UserNameSpace   string
	Route           RouteKey
	Extension       Extension
	Intermediate    Intermediate // intermediate data, not to upstream
	Method          string
	Path            string
	StaticRoutePath string // 路由配置里的原始path
	ContentType     string
	WssFlag         bool     // wss flag
	GrayTags        []string //
	KycCountry      string
	KycLevel        int32
	SmpGroup        int32
	SuiProto        string
	SuiKyc          string
	AuthExtInfo     map[string]string
	AioFlag         bool
	Category        *string
	BuyOI           *bool
	SellOI          *bool
	APIKey          string
	SiteConf        string
	MemberTags      map[string]string
	UserSiteID      string
	ParentUID       string
	MemberRelation  string
	IsDemoUID       bool
	LimitRule       string
	LimitValue      int
	LimitPeriod     int64
	Scope           []string
	ClientID        string
	BatchOI         string
	BatchBan        string
	BatchUaeSymbol  string
	BatchItems      int32
	OauthExtInfo    map[string]string
	BspInfo         string
}

type Extension struct {
	RemoteIP         string `json:"ip,omitempty"`
	Origin           string `json:"origin,omitempty"`
	Host             string `json:"host,omitempty"`
	URI              string `json:"uri,omitempty"`
	RawBody          []byte `json:"raw,omitempty"`
	UserAgent        string `json:"user_agent,omitempty"`
	OriginPlatform   string `json:"origin_pf,omitempty"` // origin platform
	Platform         string `json:"pf,omitempty"`        // platform
	EPlatform        int32  `json:"epf,omitempty"`
	OpPlatform       string `json:"opf,omitempty"` // opplatform
	EOpPlatform      int32  `json:"eopf,omitempty"`
	OpFrom           string `json:"opfrom,omitempty"` // opfrom
	XOriginFrom      string `json:"x_o_f,omitempty"`  // X-Origin-From
	ReqExpireTime    string `json:"exp,omitempty"`    // for mixer trading
	ReqExpireTimeE9  int64  `json:"expe9,omitempty"`  // for mixer trading
	AppName          string `json:"app,omitempty"`
	AppVersion       string `json:"ver,omitempty"`
	AppVersionCode   string `json:"ver_code,omitempty"`
	CountryISO       string `json:"ctry_iso,omitempty"`
	CountryISO3      string `json:"ctry_iso3,omitempty"`
	CurrencyCode     string `json:"fiat,omitempty"`
	CountryGeoNameID int64  `json:"ctry_id,omitempty"`
	CityName         string `json:"ct_name,omitempty"`
	CityGeoNameID    int64  `json:"ct_id,omitempty"`
	SubVisionId      int64  `json:"sub_vision_id,omitempty"`
	Referer          string `json:"ref,omitempty"`
	XReferer         string `json:"xref,omitempty"`
	DeviceID         string `json:"dev_id,omitempty"`
	Guid             string `json:"guid,omitempty"`
	Fingerprint      string `json:"fingerprint,omitempty"`
	GFingerprint     string `json:"gdfp,omitempty"` // global device fingerprint
	XClientTag       string `json:"x_client_tag,omitempty"`
	Language         string `json:"lang,omitempty"`
	GWSource         string `json:"gw_source,omitempty"`
}

type Intermediate struct {
	SecureToken *string // token refresh
	WeakToken   *string // token refresh
	Baggage     string  // baggage
	LaneEnv     string  // lane env label
	CallOrigin  string  // traffic origin
	RiskSign    string
}

type ACL struct {
	Group      string   `json:"group,omitempty" yaml:"group"`
	Permission string   `json:"permission,omitempty" yaml:"permission"`
	AllGroup   bool     `json:"all_group,omitempty" yaml:"all_group"`
	Groups     []string `json:"groups,omitempty" yaml:"groups"`
}

// RouteKey setup in context, used in filters
type RouteKey struct {
	Protocol    string
	AppName     string
	ModuleName  string
	Registry    string
	ServiceName string
	MethodName  string
	HttpMethod  string
	Group       string
	ACL         ACL
	Category    string
	AllApp      bool
	AppCfg      AppCfg
}

type AppCfg struct {
	Mapping bool              `yaml:"mapping" json:"mapping"`
	Key     string            `yaml:"key" json:"key"`
	Value   map[string]string `yaml:"value" json:"value"`
}

const keyCategory = "category"

// GetAppName get app name
func (r RouteKey) GetAppName(ctx *types.Ctx) string {
	if !r.AppCfg.Mapping {
		return r.AppName
	}

	// find in route key
	md := MDFromContext(ctx)
	if md.Route.Category != "" {
		return md.Route.Category
	}

	// find in request
	var (
		key = keyCategory
		ct  string
	)
	if r.AppCfg.Key != "" {
		key = r.AppCfg.Key
	}
	if md.Category != nil {
		ct = *md.Category
		if v, ok := r.AppCfg.Value[ct]; ok {
			return v
		}
		return ct
	}
	if ctx.IsPost() {
		if bytes.HasPrefix(ctx.Request.Header.ContentType(), bhttp.ContentTypePostForm) {
			ct = cast.UnsafeBytesToString(ctx.PostArgs().Peek(key))
		}
		ct = util.JsonGetString(ctx.PostBody(), key)
	} else {
		ct = cast.UnsafeBytesToString(ctx.QueryArgs().Peek(key))
	}
	md.Category = &ct

	return ct
}

// Valid check route key valid
func (r RouteKey) Valid() bool {
	return r.AppName != "" && r.ModuleName != "" && r.ServiceName != "" && r.Registry != "" && r.MethodName != "" && r.HttpMethod != ""
}

// AsApp get app
func (r RouteKey) AsApp() string {
	return r.AppName
}

// AsModule get app and module
func (r RouteKey) AsModule() string {
	return r.AppName + "." + r.ModuleName
}

// AsService get app,module,service,registry
func (r RouteKey) AsService() string {
	return r.AsModule() + "." + r.ServiceName + "." + r.Registry
}

// AsMethod get AsService and method
func (r RouteKey) AsMethod() string {
	return r.AsService() + "." + r.MethodName
}

// String full qualified
func (r RouteKey) String() string {
	return r.AsMethod() + "." + r.HttpMethod + "." + cast.ToString(r.AllApp)
}

// Parse option.hello.HelloService.HelloService.SayHelloGet.HTTP_METHOD_GET
// don't have ACL,Group message
func (r RouteKey) Parse(route string) RouteKey {
	routes := strings.Split(route, ".")
	if len(routes) == 7 {
		r.AppName = routes[0]
		r.ModuleName = routes[1]
		r.ServiceName = routes[2]
		r.Registry = routes[3]
		r.MethodName = routes[4]
		r.HttpMethod = routes[5]
		r.AllApp = cast.StringToBool(routes[6])
	}
	return r
}

// Copy will copy md
func (md *Metadata) Copy() *Metadata {
	if md == nil {
		return gMetadataBufferPool.Get()
	}
	m := *md
	return &m
}

var emptyMetadata = Metadata{}

// Reset will reset md
func (md *Metadata) Reset() {
	*md = emptyMetadata
}

// GetRoute get route key form md
func (md *Metadata) GetRoute() RouteKey {
	if md != nil {
		return md.Route
	}

	return RouteKey{}
}

// GetPlatform get platform from md
func (md *Metadata) GetPlatform() string {
	return strings.ToLower(md.Extension.Platform)
}

// GetStaticRoutePath get static path from md
func (md *Metadata) GetStaticRoutePath() string {
	return md.StaticRoutePath
}

// GetLanguage get language from md
func (md *Metadata) GetLanguage() string {
	return strings.ToLower(md.Extension.Language)
}

// WithUid set uid to md
func WithUid(uid int64) MetadataOption {
	return func(md *Metadata) {
		md.UID = uid
	}
}

// WithAccountID set account id to md
func WithAccountID(aid int64) MetadataOption {
	return func(md *Metadata) {
		md.AccountID = aid
	}
}

// WithTraceID set trace id to md
func WithTraceID(trace string) MetadataOption {
	return func(md *Metadata) {
		md.TraceID = trace
	}
}

// WithPath set path to md
func WithPath(path string) MetadataOption {
	return func(md *Metadata) {
		md.Path = path
	}
}

// WithRouteKey set route key to md
func WithRouteKey(route RouteKey) MetadataOption {
	return func(md *Metadata) {
		md.Route = route
	}
}

// WithCtx set context to md
func WithCtx(ctx context.Context) MetadataOption {
	return func(md *Metadata) {
		md.ctx = ctx
	}
}

// NewMetadata new metadata
func NewMetadata(options ...MetadataOption) *Metadata {
	md := gMetadataBufferPool.Get()

	for _, opt := range options {
		opt(md)
	}

	if md.ctx != nil {
		ContextWithMD(md.ctx, md)
	}

	return md
}

// Release release metadata
func Release(md *Metadata) {
	gMetadataBufferPool.Put(md)
}

// Request get request MD
func (md *Metadata) Request() metadata.MD {
	origin := make(metadata.MD, 20)

	origin.Set("reqinitate9", md.ReqInitAtE9)                        // traffic init time
	origin.Set("inittime", cast.Int64toa(md.ReqInitTime.UnixNano())) // bgw req time
	origin.Set("reqtime", cast.Int64toa(md.ReqTime.UnixNano()))      // bgw invoke time
	origin.Set("traceid", md.TraceID)                                // for access log and response
	origin.Set("awstraceid", md.AKMTrace)
	origin.Set("consumerApplication", constant.GWSource) // traffic sources

	// Extension valid
	ext := util.ToValidateGrpcHeader("Extension", util.ToJSONString(md.Extension))
	origin.Set("extension", ext)

	if md.WssFlag {
		origin.Set("consumerApplication", constant.BGWS)
	}
	if md.UID != 0 {
		origin.Set("memberid", cast.Int64toa(md.UID))
	}

	origin.Set("uta_status", md.UaTag)

	if md.AccountID != 0 {
		origin.Set("accountid", cast.Int64toa(md.AccountID))
	}
	if md.UnifiedMargin {
		origin.Set("unifiedmargin", "Y")
		origin.Set("accountid", cast.Int64toa(md.UnifiedID))
	}
	if md.UnifiedTrading {
		origin.Set("unifiedtrading", "Y")
		origin.Set("accountid", cast.Int64toa(md.UaID))
	}
	if md.BrokerID > 0 {
		origin.Set("broker_id", cast.Itoa(int(md.BrokerID)))
	}

	origin.Set("x-refer-site-id", md.SiteID)

	if md.UserNameSpace != "" {
		origin.Set("user_namespace", md.UserNameSpace)
	}
	if md.Path != "" {
		origin.Set("path", md.Path)
	}
	if md.Method != "" {
		origin.Set("method", md.Method)
	}
	if md.ContentType != "" {
		origin.Set("Content-Type", md.ContentType)
	}
	if md.Intermediate.Baggage != "" {
		origin.Set("baggage", md.Intermediate.Baggage)
	}
	if md.Intermediate.CallOrigin != "" {
		origin.Set("callOrigin", md.Intermediate.CallOrigin)
	}
	if md.Intermediate.RiskSign != "" {
		origin.Set(constant.RiskSignBin, md.Intermediate.RiskSign)
	}

	if len(md.GrayTags) > 0 {
		origin.Set("gray_tags", md.GrayTags...)
	}

	if md.KycCountry != "" {
		origin.Set("kyc_country", md.KycCountry)
	}

	if md.KycLevel > 0 {
		origin.Set("kyc_level", cast.Itoa(int(md.KycLevel)))
	}

	if md.SmpGroup > 0 {
		origin.Set("smp_group", cast.Itoa(int(md.SmpGroup)))
	}

	if md.SuiProto != "" {
		origin.Set("sui_proto", md.SuiProto)
	}

	if md.SuiKyc != "" {
		origin.Set("sui_kyc", md.SuiKyc)
	}

	if len(md.AuthExtInfo) > 0 {
		origin.Set("auth_ext_in-bin", util.ToJSONString(md.AuthExtInfo))
	}

	aioFlag := 0
	if md.AioFlag {
		aioFlag = 1
	}
	origin.Set("aio_flag", cast.Itoa(aioFlag))

	if md.BuyOI != nil {
		origin.Set("buy_open_limited", cast.ToString(*md.BuyOI))
	}

	if md.SellOI != nil {
		origin.Set("sell_open_limited", cast.ToString(*md.SellOI))
	}

	if md.SiteConf != "" {
		origin.Set("site_config", md.SiteConf)
	}

	if len(md.MemberTags) > 0 {
		origin.Set("member_tags", util.ToJSONString(md.MemberTags))
	}

	if md.UserSiteID != "" {
		origin.Set("user-site-id", md.UserSiteID)
	}

	if md.ParentUID != "" {
		origin.Set("parent_uid", md.ParentUID)
	}
	if md.IsDemoUID {
		origin.Set("is_demo_uid", cast.ToString(md.IsDemoUID))
	}

	if md.LimitRule != "" {
		origin.Set("limit_rule", md.LimitRule)
	}

	if md.LimitValue > 0 {
		origin.Set("limit_value", cast.ToString(md.LimitValue))
	}

	if md.LimitPeriod > 0 {
		origin.Set("limit_period", cast.ToString(md.LimitPeriod))
	}

	if md.BatchOI != "" {
		origin.Set("batch_oi", md.BatchOI)
	}

	if md.BatchBan != "" {
		origin.Set("batch_tradecheck", md.BatchBan)
	}

	if md.BatchUaeSymbol != "" {
		origin.Set("batch_uaesymbol", md.BatchUaeSymbol)
	}

	if md.BatchItems > 0 {
		origin.Set("items_count", cast.ToString(md.BatchItems))
	}

	if len(md.Scope) > 0 {
		origin.Set("oauth_scope", util.ToJSONString(md.Scope))
	}

	if md.ClientID != "" {
		origin.Set("oauth_client_id", md.ClientID)
	}

	if len(md.OauthExtInfo) > 0 {
		origin.Set("oauth_ext_info", util.ToJSONString(md.OauthExtInfo))
	}

	if md.BspInfo != "" {
		origin.Set("bsp_info", md.BspInfo)
	}

	// build biz extension metadata
	for _, ext := range GetMetadataExts() {
		for key, vs := range ext.Extract(md.ctx) {
			origin.Set(key, vs...)
		}
	}
	return origin
}

// WithContext set metadata in context
func (md *Metadata) WithContext(c *types.Ctx) *Metadata {
	c.SetUserValue(constant.METADATA_CTX, md)
	return md
}

// GetPartitionID get partition id
func (md *Metadata) GetPartitionID() int64 {
	return md.UID
}

// GetMemberID get member id
func (md *Metadata) GetMemberID() int64 {
	return md.UID
}

// GetAccountID get account id
func (md *Metadata) GetAccountID() int64 {
	return md.AccountID
}

// GetClientIP get client ip
func (md *Metadata) GetClientIP() string {
	return md.Extension.RemoteIP
}

// GetAppVersion get app version
func (md *Metadata) GetAppVersion() string {
	return md.Extension.AppVersion
}

// GetPlatForm get platform
func (md *Metadata) GetPlatForm() string {
	return md.Extension.Platform
}

// GetOpPlatForm get op platform
func (md *Metadata) GetOpPlatForm() string {
	return md.Extension.OpPlatform
}

// GetStateCode get state code
func (md *Metadata) GetStateCode() int {
	if md.ctx == nil {
		return 0
	}
	if c, ok := md.ctx.(*types.Ctx); ok {
		return c.Response.StatusCode()
	}
	return 0
}

var gMetadataBufferPool = &metadataPool{
	sync.Pool{
		New: func() interface{} {
			return new(Metadata)
		},
	},
}

type metadataPool struct {
	sync.Pool
}

// Get will get metadata from pool
func (m *metadataPool) Get() *Metadata {
	if md, ok := m.Pool.Get().(*Metadata); ok && md != nil {
		return md
	}
	return &Metadata{}
}

// Put will put metadata to pool
func (m *metadataPool) Put(md *Metadata) {
	if md == nil {
		return
	}
	defer m.Pool.Put(md)
	md.Reset()
}

type metadataCtx struct{}

// MDFromContext get metadata from context
func MDFromContext(ctx context.Context) *Metadata {
	var v interface{}
	if c, ok := ctx.(*types.Ctx); ok {
		v = c.UserValue(constant.METADATA_CTX)
	} else {
		v = ctx.Value(metadataCtx{})
	}
	if v != nil {
		if md, ok := v.(*Metadata); ok {
			return md
		}
	}

	return NewMetadata(WithCtx(ctx))
}

// ContextWithMD returns a new `context.Context` that holds a reference to
// the metadata. If metadata is nil, a new context without a metadata is returned.
func ContextWithMD(ctx context.Context, md *Metadata) context.Context {
	if md == nil {
		return ctx
	}
	if c, ok := ctx.(*types.Ctx); ok {
		c.SetUserValue(constant.METADATA_CTX, md)
	} else {
		return context.WithValue(ctx, metadataCtx{}, md)
	}
	return nil
}

// ContextWithTraceId set trace id to context
func ContextWithTraceId(ctx *types.Ctx, traceId string) {
	md := MDFromContext(ctx)
	md.TraceID = traceId
	md.WithContext(ctx)
}

const requestHandledBody = "request-handled-body"

// ContextWithRequestHandledBody set request handle mark
func ContextWithRequestHandledBody(ctx *types.Ctx, body []byte) {
	ctx.SetUserValue(requestHandledBody, body)
	ctx.SetUserValue(constant.BgwRequestHandled, true)
}

// RequestHandledBodyFromContext get request handle mark
func RequestHandledBodyFromContext(ctx *types.Ctx) ([]byte, bool) {
	if v := ctx.UserValue(constant.BgwRequestHandled); v != nil {
		if handled, ok := v.(bool); ok {
			return ctx.UserValue(requestHandledBody).([]byte), handled
		}
	}
	return nil, false
}

// ContextWithUpstreamCode set upstream code to context
func ContextWithUpstreamCode(ctx *types.Ctx, code ...int64) {
	ctx.SetUserValue(constant.BgwUpstreamCodes, code)
}

// CodeFromUpstreamContext get upstream code from context
func CodeFromUpstreamContext(ctx *types.Ctx) []int64 {
	if v := ctx.UserValue(constant.BgwUpstreamCodes); v != nil {
		if d, ok := v.([]int64); ok {
			return d
		}
	}
	return nil
}

type selectMetas struct{}

// ContextWithSelectMetas set select metas from context
func ContextWithSelectMetas(ctx context.Context, origin interface{}) context.Context {
	if c, ok := ctx.(*types.Ctx); ok {
		c.SetUserValue(constant.BgwSelectMetas, origin)
	} else {
		return context.WithValue(ctx, selectMetas{}, origin)
	}
	return nil
}

// MetasFromContext get metas from context
func MetasFromContext(ctx context.Context) interface{} {
	var v interface{}
	if c, ok := ctx.(*types.Ctx); ok {
		v = c.UserValue(constant.BgwSelectMetas)
	} else {
		v = ctx.Value(selectMetas{})
	}

	if v != nil {
		return v
	}
	return nil
}

type RateLimitInfo struct {
	RateLimitStatus  int `json:"rate_limit_status"`
	RateLimit        int `json:"rate_limit"`
	RateLimitResetMs int `json:"rate_limit_reset_ms"`
}

type rateLimitInfo struct{}

// ContextWithRateLimitInfo set rate limit info to context, for fbu v2 response
func ContextWithRateLimitInfo(ctx context.Context, r RateLimitInfo) context.Context {
	if c, ok := ctx.(*types.Ctx); ok {
		c.SetUserValue(constant.BgwRateLimitInfo, r)
	} else {
		return context.WithValue(ctx, rateLimitInfo{}, r)
	}
	return nil
}

// RateLimitInfoFromContext get rate limit info from context, for fbu v2 response
func RateLimitInfoFromContext(ctx context.Context) RateLimitInfo {
	var v interface{}
	if c, ok := ctx.(*types.Ctx); ok {
		v = c.UserValue(constant.BgwRateLimitInfo)
	} else {
		v = ctx.Value(rateLimitInfo{})
	}

	if r, ok := v.(RateLimitInfo); ok {
		return r
	}
	return RateLimitInfo{}
}
