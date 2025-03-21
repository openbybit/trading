package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	oauthv1 "code.bydev.io/cht/backend-bj/user-service/buf-user-gen.git/pkg/bybit/oauth/v1"

	"github.com/valyala/fasthttp"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	"bgw/pkg/server/metadata/bizmetedata"
	"bgw/pkg/service"
	"bgw/pkg/service/ban"
	"bgw/pkg/service/dynconfig"
	"bgw/pkg/service/masque"
	"bgw/pkg/service/symbolconfig"
	"bgw/pkg/service/user"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"git.bybit.com/svc/go/pkg/bconst"
	"git.bybit.com/svc/mod/pkg/bproto"
)

const (
	userToken        = "UserToken"
	oauthToken       = "authorization"
	parentUID        = "pid"
	subMemberTypeKey = "typ"
	demoSubMember    = "MEMBER_RELATION_TYPE_DEMO"
	stationType      = "Station-Type"
)

func Init() {
	filter.Register(filter.AuthFilterKey, newAuth())
}

const (
	utaBan = "unified_trading_ban"
	comBan = "common_account_ban"

	suiProtoTag = "spot_sui_protocol_confirm"
	suiKycTag   = "spot_sui_kyc_white_user"

	ExtInfoSiteID = "sid"
)

const (
	errBan = 10008
)

const logKeyReqToken = "req-token"
const logKeySecureKey = "secure-key"
const logKeyBrokerId = "broker-id"

type authRule struct {
	bizType             int
	allowGuest          bool
	refreshToken        bool
	weakAuth            bool
	skipAID             bool
	copyTrade           bool
	copyTradeInfo       *user.CopyTradeInfo
	tradeCheck          bool
	tradeCheckCfg       *tradeCheckCfg
	batchTradeCheck     map[string]struct{}
	batchTradeCheckCfg  map[string]*tradeCheckCfg
	unified             bool // for unified margin
	unifiedTrading      bool
	aidQuery            []string
	unifiedTradingCheck string
	utaProcessBan       bool
	utaStatus           bool
	suiInfo             bool
	memberTags          []string
	oauth               bool
}

type tradeCheckCfg struct {
	SymbolField string `json:"symbolField"`
}

type auth struct {
	ms    masque.MasqueIface
	as    user.AccountIface
	os    masque.OauthIface
	rules sync.Map
}

// new auth filter.
func newAuth() filter.Filter {
	ms, err := masque.GetMasqueService()
	if err != nil {
		panic(err)
	}

	as, err := user.NewAccountService()
	if err != nil {
		panic(err)
	}

	os, err := masque.GetOAuthService()
	if err != nil {
		panic(err)
	}

	return &auth{
		ms: ms,
		as: as,
		os: os,
	}
}

func (a *auth) GetName() string {
	return filter.AuthFilterKey
}

func (a *auth) Do(next types.Handler) types.Handler {
	return func(c *types.Ctx) error {
		var now = time.Now()
		md := metadata.MDFromContext(c)
		route := md.GetRoute()
		if !route.Valid() {
			glog.Error(c, "user route error", glog.Any("route", route))
			return berror.NewInterErr("auth user route error")
		}
		appName := route.GetAppName(c)
		rule := a.getRule(md.Route)
		if rule == nil {
			glog.Error(c, "invalid auth rule", glog.Any("route", md.Route))
			return berror.NewInterErr("invalid auth rule")
		}

		if rule.oauth {
			glog.Debug(c, "enter oauth logic", glog.Any("rule", rule))
			token := util.DecodeHeaderValue(c.Request.Header.Peek(oauthToken))
			oresp, err := a.checkOAuth(c, rule, md, token)
			if err != nil {
				c.Response.SetStatusCode(fasthttp.StatusUnauthorized)
				shortToken, hashToken := a.hashToken(token)
				glog.Info(c, "checkOAuth fail, request token info", glog.String(logKeyReqToken, shortToken), glog.Bool("rule.oauth", rule.oauth),
					glog.String("hash-token", hashToken), glog.Int64(logKeyBrokerId, int64(md.BrokerID)),
					glog.String("err", err.Error()), glog.String("xof", md.Extension.XOriginFrom))
				return err
			}
			md.AuthExtInfo = make(map[string]string)
			md.OauthExtInfo = oresp.ExtInfo
			md.Scope = oresp.Scope
			md.ClientID = oresp.ClientId
			md.UID = oresp.MemberId
		} else {
			secureToken := config.GetSecureTokenKey()
			token := string(c.Request.Header.Cookie(secureToken))
			if token == "" {
				token = util.DecodeHeaderValue(c.Request.Header.Peek(userToken))
				glog.Debug(c, "get secure-token failed, user token from header", glog.String(logKeySecureKey, secureToken))
			}
			resp, err := a.checkLogin(c, rule, md, token)
			if err != nil {
				shortToken, hashToken := a.hashToken(token)
				glog.Info(c, "checkLogin fail, request token info", glog.String(logKeyReqToken, shortToken),
					glog.String("hash-token", hashToken), glog.Int64(logKeyBrokerId, int64(md.BrokerID)),
					glog.String("err", err.Error()), glog.String(logKeySecureKey, secureToken), glog.String("xof", md.Extension.XOriginFrom))
				return err
			}
			shortToken, hashToken := a.hashToken(token)
			glog.Debug(c, "request token info", glog.Int64("uid", md.UID), glog.String(logKeyReqToken, shortToken),
				glog.String("req-hash-token", hashToken), glog.Int64(logKeyBrokerId, int64(md.BrokerID)), glog.String(logKeySecureKey, secureToken),
				glog.String("xof", md.Extension.XOriginFrom))

			md.UserNameSpace = resp.NameSpace
			md.AuthExtInfo = resp.ExtInfo
			md.UID = resp.UserId
			if resp.HasNextToken {
				secureToken := resp.GetNextToken()
				weakToken := resp.GetWeakToken()
				md.Intermediate.SecureToken = &secureToken
				md.Intermediate.WeakToken = &weakToken
				shortToken, hashToken = a.hashToken(secureToken)
				shortToken1, hashToken1 := a.hashToken(weakToken)
				glog.Info(c, "next token info", glog.Int64("uid", md.UID), glog.Int64(logKeyBrokerId, int64(md.BrokerID)),
					glog.String("secure-token", shortToken), glog.String("secure-hash-token", hashToken),
					glog.String("weak-token", shortToken1), glog.String("weak-hash-token", hashToken1))
			}
		}

		if pu, ok := md.AuthExtInfo[parentUID]; ok {
			md.ParentUID = pu
		}
		if t, ok := md.AuthExtInfo[subMemberTypeKey]; ok {
			md.MemberRelation = t
			md.IsDemoUID = t == demoSubMember && md.ParentUID != ""
		}

		if rule.copyTrade || rule.copyTradeInfo != nil {
			cs, err := user.GetCopyTradeService(config.Global.UserServicePrivate)
			if err != nil {
				glog.Info(c, "GetCopyTradeService error", glog.String("err", err.Error()))
				if rule.copyTradeInfo != nil && rule.copyTradeInfo.AllowGuest {
					glog.Debug(c, "auth GetCopyTradeData allow guest")
					return next(c)
				}
			}

			ids, err := cs.GetCopyTradeData(service.GetContext(c), md.UID)
			if err != nil {
				if rule.copyTradeInfo != nil && rule.copyTradeInfo.AllowGuest {
					glog.Debug(c, "auth GetCopyTradeData allow guest")
					return next(c)
				}
				glog.Info(c, "auth GetCopyTradeData error", glog.String("error", err.Error()), glog.Int64("uid", md.UID), glog.String("app", appName))
				return err
			}
			bizmetedata.WithCopyTradeMetadata(c, ids) // set to context
			glog.Debug(c, "auth get GetCopyTrade id ok", glog.Any("copyTrade", ids),
				glog.Duration("cost", time.Since(now)), glog.Int64("uid", md.UID))
		}

		if err := a.handleAccountID(c, md, appName, rule); err != nil {
			return err
		}

		if err := a.unifiedTradingCheck(c, rule, md); err != nil {
			return err
		}

		if err := a.tradeCheck(c, rule, md); err != nil {
			return err
		}

		if rule.suiInfo {
			a.handleSuiInfo(c, md)
		}

		a.handlerMemberTags(c, rule.memberTags, md)

		val, err := a.as.QueryMemberTag(service.GetContext(c), md.UID, user.UserSiteIDTag)
		if err != nil {
			glog.Error(c, "get user site id failed", glog.String("err", err.Error()), glog.Int64("uid", md.UID))
			if md.AuthExtInfo != nil {
				val = md.AuthExtInfo["ExtInfoSiteID"]
			}
		}
		md.UserSiteID = val

		glog.Debug(c, "auth check ok",
			glog.Duration("cost", time.Since(now)), glog.Bool("refresh", rule.refreshToken), glog.Int64(logKeyBrokerId, int64(md.BrokerID)),
			glog.Bool("allowGuest", rule.allowGuest), glog.Bool("skipAID", rule.skipAID), glog.Bool("unified", rule.unified),
			glog.Bool("copyTrade", rule.copyTrade), glog.Int64("uid", md.UID), glog.String("app", appName), glog.Int64("bizType", int64(rule.bizType)),
		)

		return next(c)
	}
}

func (a *auth) hashToken(reqToken string) (shortToken string, hashToken string) {
	if len(reqToken) >= 8 {
		shortToken = reqToken[len(reqToken)-8:]
	}
	if tn := strings.SplitN(reqToken, ".", 3); len(tn) == 3 {
		subToken := tn[2]
		start := int(math.Ceil((float64(len(subToken)) / 3) * 2))
		subToken = subToken[start:]
		hashToken = util.ToMD5(subToken)
	}
	return
}

func (a *auth) unifiedTradingCheck(c *types.Ctx, rule *authRule, md *metadata.Metadata) error {
	if md.UID <= 0 {
		return nil
	}

	if !rule.utaProcessBan && rule.unifiedTradingCheck == "" {
		return nil
	}

	tag, err := a.as.QueryMemberTag(service.GetContext(c), md.UID, user.UnifiedTradingTag)
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

func (a *auth) handleAccountID(c *types.Ctx, md *metadata.Metadata, appName string, rule *authRule) error {
	if !rule.skipAID && md.UID > 0 {
		err := a.getAid(c, md, appName, rule)
		if err != nil {
			return err
		}
	}

	if len(rule.aidQuery) == 0 {
		return nil
	}
	accoutIds, errs := a.as.GetBizAccountIDByApps(service.GetContext(c), md.UID, rule.bizType, rule.aidQuery...)
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

func (a *auth) getAid(c *types.Ctx, md *metadata.Metadata, appName string, rule *authRule) error {
	if rule.unifiedTrading {
		uaTag, err := a.as.QueryMemberTag(service.GetContext(c), md.UID, user.UnifiedTradingTag)
		if err != nil {
			return err
		}
		md.UaTag = uaTag
		uaID, err := a.as.GetUnifiedTradingAccountID(service.GetContext(c), md.UID, rule.bizType)
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

	if rule.utaStatus && md.UaTag == "" {
		uaTag, err := a.as.QueryMemberTag(service.GetContext(c), md.UID, user.UnifiedTradingTag)
		if err != nil {
			return err
		}
		md.UaTag = uaTag
	}

	if rule.unified {
		umID, err := a.as.GetUnifiedMarginAccountID(service.GetContext(c), md.UID, rule.bizType)
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

	aid, err := a.as.GetAccountID(service.GetContext(c), md.UID, appName, rule.bizType)
	if err != nil {
		return err
	}
	md.AccountID = aid
	return nil
}

func (a *auth) handleSuiInfo(ctx context.Context, md *metadata.Metadata) {
	var err error
	defer func() {
		if err != nil {
			glog.Error(ctx, "get sui member tag failed", glog.String("err", err.Error()))
			gmetric.IncDefaultError("auth", "sui_info_err")
		}
	}()

	if md.UID < 0 {
		return
	}
	val, err := a.as.QueryMemberTag(service.GetContext(ctx), md.UID, suiKycTag)
	if err != nil {
		return
	}
	md.SuiKyc = val
	val, err = a.as.QueryMemberTag(service.GetContext(ctx), md.UID, suiProtoTag)
	if err != nil {
		return
	}
	md.SuiProto = val
}

func (a *auth) handlerMemberTags(ctx context.Context, tags []string, md *metadata.Metadata) {
	if len(tags) == 0 {
		return
	}
	temp := make(map[string]string)
	for _, tag := range tags {
		val, err := a.as.QueryMemberTag(service.GetContext(ctx), md.UID, tag)
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

// Init implement filter.Initializer
// args like: []string{"routeKey", "--bizType=1", "--allowGuest=true", "--refreshToken=true"}
func (a *auth) Init(ctx context.Context, args ...string) error {
	if len(args) == 0 {
		return nil
	}

	rule, err := limiterFlagParse(ctx, args)
	if err != nil {
		return err
	}
	a.rules.Store(args[0], &rule)

	_, err = dynconfig.GetBrokerIdLoader(ctx)
	return err
}

func (a *auth) checkLogin(ctx *types.Ctx, rule *authRule, md *metadata.Metadata, token string) (resp *masque.AuthResponse, err error) {
	var (
		secureToken = ""
		weakToken   = ""
	)

	if token == "" {
		if rule.allowGuest {
			glog.Debug(ctx, "no token, auth allow guest")
			return new(masque.AuthResponse), nil
		}
		glog.Debug(ctx, "no token, auth not allow guest")
		return nil, berror.ErrAuthVerifyFailed
	}

	pc := metadata.MDFromContext(ctx).GetPlatform()
	originUrl := md.Extension.Referer + md.Path
	if rule.refreshToken {
		resp, err = a.ms.MasqueTokenInvoke(service.GetContext(ctx), pc, token, originUrl, masque.RefreshToken)
	} else if rule.weakAuth {
		resp, err = a.ms.MasqueTokenInvoke(service.GetContext(ctx), pc, token, originUrl, masque.WeakAuth)
	} else {
		resp, err = a.ms.MasqueTokenInvoke(service.GetContext(ctx), pc, token, originUrl, masque.Auth)
	}
	if err != nil {
		return
	}

	if !bproto.Ok(resp.Error) {
		glog.Debug(ctx, "rpc error code", glog.Int64("code", int64(resp.Error.ErrorCode)))
		switch resp.Error.ErrorCode {
		case bconst.RpcErrorCodeFailed:
			if !rule.allowGuest {
				md.Intermediate.SecureToken = &secureToken
				md.Intermediate.WeakToken = &weakToken
				return nil, berror.ErrAuthVerifyFailed
			}
			return resp, nil
		default:
			return nil, berror.NewUpStreamErr(berror.UpstreamErrMasqInvokeFailed, "masq Auth resp error", resp.Error.String())
		}
	}

	if md.BrokerID <= 0 {
		md.BrokerID = resp.BrokerId
		brokerMgr, err1 := dynconfig.GetBrokerIdLoader(ctx)
		if err1 != nil {
			glog.Error(ctx, "GetBrokerIdLoader error", glog.Int64("uid", resp.UserId), glog.NamedError("err", err1))
			galert.Error(ctx, "GetBrokerIdLoader error", galert.WithField("err", err1))
			return resp, nil
		}
		st := string(ctx.Request.Header.Peek(stationType))
		deny, err1 := brokerMgr.IsDeny(md.Extension.XOriginFrom, int(resp.BrokerId), st, resp.ExtInfo[stationType])
		if err1 != nil {
			glog.Error(ctx, "GetBrokerIdLoader IsDeny", glog.Int64("uid", resp.UserId), glog.NamedError("err", err1))
			galert.Error(ctx, "GetBrokerIdLoader IsDeny", galert.WithField("err", err1))
			return resp, nil
		}
		if deny {
			glog.Debug(ctx, "GetBrokerIdLoader deny", glog.Int64("resp-BrokerID", int64(resp.BrokerId)), glog.String("xof", md.Extension.XOriginFrom),
				glog.String("stationType", st), glog.String("resp-stationType", resp.ExtInfo[stationType]))
			gmetric.IncDefaultError("auth", "broker_id_deny")
			md.Intermediate.SecureToken = &secureToken
			md.Intermediate.WeakToken = &weakToken
			return nil, berror.ErrAuthVerifyFailed
		}
		return resp, nil
	}
	// check site id
	glog.Debug(ctx, "wl auth verify site id", glog.Int64("resp-BrokerID", int64(resp.BrokerId)), glog.Int64(logKeyBrokerId, int64(md.BrokerID)))
	if resp.BrokerId != md.BrokerID {
		if !rule.allowGuest {
			md.Intermediate.SecureToken = &secureToken
			md.Intermediate.WeakToken = &weakToken
			return nil, berror.ErrAuthVerifyFailed
		}
	}
	return
}

func (a *auth) checkOAuth(ctx *types.Ctx, rule *authRule, md *metadata.Metadata, token string) (*oauthv1.OAuthResponse, error) {
	if token == "" {
		glog.Debug(ctx, "no token, auth not allow guest")
		return nil, berror.ErrAuthVerifyFailed
	}

	oresp, err := a.os.OAuth(service.GetContext(ctx), token)
	if err != nil {
		glog.Error(ctx, "OAuth error", glog.NamedError("err", err))
		return nil, berror.ErrAuthVerifyFailed
	}

	if oresp == nil || oresp.Error == nil || oresp.Error.ErrorCode != 0 || oresp.MemberId == 0 {
		glog.Debug(ctx, "oauth rpc error", glog.Any("oresp", oresp))
		return nil, berror.ErrAuthVerifyFailed
	}
	return oresp, nil
}

func (a *auth) getRule(route metadata.RouteKey) *authRule {
	value, ok := a.rules.Load(route.String())
	if ok {
		return value.(*authRule)
	}
	return nil
}

var suiOnce sync.Once

func limiterFlagParse(ctx context.Context, args []string) (authRule, error) {
	var (
		rule            authRule
		copyTradeOrigin string
		aidQuery        string
	)

	parse := flag.NewFlagSet("auth", flag.ContinueOnError)
	parse.BoolVar(&rule.allowGuest, "allowGuest", false, "allow guest login, default not allow")
	parse.IntVar(&rule.bizType, "biz_type", 1, "aid biz type, old flag") // compatible
	parse.IntVar(&rule.bizType, "bizType", 1, "aid biz type")
	parse.BoolVar(&rule.refreshToken, "refreshToken", false, "refreshToken")
	parse.BoolVar(&rule.skipAID, "skipAID", false, "skip to get account id")
	parse.BoolVar(&rule.copyTrade, "copyTrade", false, "get copyTrade leader id and follower id")
	parse.StringVar(&copyTradeOrigin, "copyTradeInfo", "", "get copyTrade leader id and follower id")
	parse.BoolVar(&rule.unified, "unified", false, "get unified margin account id")
	parse.StringVar(&aidQuery, "accountIDQuery", "", "account id query list")
	parse.BoolVar(&rule.weakAuth, "weakAuth", false, "weakAuth")
	parse.BoolVar(&rule.tradeCheck, "tradeCheck", false, "if enable trade check")
	parse.BoolVar(&rule.unifiedTrading, "unifiedTrading", false, "uta info")
	parse.StringVar(&rule.unifiedTradingCheck, "unifiedTradingCheck", "", "uta check")
	parse.BoolVar(&rule.utaProcessBan, "utaProcessBan", false, "uta process ban")
	parse.BoolVar(&rule.utaStatus, "utaStatus", false, "uta status")
	parse.BoolVar(&rule.suiInfo, "suiInfo", false, "sui info")
	var (
		memberTags            string
		batchTradeCheck       string
		tradeCheckCfgStr      string
		batchTradeCheckCfgStr string
	)
	parse.StringVar(&memberTags, "memberTags", "", "member tags")
	parse.StringVar(&batchTradeCheck, "batchTradeCheck", "", "batch trade check")
	parse.StringVar(&tradeCheckCfgStr, "tradeCheckCfg", "{}", "trade check cfg")
	parse.StringVar(&batchTradeCheckCfgStr, "batchTradeCheckCfg", "{}", "batch trade check cfg")
	parse.BoolVar(&rule.oauth, "oauth", false, "oauth")

	if err := parse.Parse(args[1:]); err != nil {
		glog.Error(ctx, "auth limiterFlagParse error", glog.String("error", err.Error()), glog.Any("args", args))
		return authRule{}, err
	}
	if rule.bizType <= 0 {
		rule.bizType = 1
	}

	if rule.weakAuth && rule.refreshToken {
		return authRule{}, fmt.Errorf("weakAuth and refreshToken can't defind at the same time")
	}

	cti, err := user.CopyTradeInfo{}.Parse(ctx, copyTradeOrigin)
	if err != nil {
		return authRule{}, err
	}
	rule.copyTradeInfo = cti

	if rule.copyTrade || rule.copyTradeInfo != nil {
		if _, err := user.GetCopyTradeService(config.Global.UserServicePrivate); err != nil {
			return authRule{}, err
		}
	}

	if rule.tradeCheck {
		if err := symbolconfig.InitSymbolConfig(); err != nil {
			return authRule{}, err
		}
		rule.tradeCheckCfg = &tradeCheckCfg{}
		if tradeCheckCfgStr != "" {
			if err = json.Unmarshal([]byte(tradeCheckCfgStr), rule.tradeCheckCfg); err != nil {
				return authRule{}, err
			}
		}
	}

	if rule.suiInfo {
		suiOnce.Do(func() {
			user.RegisterMemberTag(suiProtoTag)
			user.RegisterMemberTag(suiKycTag)
		})
	}

	list := strings.Split(aidQuery, ",")
	for _, app := range list {
		if app = strings.TrimSpace(app); app != "" {
			rule.aidQuery = append(rule.aidQuery, app)
		}
	}

	if memberTags != "" {
		tags := strings.Split(memberTags, ",")
		rule.memberTags = tags
		for _, tag := range tags {
			user.RegisterMemberTag(tag)
		}
	}

	if batchTradeCheck != "" {
		products := strings.Split(batchTradeCheck, ",")
		val := make(map[string]struct{})
		for _, p := range products {
			val[p] = struct{}{}
		}
		rule.batchTradeCheck = val

		if batchTradeCheckCfgStr != "" {
			btcc := make(map[string]*tradeCheckCfg)
			if err = json.Unmarshal([]byte(batchTradeCheckCfgStr), &btcc); err != nil {
				return authRule{}, err
			}
			rule.batchTradeCheckCfg = btcc
		}
	}

	return rule, nil
}

func (a *auth) tradeCheck(ctx *types.Ctx, rule *authRule, md *metadata.Metadata) error {
	if !rule.tradeCheck && len(rule.batchTradeCheck) == 0 {
		return nil
	}
	banSvc, err := ban.GetBanService()
	if err != nil {
		return err
	}
	status, err := banSvc.CheckStatus(service.GetContext(ctx), md.UID)
	if err != nil {
		return err
	}
	if rule.tradeCheck {
		return ban.TradeCheckSingleSymbol(ctx, md.Route.GetAppName(ctx), rule.tradeCheckCfg.SymbolField, md.UID, true, status)
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
		banJson, err := ban.TradeCheckBatchSymbol(ctx, sf, app, md.UID, true, status)
		if err != nil {
			return err
		}
		md.BatchBan = banJson
	}
	return nil
}
