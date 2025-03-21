package ban

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"bgw/pkg/diagnosis"

	"code.bydev.io/frameworks/byone/kafka"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/kafkaconsume"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
	"bgw/pkg/service"
	"bgw/pkg/service/symbolconfig"

	futenumsv1 "code.bydev.io/fbu/future/bufgen.git/pkg/bybit/future/futenums/v1"
	"code.bydev.io/fbu/future/sdk.git/pkg/future"
	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/ggrpc"
	"code.bydev.io/fbu/gateway/gway.git/gkafka"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/frameworks/byone/zrpc"
	"git.bybit.com/svc/stub/pkg/pb/api/ban"
	"github.com/coocood/freecache"
	jsoniter "github.com/json-iterator/go"
	"google.golang.org/protobuf/proto"
)

const (
	defaultBanCacheSize   = 128
	banCacheExpireSeconds = 120 * 3600
	bannedFrom            = "25e56d33ac3f5b543d7d9ea4ec513242" // apply for from banned platform
)

type UserStatus = ban.UserStatus
type UserStatusWrap struct {
	LoginStatus    int32
	WithdrawStatus int32
	TradeStatus    int32
	LoginBanType   int32
	UserState      *UserStatus
}

type userStatusInternal struct {
	LoginStatus    int32  `json:"login_status"`
	WithdrawStatus int32  `json:"withdraw_status"`
	TradeStatus    int32  `json:"trade_status"`
	LoginBanType   int32  `json:"login_ban_type"`
	UserState      []byte `json:"user_state"`
}

type BanServiceIface interface {
	GetMemberStatus(ctx context.Context, uid int64) (*UserStatusWrap, error)
	CheckStatus(ctx context.Context, uid int64) (*UserStatusWrap, error)
	VerifyTrade(ctx context.Context, uid int64, app string, status *UserStatusWrap, opts ...Option) (bool, error)
}

var (
	defaultBanService BanServiceIface
	banOnce           sync.Once
)

type banService struct {
	client ggrpc.ClientConnInterface
	cache  *freecache.Cache
}

type Config struct {
	RpcConf   zrpc.RpcClientConf
	KafkaConf kafka.UniversalClientConfig
	CacheSize int
}

func Init(conf Config) error {
	if conf.CacheSize < defaultBanCacheSize {
		conf.CacheSize = defaultBanCacheSize
	}

	if conf.RpcConf.Nacos.Key == "" {
		conf.RpcConf.Nacos.Key = "ban_service_private.rpc"
	}

	rpcClient, err := zrpc.NewClient(conf.RpcConf, zrpc.WithDialOptions(service.DefaultDialOptions...))
	if err != nil {
		glog.Errorf(context.Background(), "dial ban service fail, error=%v", err)
		galert.Error(context.Background(), "ban service dial fail", galert.WithField("error", err))
		return err
	}

	svc := &banService{
		client: rpcClient.Conn(),
		cache:  freecache.NewCache(conf.CacheSize * 1024 * 1024),
	}

	svc.kafkaConsume(conf.KafkaConf)
	defaultBanService = svc
	_ = diagnosis.Register(&diagnose{
		cfg:  config.Global.BanServicePrivate,
		kCfg: config.Global.KafkaCli,
		svc:  defaultBanService.(*banService),
	})
	registerAdmin()
	return nil
}

func SetBanService(s BanServiceIface) {
	defaultBanService = s
}

// GetBanService creates a new BanService
func GetBanService() (BanServiceIface, error) {
	var err error
	if defaultBanService == nil {
		banOnce.Do(func() {
			conf := Config{
				RpcConf:   config.Global.BanServicePrivate,
				KafkaConf: config.Global.KafkaCli,
				CacheSize: config.Global.Data.CacheSize.BanCacheSize,
			}
			err = Init(conf)
		})
	}
	if defaultBanService == nil {
		gmetric.IncDefaultError("ban", "empty_ban_service")
		return nil, fmt.Errorf("empty ban service: %w", err)
	}
	return defaultBanService, nil
}

func (b *banService) kafkaConsume(kconf kafka.UniversalClientConfig) {
	ctx := context.Background()
	kafkaconsume.AsyncHandleKafkaMessage(ctx, constant.EventMemberBanned, kconf, b.handleChtMemberBannedMessage, b.banOnErr)
}

// GetMemberStatus get member status by member id
func (b *banService) GetMemberStatus(ctx context.Context, uid int64) (*UserStatusWrap, error) {
	key := fmt.Sprintf("%d_banned_info", uid)
	byteUserStatus, err := b.cache.Get([]byte(key))
	if err == nil {
		glog.Debug(ctx, "openapi member banned cache hit", glog.String("key", key))
		cachedMsg := &userStatusInternal{}
		if err := jsoniter.Unmarshal(byteUserStatus, cachedMsg); cachedMsg.UserState == nil || err != nil {
			return nil, err
		}
		us := &UserStatus{}
		err = proto.Unmarshal(cachedMsg.UserState, us)
		if err != nil {
			return nil, err
		}
		glog.Debug(ctx, "openapi member banned cache hit, banned status", glog.String("key", key),
			glog.Any("status", us), glog.Any("loginBanType", cachedMsg.LoginBanType))
		return &UserStatusWrap{
			LoginStatus:    cachedMsg.LoginStatus,
			WithdrawStatus: cachedMsg.WithdrawStatus,
			TradeStatus:    cachedMsg.TradeStatus,
			LoginBanType:   cachedMsg.LoginBanType,
			UserState:      us,
		}, nil
	}
	glog.Debug(ctx, "openapi member banned cache not hit", glog.String("key", key))

	req := &ban.BatchQueryBanStatusRequest{
		Uids: []int64{uid},
		From: bannedFrom,
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := ban.NewBanInternalClient(b.client).BatchQueryBanStatus(ctx, req)
	if err != nil || len(resp.UserStatusItems) == 0 {
		return nil, berror.NewUpStreamErr(berror.UpstreamErrBanServiceInvokeFailed, fmt.Sprintf("BatchQueryBanStatus error: %v, %s", err, req.String()))
	}
	var userBanStatus *UserStatus
	for _, status := range resp.UserStatusItems {
		if status.Uid == uid {
			userBanStatus = status
			break
		}
	}

	parsedBanState, parsedBanType := b.parseBanStatus(ctx, userBanStatus, uid)
	result := &UserStatusWrap{
		LoginStatus:    parsedBanState.loginBanStatus,
		WithdrawStatus: parsedBanState.withdrawStatus,
		TradeStatus:    parsedBanState.tradeBanStatus,
		LoginBanType:   int32(parsedBanType),
		UserState:      userBanStatus,
	}
	pbUserStatus, err := proto.Marshal(userBanStatus)
	if err != nil {
		glog.Error(ctx, "openapi member not hit, banned status Marshal error", glog.String("key", key), glog.Any("status", resp.UserStatusItems[0]))
		result.UserState = resp.UserStatusItems[0]
		return result, nil
	}

	needCacheMsg := &userStatusInternal{
		LoginStatus:    result.LoginStatus,
		WithdrawStatus: result.WithdrawStatus,
		TradeStatus:    result.TradeStatus,
		LoginBanType:   result.LoginBanType,
		UserState:      pbUserStatus,
	}
	data, err := jsoniter.Marshal(needCacheMsg)
	if err != nil {
		glog.Error(ctx, "openapi member not hit, banned status Marshal error", glog.String("key", key), glog.Any("status", resp.UserStatusItems[0]))
		result.UserState = resp.UserStatusItems[0]
		return result, nil
	}

	random := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(14400)
	_ = b.cache.Set([]byte(key), data, banCacheExpireSeconds+random)
	glog.Debug(ctx, "openapi member not hit, banned status", glog.String("key", key), glog.Any("status", resp.UserStatusItems[0]))

	return result, nil
}

// CheckStatus check ban login status
func (b *banService) CheckStatus(ctx context.Context, uid int64) (userState *UserStatusWrap, err error) {
	userState, err = b.GetMemberStatus(ctx, uid)
	if err != nil {
		return nil, err
	}

	glog.Debug(ctx, "CheckStatus", glog.Any("CheckStatus.userState", userState))

	// login check
	if userState.LoginBanType == int32(BantypeLogin) {
		return nil, berror.ErrOpenAPIUserLoginBanned
	}

	return userState, nil
}

type Options struct {
	app     string
	uid     int64
	siteAPI bool
	symbol  string
}

type Option func(*Options)

// WithSymbol with symbol
func WithSymbol(symbol string) Option {
	return func(o *Options) {
		o.symbol = symbol
	}
}

// WithSiteAPI with site api
func WithSiteAPI(isSiteAPI bool) Option {
	return func(o *Options) {
		o.siteAPI = isSiteAPI
	}
}

// VerifyTrade verify trade ban
func (b *banService) VerifyTrade(ctx context.Context, uid int64, app string, status *UserStatusWrap, opts ...Option) (bool, error) {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	glog.Debug(ctx, "do tradeCheck", glog.String("app", app),
		glog.Int64("uid", uid), glog.Any("userStatus", status.UserState), glog.Any("options", o))
	switch app {
	case constant.AppTypeFUTURES:
		bannedReduceOnly, err := b.futuresTradeCheck(ctx, status.UserState, uid, o.symbol, o.siteAPI)
		if err != nil {
			return false, err
		}
		return bannedReduceOnly, nil
	case constant.AppTypeSPOT:
		if err := b.spotTradeCheck(ctx, status.UserState); err != nil {
			return false, err
		}
	case constant.AppTypeOPTION:
		bannedReduceOnly, err := b.optionTradeCheck(ctx, status.UserState, o)
		if err != nil {
			return false, err
		}
		return bannedReduceOnly, nil
	}
	return false, nil
}

func (b *banService) futuresTradeCheck(ctx context.Context, userStatus *UserStatus, uid int64, symbol string, siteAPI bool) (bannedReduceOnly bool, err error) {
	glog.Debug(ctx, "futures trade check", glog.Int64("uid", uid), glog.String("symbol", symbol), glog.Any("userStatus", userStatus))
	banList := make([]BanType, 0, 1)
	banItemMap := make(map[BanType]*ban.UserStatus_BanItem)

	for _, banItem := range userStatus.GetBanItems() {
		biz := banItem.GetBizType()
		tag := banItem.GetTagName()
		if (biz == FbuType || biz == TradeType || biz == UTAType || biz == DBUType || biz == DERIVATIVESType) && tag == TradeTag {
			sc, err := symbolconfig.GetSymbolModule()
			if err != nil {
				return false, nil
			}
			value := banItem.GetTagValue()
			banType, ok := GetBanTypeByString(value)
			glog.Debug(ctx, "futures trade check", glog.Any("banType", banType), glog.Any("ok", ok))
			if symbol == "" {
				return false, berror.ErrInvalidRequest
			}
			if ok {
				// because usdc and usdt have same contract type, so we need to check usdc ban type directly
				if IsUsdcBanType(banType) && !symbolconfig.IsLinearUsdcSymbol(symbol, sc) {
					continue
				} else if !IsUsdcBanType(banType) && symbolconfig.IsLinearUsdcSymbol(symbol, sc) {
					continue
				}

				banItemMap[banType] = banItem
				t, exist := GetContractTypeByBanType(banType)
				glog.Debug(ctx, "futures trade check", glog.Any("banType", banType), glog.Any("ContractType", t))

				// not product ban
				if !exist {
					banList = append(banList, banType)
					continue
				}

				// product ban
				if exist && symbolconfig.GetContractType(symbol, sc) == t {
					banList = append(banList, banType)
					continue
				}
			}
			// 剩下的是symbol维度的封禁
			if !strings.Contains(value, SymbolBanPrefix) {
				continue
			}
			// symbol_BTCUSDT_lu: BTCUSDT只减仓封禁
			// symbol_BTCUSDT: BTCUSDT 交易封禁
			res := strings.Split(value, Delimiter)
			if len(res) >= 2 {
				sym := symbolconfig.GetSymbolEnum(res[1], sc)
				if sym != future.Symbol(0) && sym == future.Symbol(symbolconfig.HandleSymbol(symbol, sc)) {
					if len(res) == 3 && res[2] == LightUpSuffix {
						banList = append(banList, BantypeLightenUp)
					} else {
						banList = append(banList, BantypeAllTrade)
						if _, exist := banItemMap[BantypeAllTrade]; !exist {
							banItemMap[BantypeAllTrade] = banItem
						}
					}
				}
			}
		}
	}
	// 原则是取第一个匹配上的封禁项
	for _, banInfo := range banList {
		switch banInfo {
		case BantypeAllTrade, BantypeAPI, BanTypeUTAUpgrade:
			glog.Debug(ctx, "futures user banned", glog.Int64("uid", uid), glog.Any("banned list", banList))
			if banInfo == BanTypeUTAUpgrade && siteAPI {
				return bannedReduceOnly, berror.ErrTradeCheckUTAProcessBanned
			}
			return bannedReduceOnly, getErrFromBanItemMap(banInfo, berror.ErrOpenAPIUserAllBanned, banItemMap)
		case BantypeUsdtAllKo, BantypeUsdtAll, BanTypeUsdtFutureAllKo, BanTypeUsdtFutureAll,
			BantypeInversePerpetualAll, BantypeInversePerpetualeAllKo,
			BantypeInverseFutureAll, BantypeInverseFutureAllKo, BantypeUsdcPerpetualAllKo, BantypeUsdcFutureAllKo, BantypeUsdcFutureAll, BantypeUsdcAll:
			glog.Debug(ctx, "futures user banned", glog.Int64("uid", uid), glog.Any("banned list", banList))
			return bannedReduceOnly, getErrFromBanItemMap(banInfo, berror.ErrOpenAPIUserUsdtAllBanned, banItemMap)
		case BantypeLightenUp, BantypeUsdtLu, BantypeUsdtLuKo,
			BanTypeUsdtFutureLuKo, BanTypeUsdtFutureLu,
			BantypeInversePerpetualLu, BantypeInversePerpetualLuKo,
			BantypeInverseFutureLu, BantypeInverseFutureLuKo, BantypeUsdcFutureLu, BantypeUsdcLu:
			glog.Debug(ctx, "futures user banned reduce only", glog.Int64("uid", uid))
			bannedReduceOnly = true
		}
	}
	return bannedReduceOnly, nil
}

func (b *banService) spotTradeCheck(ctx context.Context, userStatus *UserStatus) error {
	items := userStatus.GetBanItems()
	for _, item := range items {
		err := tradeCheckSpot(ctx, item)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *banService) optionTradeCheck(ctx context.Context, userStatus *UserStatus, opts *Options) (bannedReduceOnly bool, err error) {
	items := userStatus.GetBanItems()
	for _, item := range items {
		ro, err := tradeCheckOptions(ctx, item, opts)
		if err != nil {
			return false, err
		}
		if ro {
			bannedReduceOnly = ro
		}
	}
	return bannedReduceOnly, nil
}

// parseBanStatus  https://confluence.yijin.io/pages/viewpage.action?pageId=31911660
func (b *banService) parseBanStatus(ctx context.Context, status *UserStatus, memberID int64) (banStatus, BanType) {
	loginBanType := BantypeUnspecified
	bs := banStatus{
		loginBanStatus: int32(MemberLoginStatus_LOGIN_NORMAL),
		tradeBanStatus: int32(MemberTradeStatus_TRADE_NORMAL),
		withdrawStatus: int32(MemberWithdrawStatus_WITHDRAW_NORMAL),
	}
	for _, banItem := range status.GetBanItems() {
		if banItem.GetBizType() == LoginType && banItem.GetTagName() == LoginTag {
			value := banItem.GetTagValue()
			banType, ok := banTypeValue[value]
			if !ok {
				glog.Warn(ctx, "unknown login ban type", glog.Int64("member id", memberID), glog.String("tag", value))
				bs.loginBanStatus = int32(MemberLoginStatus_LOGIN_UNKNOWN)
				continue
			}
			loginBanType = banType
			bs.loginBanStatus = int32(MemberLoginStatus_LOGIN_BAN)
			continue
		}
		if banItem.GetBizType() == WithdrawType && banItem.GetTagName() == WithdrawTag {
			value := banItem.GetTagValue()
			_, ok := banTypeValue[value]
			if !ok {
				glog.Warn(ctx, "unknown withdraw ban type", glog.Int64("member id", memberID), glog.String("tag", value))
				bs.withdrawStatus = int32(MemberWithdrawStatus_WITHDRAW_UNKNOWN)
				continue
			}
			bs.withdrawStatus = int32(MemberWithdrawStatus_WITHDRAW_BAN)
			continue
		}
		if banItem.GetBizType() == TradeType && banItem.GetTagName() == TradeTag {
			value := banItem.GetTagValue()
			_, ok := banTypeValue[value]
			if !ok {
				glog.Warn(ctx, "unknown trade ban type", glog.Int64("member id", memberID), glog.String("tag", value))
				bs.tradeBanStatus = int32(MemberTradeStatus_TRADE_UNKNOWN)
				continue
			}
			bs.tradeBanStatus = int32(MemberTradeStatus_TRADE_BAN)
			continue
		}
	}
	glog.Debug(ctx, "loginBanStatus", glog.Int64("member id", memberID), glog.Int64("banType", int64(loginBanType)))
	return bs, loginBanType
}

type banStatus struct {
	loginBanStatus int32
	tradeBanStatus int32
	withdrawStatus int32
}

const (
	// ban biz type
	FbuType         = "FBU"
	LoginType       = "LOGIN"
	TradeType       = "TRADE"
	WithdrawType    = "WITHDRAW"
	UTAType         = "UTA"
	DBUType         = "DBU"
	SPOTType        = "SPOT"
	DERIVATIVESType = "DERIVATIVES"

	// ban tag name
	LoginTag    = "account"
	TradeTag    = "trade"
	WithdrawTag = "withdraw"

	KeepOrderSuffix = "ko"
	LightUpSuffix   = "lu"
	SymbolBanPrefix = "symbol"

	Delimiter = "_"
)

type BanType int32

const (
	BantypeLogin BanType = iota
	BantypeAllTrade
	BantypeLightenUp
	BantypeAPI
	BantypeUsdtAllKo
	BantypeUsdtAll
	BantypeUsdtLu
	BantypeUsdtLuKo

	BanTypeUsdtFutureAllKo
	BanTypeUsdtFutureAll
	BanTypeUsdtFutureLuKo
	BanTypeUsdtFutureLu
	BantypeInversePerpetualAll
	BantypeInversePerpetualeAllKo
	BantypeInversePerpetualLu
	BantypeInversePerpetualLuKo
	BantypeInverseFutureAll
	BantypeInverseFutureAllKo
	BantypeInverseFutureLu
	BantypeInverseFutureLuKo

	BantypeWithdrawAll
	BantypeWithdrawNonPrincipal

	BanTypeUTAUpgrade

	BantypeSpotAllKo
	BantypeOptionsAllKo

	BantypeUnspecified

	BantypeUsdcPerpetualAllKo

	BantypeUsdcAll
	BantypeUsdcLu

	BantypeUsdcFutureAllKo

	BantypeUsdcFutureAll
	BantypeUsdcFutureLu
)

const (
	Login                 = "login"
	AllTrade              = "all_trade"
	LightenUp             = "lighten_up"
	OpenAPI               = "open_api"
	USDTAllKO             = "usdt_all_ko"
	USDTAll               = "usdt_all"
	USDTLU                = "usdt_lu"
	USDTLUKO              = "usdt_lu_ko"
	USDTFutureAllKO       = "usdt_future_all_ko"
	USDTFutureAll         = "usdt_future_all"
	USDTFutureLUKO        = "usdt_future_lu_ko"
	USDTFutureLU          = "usdt_future_lu" // todo 没用？
	InversePerpetualAll   = "inverse_perpetual_all"
	InversePerpetualAllKO = "inverse_perpetual_all_ko"
	InversePerpetualLU    = "inverse_perpetual_lu"
	InversePerpetualLUKO  = "inverse_perpetual_lu_ko"
	InverseFutureAll      = "inverse_future_all"
	InverseFutureAllKO    = "inverse_future_all_ko"
	InverseFutureLU       = "inverse_future_lu"
	InverseFutureLUKO     = "inverse_future_lu_ko"
	AllWithdraw           = "all_withdraw"
	NonPrincipal          = "non-principal"
	Upgrade               = "upgrade"

	USDCPERPETUALALLKO = "usdc_perpetual_all_ko"
	USDCAll            = "usdc_perpetual_all"
	USDCLU             = "usdc_perpetual_lu"

	USDCFUTUREALLKO = "usdc_future_all_ko"
	USDCFutureAll   = "usdc_future_all"
	USDCFutureLU    = "usdc_future_lu"
)

var banTypeValue = map[string]BanType{
	Login:     BantypeLogin,
	AllTrade:  BantypeAllTrade,
	LightenUp: BantypeLightenUp,
	OpenAPI:   BantypeAPI,
	USDTAllKO: BantypeUsdtAllKo,
	USDTAll:   BantypeUsdtAll,
	USDTLU:    BantypeUsdtLu,
	USDTLUKO:  BantypeUsdtLuKo,

	USDTFutureAllKO: BanTypeUsdtFutureAllKo,
	USDTFutureAll:   BanTypeUsdtFutureAll,
	USDTFutureLUKO:  BanTypeUsdtFutureLuKo,
	USDTFutureLU:    BanTypeUsdtFutureLu,

	USDCPERPETUALALLKO: BantypeUsdcPerpetualAllKo,
	USDCAll:            BantypeUsdcAll,
	USDCLU:             BantypeUsdcLu,

	USDCFUTUREALLKO: BantypeUsdcFutureAllKo,
	USDCFutureAll:   BantypeUsdcFutureAll,
	USDCFutureLU:    BantypeUsdcFutureLu,

	InversePerpetualAll:   BantypeInversePerpetualAll,
	InversePerpetualAllKO: BantypeInversePerpetualeAllKo,
	InversePerpetualLU:    BantypeInversePerpetualLu,
	InversePerpetualLUKO:  BantypeInversePerpetualLuKo,
	InverseFutureAll:      BantypeInverseFutureAll,
	InverseFutureAllKO:    BantypeInverseFutureAllKo,
	InverseFutureLU:       BantypeInverseFutureLu,
	InverseFutureLUKO:     BantypeInverseFutureLuKo,
	AllWithdraw:           BantypeWithdrawAll,
	NonPrincipal:          BantypeWithdrawNonPrincipal,
	Upgrade:               BanTypeUTAUpgrade,
	SPOTAllKO:             BantypeSpotAllKo,
	OptionsAllKO:          BantypeOptionsAllKo,
}

type LoginBanStatus int32

const (
	MemberLoginStatus_LOGIN_UNKNOWN LoginBanStatus = iota
	MemberLoginStatus_LOGIN_NORMAL
	MemberLoginStatus_LOGIN_BAN
)

type WithdrawBanStatus int32

const (
	MemberWithdrawStatus_WITHDRAW_UNKNOWN WithdrawBanStatus = iota
	MemberWithdrawStatus_WITHDRAW_NORMAL
	MemberWithdrawStatus_WITHDRAW_BAN
)

type TradeBanStatus int32

const (
	MemberTradeStatus_TRADE_UNKNOWN TradeBanStatus = iota
	MemberTradeStatus_TRADE_NORMAL
	MemberTradeStatus_TRADE_BAN
)

func GetBanTypeByString(value string) (BanType, bool) {
	if v, ok := banTypeValue[value]; ok {
		return v, true
	}
	return BantypeUnspecified, false
}

var banValueToContractType = map[BanType]futenumsv1.ContractType{
	BantypeUsdtAllKo: futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_PERPETUAL,
	BantypeUsdtAll:   futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_PERPETUAL,
	BantypeUsdtLu:    futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_PERPETUAL,
	BantypeUsdtLuKo:  futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_PERPETUAL,

	BantypeUsdcPerpetualAllKo: futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_PERPETUAL,
	BantypeUsdcAll:            futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_PERPETUAL,
	BantypeUsdcLu:             futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_PERPETUAL,

	BantypeUsdcFutureAllKo: futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_FUTURES,
	BantypeUsdcFutureAll:   futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_FUTURES,
	BantypeUsdcFutureLu:    futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_FUTURES,

	BanTypeUsdtFutureAllKo:        futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_FUTURES,
	BanTypeUsdtFutureAll:          futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_FUTURES,
	BanTypeUsdtFutureLuKo:         futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_FUTURES,
	BanTypeUsdtFutureLu:           futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_FUTURES,
	BantypeInversePerpetualAll:    futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_PERPETUAL,
	BantypeInversePerpetualeAllKo: futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_PERPETUAL,
	BantypeInversePerpetualLu:     futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_PERPETUAL,
	BantypeInversePerpetualLuKo:   futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_PERPETUAL,
	BantypeInverseFutureAll:       futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_FUTURES,
	BantypeInverseFutureAllKo:     futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_FUTURES,
	BantypeInverseFutureLu:        futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_FUTURES,
	BantypeInverseFutureLuKo:      futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_FUTURES,
	BanTypeUTAUpgrade:             futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_PERPETUAL,
}

func GetContractTypeByBanType(t BanType) (futenumsv1.ContractType, bool) {
	if v, ok := banValueToContractType[t]; ok {
		return v, true
	}
	return futenumsv1.ContractType_CONTRACT_TYPE_UNSPECIFIED, false
}

type ChtBannedMessage struct {
	Uids      []int  `json:"Uids"`
	From      string `json:"From"`
	Operator  string `json:"Operator"`
	Status    int    `json:"Status"`
	RequestID string `json:"RequestID"`
	BanTag    []struct {
		BizType string `json:"biz_type"`
		Name    string `json:"name"`
		Value   string `json:"value"`
	} `json:"BanTag"`
	Reason     string `json:"Reason"`
	Comment    string `json:"Comment"`
	BrokerID   string `json:"BrokerID"`
	ExpireTime int    `json:"ExpireTime"`
}

func (b *banService) handleChtMemberBannedMessage(ctx context.Context, msg *gkafka.Message) {
	var bannedMsg ChtBannedMessage
	if err := util.JsonUnmarshal(msg.Value, &bannedMsg); err != nil {
		glog.Error(ctx, "HandleChtMemberBannedMessage Unmarshal error", glog.String("error", err.Error()))
		return
	}
	glog.Info(ctx, "cht member banned msg", glog.Any("uid", bannedMsg.Uids),
		glog.Any("detail", bannedMsg), glog.Int64("offset", msg.Offset))

	for _, id := range bannedMsg.Uids {
		key := fmt.Sprintf("%d_banned_info", id)
		glog.Debug(ctx, "user status del banned_info", glog.String("key", key))
		b.cache.Del([]byte(key))
	}
}

func (b *banService) banOnErr(err *gkafka.ConsumerError) {
	if err != nil {
		galert.Error(context.Background(), "member ban consumer err "+err.Error())
	}
}

type diagnose struct {
	cfg  zrpc.RpcClientConf
	kCfg kafka.UniversalClientConfig
	svc  *banService
}

func (o *diagnose) Key() string {
	return "ban_service_private"
}

func (o *diagnose) Diagnose(ctx context.Context) (interface{}, error) {
	resp := make(map[string]interface{})
	resp["kafka"] = diagnosis.DiagnoseKafka(ctx, constant.EventMemberBanned, o.kCfg)
	resp["grpc"] = diagnosis.DiagnoseGrpcDependency(ctx, o.cfg)
	return resp, nil
}
