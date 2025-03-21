package openapi

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"sync"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/config"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	"bgw/pkg/server/metadata/bizmetedata"
	"bgw/pkg/service"
	"bgw/pkg/service/ban"
	"bgw/pkg/service/geoip"
	ropenapi "bgw/pkg/service/openapi"
	"bgw/pkg/service/smp"
	"bgw/pkg/service/symbolconfig"
	"bgw/pkg/service/tradingroute"
	"bgw/pkg/service/user"

	"code.bydev.io/fbu/gateway/gway.git/gcore/sign"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"git.bybit.com/svc/mod/pkg/bplatform"
	ruser "git.bybit.com/svc/stub/pkg/pb/api/user"
)

type openapiRule struct {
	bizType             int
	skipAID             bool
	copyTrade           bool
	copyTradeInfo       *user.CopyTradeInfo
	unified             bool
	unifiedTrading      bool
	allowGuest          bool
	queryFallback       bool
	skipIpCheck         bool
	tradeCheck          bool
	tradeCheckCfg       *tradeCheckCfg
	batchTradeCheck     map[string]struct{}
	batchTradeCheckCfg  map[string]*tradeCheckCfg
	aidQuery            []string
	unifiedTradingCheck string
	utaProcessBan       bool
	symbolCheck         bool
	smpGroup            bool
	suiInfo             bool
	memberTags          []string
	aioFlag             bool
}

const (
	utaBan = "unified_trading_ban"
	comBan = "common_account_ban"

	suiProtoTag = "spot_sui_protocol_confirm"
	suiKycTag   = "spot_sui_kyc_white_user"
)

const (
	errBan = 10008
)

func Init() {
	filter.Register(filter.OpenAPIFilterKey, newOpenapi())
}

type tradeCheckCfg struct {
	SymbolField string `json:"symbolField"`
}

type openapi struct {
	as              user.AccountIface
	loader          WhiteListIPLoaderIface
	nacosLoader     WhiteListIPLoaderIface
	bc              string // banned countries
	once            sync.Once
	nacosLoaderOnce sync.Once
	rules           sync.Map
}

func newOpenapi() filter.Filter {
	as, err := user.NewAccountService()
	if err != nil {
		panic(err)
	}

	var bc string
	if rc := &config.Global.Geo; rc != nil {
		bc = strings.ToUpper(strings.TrimSpace(rc.GetOptions("openapi_banned_countries", "")))
		if bc != "" {
			_ = geoip.InitIPWhitelist(context.Background())
		}
	}
	return &openapi{
		as: as,
		bc: bc,
	}
}

// GetName returns the name of the filter
func (*openapi) GetName() string {
	return filter.OpenAPIFilterKey
}

// Do the filter
func (o *openapi) Do(next types.Handler) types.Handler {
	return func(c *types.Ctx) error {
		md := metadata.MDFromContext(c)

		// set openapi platform
		o.setPlatform(md)

		route := md.GetRoute()
		if !route.Valid() {
			return berror.NewInterErr("openapi invalid route")
		}

		rule := o.getRule(md.Route)
		if rule == nil {
			return berror.NewInterErr("invalid openapi rule", md.Route.String())
		}

		checkers, err := o.getCheckers(c, md, rule.allowGuest, rule.queryFallback)
		if err != nil {
			return err
		}

		if rule.allowGuest && checkers[0].GetAPIKey() == "" {
			glog.Debug(c, "openapi allowGuest mode, no apikey")
			return next(c)
		}

		if route.ACL.Group == constant.ResourceGroupBlockTrade {
			vr, err := o.verifyBlockTrade(c, checkers, md, route, rule)
			md.UID = vr.memberId // taker uid
			if err != nil {
				return err
			}
		} else {
			vr, err := o.verify(c, checkers[0], md, route, rule)
			if err != nil {
				md.UID = vr.memberId
				if rule.allowGuest {
					glog.Debug(c, "openapi allowGuest mode, get uid fail")
					return next(c)
				}
				return err
			}

			md.UID = vr.memberId
		}

		val, err := o.as.QueryMemberTag(service.GetContext(c), md.UID, user.UserSiteIDTag)
		if err != nil {
			glog.Error(c, "get user site id failed", glog.String("err", err.Error()), glog.Int64("uid", md.UID))
		}
		md.UserSiteID = val

		if rule.allowGuest {
			glog.Debug(c, "openapi allowGuest mode", glog.Int64("uid", md.UID))
			return next(c)
		}

		appName := route.GetAppName(c)
		if rule.copyTrade || rule.copyTradeInfo != nil {
			cs, err := user.GetCopyTradeService(config.Global.UserServicePrivate)
			if err != nil {
				glog.Error(c, "GetCopyTradeService error", glog.String("err", err.Error()))
				if rule.copyTradeInfo != nil && rule.copyTradeInfo.AllowGuest {
					glog.Debug(c, "openapi GetCopyTradeData allow guest")
					return next(c)
				}
			}

			ids, err := cs.GetCopyTradeData(service.GetContext(c), md.GetMemberID())
			if err != nil {
				if rule.copyTradeInfo != nil && rule.copyTradeInfo.AllowGuest {
					glog.Debug(c, "openapi GetCopyTradeData allow guest")
					return next(c)
				}
				glog.Error(c, "openapi GetCopyTradeData error", glog.String("error", err.Error()), glog.Int64("uid", md.GetMemberID()), glog.String("app", appName))
				return err
			}
			bizmetedata.WithCopyTradeMetadata(c, ids) // set to context
			glog.Debug(c, "openapi GetCopyTradeData", glog.Int64("uid", md.GetMemberID()),
				glog.Any("copyTrade", ids))
		}

		if err := o.handleAccountID(c, md, appName, rule); err != nil {
			return err
		}

		if err := o.unifiedTradingCheck(c, rule, md); err != nil {
			return err
		}

		if err := o.getSmpGroup(c, rule.smpGroup, md); err != nil {
			return err
		}

		if rule.suiInfo {
			o.handleSuiInfo(c, md)
		}
		if rule.aioFlag {
			md.AioFlag, _ = tradingroute.GetRouting().IsAioUser(c, md.GetMemberID())
		}

		o.handlerMemberTags(c, rule.memberTags, md)

		glog.Debug(c, fmt.Sprintf("openapi check ok. memberId[%d] aid[%d]", md.GetMemberID(), md.AccountID))

		return next(c)
	}
}

// Init route key and biz_type ( "--bizType=1" )
func (o *openapi) Init(ctx context.Context, args ...string) (err error) {
	if gm, err := geoip.NewGeoManager(); err != nil || gm == nil {
		return fmt.Errorf("NewGeoManager error:%w", err)
	}

	getIpCheckMgr().Init()
	o.nacosLoaderOnce.Do(func() {
		o.nacosLoader, err = newIpListNacosLoader(ctx)
	})
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return nil
	}

	rule, err := limiterFlagParse(ctx, args)
	if err != nil {
		return err
	}
	if rule.tradeCheck {
		if err := symbolconfig.InitSymbolConfig(); err != nil {
			return err
		}
	}
	o.rules.Store(args[0], &rule)

	return nil
}

func (o *openapi) getRule(route metadata.RouteKey) *openapiRule {
	value, ok := o.rules.Load(route.String())
	if ok {
		return value.(*openapiRule)
	}
	return nil
}

func (o *openapi) setPlatform(md *metadata.Metadata) {
	md.Extension.OpFrom = o.bopFrom(md)
	md.Extension.Platform = string(bplatform.OpenAPI)
	md.Extension.EPlatform = int32(bplatform.OpenAPI.CMDPlatform())
	md.Extension.OpPlatform = string(bplatform.OpenAPI)
	md.Extension.EOpPlatform = int32(bplatform.OpenAPI.OPPlatform())
}

func (o *openapi) bopFrom(md *metadata.Metadata) string {
	opFrom := "api"
	referer := md.Extension.XReferer
	if referer == "" {
		referer = md.Extension.Referer
	}
	if referer != "" {
		opFrom = opFrom + "." + referer
	}

	// cut off
	if len(opFrom) > 36 {
		opFrom = opFrom[:36]
	}
	return opFrom
}

func (o *openapi) getSmpGroup(ctx *types.Ctx, smpGroup bool, md *metadata.Metadata) error {
	if !smpGroup || md.UID <= 0 {
		return nil
	}
	g, err := smp.GetGrouper(service.GetContext(ctx))
	if err != nil {
		glog.Error(ctx, "getSmpGroup GetGrouper error", glog.Int64("uid", md.UID), glog.String("err", err.Error()))
		gmetric.IncDefaultError("openapi", "no_smp_group")
		return nil
	}

	group, err := g.GetGroup(ctx, md.UID)
	if err != nil {
		glog.Error(ctx, "getSmpGroup GetGroup error", glog.Int64("uid", md.UID), glog.String("err", err.Error()))
		gmetric.IncDefaultError("openapi", "smp_group_err")
	}
	md.SmpGroup = group
	return nil
}

func (o *openapi) handleSuiInfo(ctx context.Context, md *metadata.Metadata) {
	var err error
	defer func() {
		if err != nil {
			glog.Error(ctx, "get sui member tag failed", glog.String("err", err.Error()))
			gmetric.IncDefaultError("openapi", "sui_info_err")
		}
	}()
	if md.UID < 0 {
		return
	}
	val, err := o.as.QueryMemberTag(service.GetContext(ctx), md.UID, suiKycTag)
	if err != nil {
		return
	}
	md.SuiKyc = val
	val, err = o.as.QueryMemberTag(service.GetContext(ctx), md.UID, suiProtoTag)
	if err != nil {
		return
	}
	md.SuiProto = val
}

func (o *openapi) unifiedTradingCheck(c *types.Ctx, rule *openapiRule, md *metadata.Metadata) error {
	if md.UID <= 0 {
		return nil
	}

	if !rule.utaProcessBan && rule.unifiedTradingCheck == "" {
		return nil
	}

	tag, err := o.as.QueryMemberTag(service.GetContext(c), md.UID, user.UnifiedTradingTag)
	if err != nil {
		return nil
	}
	if rule.unifiedTradingCheck == utaBan && tag == user.UnifiedStateSuccess {
		return berror.NewBizErr(errBan, "uta banned")
	}

	if rule.unifiedTradingCheck == comBan && tag != user.UnifiedStateSuccess {
		return berror.NewBizErr(errBan, "common banned")
	}

	if rule.utaProcessBan && tag == user.UnifiedStateProcess {
		return berror.NewBizErr(errBan, "uta process banned")
	}

	return nil
}

var (
	UnifiedPrivateURLPrefix = []byte("/unified/v3/")
)

func (o *openapi) handleAccountID(c *types.Ctx, md *metadata.Metadata, appName string, rule *openapiRule) error {
	if !rule.skipAID && md.UID > 0 {
		err := o.getAid(c, md, appName, rule)
		if err != nil {
			return err
		}
	}

	if len(rule.aidQuery) == 0 {
		return nil
	}
	accoutIds, errs := o.as.GetBizAccountIDByApps(service.GetContext(c), md.UID, rule.bizType, rule.aidQuery...)
	aids := bizmetedata.NewAccountID()
	for i, err := range errs {
		if err != nil {
			return err
		}
		aids.SetAccountID(rule.aidQuery[i], accoutIds[i])
	}
	bizmetedata.WithAccountIDMetadata(c, aids)

	return nil
}

func (o *openapi) getAid(c *types.Ctx, md *metadata.Metadata, appName string, rule *openapiRule) error {
	if rule.unifiedTrading {
		uaTag, err := o.as.QueryMemberTag(service.GetContext(c), md.UID, user.UnifiedTradingTag)
		if err != nil {
			return err
		}
		md.UaTag = uaTag
		uaID, err := o.as.GetUnifiedTradingAccountID(service.GetContext(c), md.UID, rule.bizType)
		if err != nil {
			return err
		}
		if uaID > 0 {
			md.AccountID = uaID
			md.UnifiedTrading = true
			md.UaID = uaID
			return nil
		}
	}

	if rule.unified {
		umID, err := o.as.GetUnifiedMarginAccountID(service.GetContext(c), md.UID, rule.bizType)
		if err != nil {
			return err
		}
		if umID > 0 {
			md.AccountID = umID
			md.UnifiedMargin = true
			md.UnifiedID = umID
			return nil
		}

		if bytes.HasPrefix(c.Path(), UnifiedPrivateURLPrefix) {
			return berror.ErrUnifiedMarginAccess
		}
	}

	aid, err := o.as.GetAccountID(service.GetContext(c), md.UID, appName, rule.bizType)
	if err != nil {
		return err
	}
	md.AccountID = aid
	return nil
}

func (o *openapi) handlerMemberTags(ctx context.Context, tags []string, md *metadata.Metadata) {
	if len(tags) == 0 {
		return
	}
	temp := make(map[string]string)
	for _, tag := range tags {
		val, err := o.as.QueryMemberTag(service.GetContext(ctx), md.UID, tag)
		if err != nil {
			glog.Error(ctx, "query member tag failed", glog.Int64("uid", md.UID),
				glog.String("tag", tag), glog.String("err", err.Error()))
			temp[tag] = user.MemberTagFailed
			continue
		}
		temp[tag] = val
	}

	md.MemberTags = temp
}

var suiOnce sync.Once

func limiterFlagParse(ctx context.Context, args []string) (openapiRule, error) {
	var (
		rule            openapiRule
		copyTradeOrigin string
		aidQuery        string
	)

	parse := flag.NewFlagSet("openapi", flag.ContinueOnError)
	parse.IntVar(&rule.bizType, "biz_type", 1, "aid biz type, old flag") // compatible
	parse.IntVar(&rule.bizType, "bizType", 1, "aid biz type")
	parse.BoolVar(&rule.skipAID, "skipAID", false, "skip to get account id")
	parse.BoolVar(&rule.unified, "unified", false, "get unified margin account id")
	parse.BoolVar(&rule.copyTrade, "copyTrade", false, "get copyTrade leader id and follower id")
	parse.StringVar(&copyTradeOrigin, "copyTradeInfo", "", "get copyTrade leader id and follower id")
	parse.BoolVar(&rule.allowGuest, "allowGuest", false, "allow guest, try get uid")
	parse.BoolVar(&rule.queryFallback, "queryFallback", false, "fallback parse query string")
	parse.BoolVar(&rule.skipIpCheck, "skipIpCheck", false, "skip ip check")
	parse.BoolVar(&rule.tradeCheck, "tradeCheck", false, "trade banned check")
	parse.StringVar(&aidQuery, "accountIDQuery", "", "account id query list")
	parse.BoolVar(&rule.unifiedTrading, "unifiedTrading", false, "uta info")
	parse.StringVar(&rule.unifiedTradingCheck, "unifiedTradingCheck", "", "uta check")
	parse.BoolVar(&rule.utaProcessBan, "utaProcessBan", false, "uta process ban")
	parse.BoolVar(&rule.symbolCheck, "symbolCheck", false, "symbol check")
	parse.BoolVar(&rule.smpGroup, "smpGroup", false, "smp group")
	parse.BoolVar(&rule.suiInfo, "suiInfo", false, "sui info")
	parse.BoolVar(&rule.aioFlag, "aioFlag", false, "aio flag")
	var (
		memberTags            string
		batchTradecheck       string
		tradeCheckCfgStr      string
		batchTradeCheckCfgStr string
	)
	parse.StringVar(&memberTags, "memberTags", "", "member tags")
	parse.StringVar(&batchTradecheck, "batchTradeCheck", "", "batch trade check")
	parse.StringVar(&tradeCheckCfgStr, "tradeCheckCfg", "{}", "trade check cfg")
	parse.StringVar(&batchTradeCheckCfgStr, "batchTradeCheckCfg", "{}", "batch trade check cfg")

	if err := parse.Parse(args[1:]); err != nil {
		glog.Error(ctx, "openapi limiterFlagParse error", glog.String("error", err.Error()), glog.Any("args", args))
		return openapiRule{}, err
	}
	if rule.bizType <= 0 {
		rule.bizType = 1
	}

	cti, err := user.CopyTradeInfo{}.Parse(ctx, copyTradeOrigin)
	if err != nil {
		return openapiRule{}, err
	}
	rule.copyTradeInfo = cti

	if rule.copyTrade || rule.copyTradeInfo != nil {
		if _, err := user.GetCopyTradeService(config.Global.UserServicePrivate); err != nil {
			return openapiRule{}, err
		}
	}

	list := strings.Split(aidQuery, ",")
	for _, app := range list {
		if app = strings.TrimSpace(app); app != "" {
			rule.aidQuery = append(rule.aidQuery, app)
		}
	}

	if rule.smpGroup {
		_, err := smp.GetGrouper(ctx)
		if err != nil {
			glog.Error(ctx, "openapi smp GetGrouper error", glog.String("error", err.Error()))
		}
	}

	if rule.suiInfo {
		suiOnce.Do(func() {
			user.RegisterMemberTag(suiProtoTag)
			user.RegisterMemberTag(suiKycTag)
		})
	}

	if memberTags != "" {
		tags := strings.Split(memberTags, ",")
		rule.memberTags = tags
		for _, tag := range tags {
			user.RegisterMemberTag(tag)
		}
	}

	if rule.tradeCheck {
		if err := symbolconfig.InitSymbolConfig(); err != nil {
			return openapiRule{}, err
		}
		rule.tradeCheckCfg = &tradeCheckCfg{}
		if tradeCheckCfgStr != "" {
			if err = json.Unmarshal([]byte(tradeCheckCfgStr), rule.tradeCheckCfg); err != nil {
				return openapiRule{}, err
			}
		}
	}

	if batchTradecheck != "" {
		products := strings.Split(batchTradecheck, ",")
		val := make(map[string]struct{})
		for _, p := range products {
			val[p] = struct{}{}
		}
		rule.batchTradeCheck = val

		if batchTradeCheckCfgStr != "" {
			btcc := make(map[string]*tradeCheckCfg)
			if err = json.Unmarshal([]byte(batchTradeCheckCfgStr), &btcc); err != nil {
				return openapiRule{}, err
			}
			rule.batchTradeCheckCfg = btcc
		}
	}

	return rule, nil
}

type verifyResp struct {
	memberId                                 int64
	loginStatus, tradeStatus, withdrawStatus int32
}

func (o *openapi) verify(c *types.Ctx, checker Checker, md *metadata.Metadata, route metadata.RouteKey, rule *openapiRule) (resp verifyResp, err error) {
	md.APIKey = checker.GetAPIKey()
	ros, err := ropenapi.GetOpenapiService()
	if err != nil {
		return
	}

	member, err := ros.VerifyAPIKey(service.GetContext(c), checker.GetAPIKey(), md.Extension.XOriginFrom)
	if err != nil {
		if rule.allowGuest {
			return resp, nil
		}
		return
	}
	resp.memberId = member.MemberId
	md.UID = member.MemberId
	md.BrokerID = int32(member.BrokerId)
	if rule.allowGuest {
		return resp, nil
	}

	banSvc, err := ban.GetBanService()
	if err != nil {
		return
	}
	memberStatus, err := banSvc.CheckStatus(service.GetContext(c), md.UID)
	if err != nil {
		return
	}

	if rule.symbolCheck && member != nil && member.ExtInfo != nil && member.ExtInfo.Limits != nil {
		if err = o.verifySymbol(c, member.ExtInfo.Limits); err != nil {
			return
		}
	}

	if err = o.tradeCheck(c, rule, md, memberStatus); err != nil {
		return
	}

	// check banned countries
	if err = o.checkBannedCountries(c, rule, checker.GetClientIP()); err != nil {
		return
	}

	if err = o.checkIp(c, md, rule.skipIpCheck, member, checker.GetClientIP()); err != nil {
		return
	}

	if err = o.checkPermission(c, member.ExtInfo.Permissions, route.ACL); err != nil {
		return
	}

	if !md.WssFlag {
		signTyp := sign.TypeHmac
		if member.ExtInfo != nil {
			signTyp = sign.Type(member.ExtInfo.Flag)
		}
		if err = checker.VerifySign(c, signTyp, member.LoginSecret); err != nil {
			return
		}
	}

	resp = verifyResp{
		memberId:       member.MemberId,
		loginStatus:    memberStatus.LoginStatus,
		tradeStatus:    memberStatus.TradeStatus,
		withdrawStatus: memberStatus.WithdrawStatus,
	}
	return
}

// verifySymbol 限制交易对
func (o *openapi) verifySymbol(ctx *types.Ctx, symbolLimits *ruser.SymbolLimit) error {
	var symbolList map[string]string
	switch metadata.MDFromContext(ctx).Route.GetAppName(ctx) {
	case constant.AppTypeSPOT:
		symbolList = symbolLimits.GetSpot()
	case constant.AppTypeFUTURES:
		symbolList = symbolLimits.GetFuture()
	case constant.AppTypeOPTION:
		symbolList = symbolLimits.GetOptions()
	}

	glog.Debug(ctx, "verifySymbol", glog.Any("symbol", symbolList))

	if len(symbolList) == 0 {
		return nil
	}

	symbol := symbolconfig.GetSymbol(ctx)
	if symbol == "" {
		return berror.ErrSymbolLimited
	}

	if _, ok := symbolList[symbol]; !ok {
		return berror.ErrSymbolLimited
	}

	return nil
}

// verifyBlockTrade blocktrade get taker and maker member info and verfiy
func (o *openapi) verifyBlockTrade(ctx *types.Ctx, v3s [2]Checker, md *metadata.Metadata, route metadata.RouteKey, rule *openapiRule) (verifyResp, error) {
	// fixme 重复代码？
	vrtaker, err := o.verify(ctx, v3s[0], md, route, rule)
	if err != nil {
		ctx.SetUserValue(constant.BlockTradeKey, constant.BlockTradeTaker)
		return verifyResp{}, err
	}

	blockTradeMD := bizmetedata.BlockTrade{}
	defer func() {
		bizmetedata.WithBlockTradeMetadata(ctx, &blockTradeMD)
	}()

	blockTradeMD.TakerMemberId = vrtaker.memberId
	blockTradeMD.TakerLoginStatus = vrtaker.loginStatus
	blockTradeMD.TakerTradeStatus = vrtaker.tradeStatus
	blockTradeMD.TakerWithdrawStatus = vrtaker.withdrawStatus
	blockTradeMD.TakerAIOFlag, _ = tradingroute.GetRouting().IsAioUser(ctx, vrtaker.memberId)

	accoutIds, _ := o.as.GetBizAccountIDByApps(service.GetContext(ctx), vrtaker.memberId, rule.bizType, constant.AppTypeFUTURES, constant.AppTypeOPTION, constant.AppTypeSPOT)
	blockTradeMD.TakerFuturesAccountId, blockTradeMD.TakerOptionAccountId, blockTradeMD.TakerSpotAccountId = accoutIds[0], accoutIds[1], accoutIds[2]
	unifiedAccountId, _ := o.as.GetUnifiedMarginAccountID(service.GetContext(ctx), vrtaker.memberId, rule.bizType)
	blockTradeMD.TakerUnifiedAccountId = unifiedAccountId
	tutaid, _ := o.as.GetUnifiedTradingAccountID(service.GetContext(ctx), vrtaker.memberId, rule.bizType)
	blockTradeMD.TakerUnifiedTradingID = tutaid
	status, _ := o.as.QueryMemberTag(service.GetContext(ctx), vrtaker.memberId, user.UnifiedTradingTag)
	if status == user.UnifiedStateProcess {
		blockTradeMD.TakerTradeStatus = int32(ruser.MemberTradeStatus_TRADE_BAN)
	}

	if v3s[1] == nil {
		return vrtaker, nil // return taker member id
	}

	vrmaker, err := o.verify(ctx, v3s[1], md, route, rule)
	if err != nil {
		ctx.SetUserValue(constant.BlockTradeKey, constant.BlockTradeMaker)
		return verifyResp{}, err
	}

	blockTradeMD.MakerMemberId = vrmaker.memberId
	blockTradeMD.MakerLoginStatus = vrmaker.loginStatus
	blockTradeMD.MakerTradeStatus = vrmaker.tradeStatus
	blockTradeMD.MakerWithdrawStatus = vrmaker.withdrawStatus
	blockTradeMD.MakerAIOFlag, _ = tradingroute.GetRouting().IsAioUser(ctx, vrmaker.memberId)

	accoutIds, _ = o.as.GetBizAccountIDByApps(service.GetContext(ctx), vrmaker.memberId, rule.bizType, constant.AppTypeFUTURES, constant.AppTypeOPTION, constant.AppTypeSPOT)
	blockTradeMD.MakerFuturesAccountId, blockTradeMD.MakerOptionAccountId, blockTradeMD.MakerSpotAccountId = accoutIds[0], accoutIds[1], accoutIds[2]
	unifiedAccountId, _ = o.as.GetUnifiedMarginAccountID(service.GetContext(ctx), vrmaker.memberId, rule.bizType)
	blockTradeMD.MakerUnifiedAccountId = unifiedAccountId
	mutaid, _ := o.as.GetUnifiedTradingAccountID(service.GetContext(ctx), vrmaker.memberId, rule.bizType)
	blockTradeMD.MakerUnifiedTradingID = mutaid
	status, _ = o.as.QueryMemberTag(service.GetContext(ctx), vrmaker.memberId, user.UnifiedTradingTag)
	if status == user.UnifiedStateProcess {
		blockTradeMD.MakerTradeStatus = int32(ruser.MemberTradeStatus_TRADE_BAN)
	}

	return vrtaker, nil // return taker member id
}

func (o *openapi) tradeCheck(ctx *types.Ctx, rule *openapiRule, md *metadata.Metadata, banStatus *ban.UserStatusWrap) error {
	if rule.tradeCheck {
		return ban.TradeCheckSingleSymbol(ctx, md.Route.GetAppName(ctx), rule.tradeCheckCfg.SymbolField, md.UID, false, banStatus)
	}
	if len(rule.batchTradeCheck) > 0 {
		app := md.Route.GetAppName(ctx)
		_, ok := rule.batchTradeCheck[app]
		if !ok {
			return nil
		}
		sf := ""
		cfg, ok := rule.batchTradeCheckCfg[app]
		if ok {
			sf = cfg.SymbolField
		}
		banJson, err := ban.TradeCheckBatchSymbol(ctx, sf, app, md.UID, false, banStatus)
		if err != nil {
			return err
		}
		md.BatchBan = banJson
	}
	return nil
}
