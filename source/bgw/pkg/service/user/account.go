package user

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"bgw/pkg/diagnosis"

	"code.bydev.io/frameworks/byone/kafka"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/ggrpc"
	"code.bydev.io/fbu/gateway/gway.git/gkafka"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/frameworks/byone/zrpc"
	"git.bybit.com/svc/stub/pkg/pb/api/consts/euser"
	"git.bybit.com/svc/stub/pkg/pb/api/user"
	"github.com/coocood/freecache"

	"bgw/pkg/common/kafkaconsume"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
	"bgw/pkg/service"
)

const (
	defaultAccountCacheSize   = 256
	accountCacheExpireSeconds = 24 * 3600

	UnifiedMarginTag  = "UNIFIED_ACCOUNT_STATE"
	UnifiedTradingTag = "UTA"
	UserSiteIDTag     = "site-id"

	UnifiedStateSuccess = "SUCCESS"
	UnifiedStateProcess = "PROCESS"
	UnifiedStateFail    = "FAIL"
	UnifiedStateNoOpen  = ""

	MemberTagFailed = "member_tag_err"
)

type UserAPIType = string

var (
	accountService *AccountService
	accountOnce    sync.Once
	memberOnce     sync.Once
)

var (
	errInvalidUserConfig  = errors.New("remote config of user error")
	errInvalidKafkaConfig = errors.New("remote config of kafka error")
)

var (
	// 存在运行时的注册，所以需要加锁保护
	mu sync.RWMutex
	// uma和uta作为default，处理方式特殊
	allTags = []string{UnifiedMarginTag, UnifiedTradingTag}
	// 其他的需要的member tag采用注册的方式
	memberTags = make([]string, 0)
)

func init() {
	RegisterMemberTag(UserSiteIDTag)
}

func RegisterMemberTag(tag string) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return
	}
	glog.Info(context.Background(), "account register member tag", glog.String("value", tag))
	mu.Lock()
	defer mu.Unlock()
	for _, t := range memberTags {
		if t == tag {
			return
		}
	}
	memberTags = append(memberTags, tag)
	allTags = append(allTags, tag)
}

func readAllTags() []string {
	mu.RLock()
	defer mu.RUnlock()
	return allTags
}

func readMemberTags() []string {
	mu.RLock()
	defer mu.RUnlock()
	return memberTags
}

type AccountService struct {
	client       ggrpc.ClientConnInterface
	accountCache *freecache.Cache
}

// AccountIface is the interface of account service
type AccountIface interface {
	GetAccountID(ctx context.Context, uid int64, accountType string, bizType int) (int64, error)
	GetBizAccountIDByApps(ctx context.Context, uid int64, bizType int, apps ...string) (aids []int64, errs []error)

	QueryMemberTag(ctx context.Context, uid int64, tag string) (string, error)
	GetUnifiedMarginAccountID(ctx context.Context, uid int64, bizType int) (aid int64, err error)
	GetUnifiedTradingAccountID(ctx context.Context, uid int64, bizType int) (aid int64, err error)
}

// NewAccountService returns a new account service
func NewAccountService() (AccountIface, error) {
	var err error
	accountOnce.Do(func() {
		rpcClient, err := zrpc.NewClient(config.Global.UserServicePrivate, zrpc.WithDialOptions(service.DefaultDialOptions...))
		if err != nil {
			glog.Errorf(context.Background(), "dial user-service-private fail, error=%v", err)
			galert.Error(context.Background(), "account service dial user-service-private fail", galert.WithField("error", err))
			return
		}

		accountService = &AccountService{
			client: rpcClient.Conn(),
		}

		_ = diagnosis.Register(&usDiagnose{
			cfg:  config.Global.UserServicePrivate,
			kCfg: config.Global.KafkaCli,
			svc:  accountService,
		})

		// account id cache
		size := config.Global.Data.CacheSize.AccountCacheSize
		if size < defaultAccountCacheSize {
			size = defaultAccountCacheSize
		}

		accountService.accountCache = freecache.NewCache(size * 1024 * 1024)
		registerAccountAdmin()
	})
	if accountService == nil {
		galert.Error(context.TODO(), "empty account service")
		gmetric.IncDefaultError("account", "empty_account_service")
		return nil, fmt.Errorf("empty account service: %w", err)
	}

	return accountService, err
}

func (as *AccountService) GetAccountID(ctx context.Context, uid int64, accountType string, bizType int) (int64, error) {
	return as.getBizAccountID(ctx, uid, accountType, bizType)
}

func (as *AccountService) GetBizAccountIDByApps(ctx context.Context, uid int64, bizType int, apps ...string) ([]int64, []error) {
	rets := make([]int64, len(apps))
	errs := make([]error, len(apps))
	for i, app := range apps {
		rets[i], errs[i] = as.getBizAccountID(ctx, uid, app, bizType)
	}
	return rets, errs
}

func (as *AccountService) getBizAccountID(ctx context.Context, uid int64, accountType string, bizType int) (int64, error) {
	// skip future aid, not unified
	if accountType == constant.AppTypeFUTURES {
		return uid, nil
	}
	appIndex, ok := constant.AppType[accountType]
	if !ok {
		glog.Debug(ctx, "app not found", glog.Any("app", accountType))
		// CHT,arch... don't need aid
		return 0, nil
	}
	key := fmt.Sprintf("%d%s%d", uid, accountType, bizType)
	aid, err := as.accountCache.Get([]byte(key))
	if err == nil {
		glog.Debug(ctx, "account cache hit", glog.String("key", key))
		return cast.BytesToInt64(aid), nil
	}
	glog.Debug(ctx, "account cache not hit", glog.String("key", key))

	accountID, err := as.getAccountID(ctx, uid, appIndex, int32(bizType))
	if err != nil {
		return 0, err
	}

	// set cache
	if err = as.accountCache.Set([]byte(key), cast.Int64ToBytes(accountID), 0); err != nil {
		glog.Error(ctx, "set account cache error", glog.String("key", key), glog.String("error", err.Error()))
	}

	return accountID, nil
}

// getAccountID get account id from remote service
func (as *AccountService) getAccountID(ctx context.Context, uid int64, appIndex, bizType int32) (int64, error) {
	if bizType <= 0 {
		glog.Info(ctx, "bizType error", glog.Int64("bizType", int64(bizType)))
		return 0, berror.NewInterErr("get accountID bizType is 0")
	}

	accountType := euser.AccountType(appIndex)

	req := &user.GetAccountByMemberRequest{
		MemberId: uid,
		Selectors: []*user.AccountSelector{
			{
				AccountType: accountType,
				BizType:     bizType,
			},
		},
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := user.NewAccountInternalClient(as.client).GetAccountIDSByMemberID(ctx, req)
	if err != nil {
		return 0, berror.NewUpStreamErr(berror.UpstreamErrUserServiceInvokeFailed, "GetAccountIDSByMemberID error", err.Error(), req.String())
	}

	if resp.Error != nil {
		return 0, berror.NewUpStreamErr(berror.UpstreamErrUserServiceInvokeFailed, "GetAccountIDSByMemberID resp error", resp.Error.String(), req.String())
	}

	if len(resp.Accounts) == 0 {
		return 0, berror.NewUpStreamErr(berror.UpstreamErrUserServiceInvokeFailed, "GetAccountIDSByMemberID response accounts is nil", req.String())
	}

	if resp.Accounts[0].AccountId <= 0 {
		return 0, berror.NewUpStreamErr(berror.UpstreamErrUserServiceInvokeFailed, "GetAccountIDSByMemberID response account_id invalid", req.String())
	}

	return resp.Accounts[0].AccountId, nil
}

func (as *AccountService) QueryMemberTag(ctx context.Context, uid int64, tag string) (string, error) {
	if uid <= 0 {
		return "", nil
	}
	memberOnce.Do(func() {
		kafkaconsume.AsyncHandleKafkaMessage(ctx, constant.EventMemberTagChange, config.Global.KafkaCli,
			as.HandleMemberTagMessage, accountOnErr)
	})

	key := fmt.Sprintf("%dmember_tag_%s", uid, tag)
	val, err := as.accountCache.Get([]byte(key))
	if err == nil {
		glog.Debug(ctx, "member tag cache hit", glog.String("key", key))
		return string(val), nil
	}

	glog.Debug(ctx, "member tag cache not hit", glog.String("key", key))

	at := readAllTags()
	req := &user.QueryMemberTagRequest{
		MemberId: uid,
		TagKeys:  at,
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := user.NewMemberInternalClient(as.client).QueryMemberTag(ctx, req)
	if err != nil {
		return "", berror.NewUpStreamErr(berror.UpstreamErrUserServiceInvokeFailed, "QueryMemberTag error", err.Error(), req.String())
	}

	if resp.Error != nil {
		return "", berror.NewUpStreamErr(berror.UpstreamErrUserServiceInvokeFailed, "QueryMemberTag resp error", resp.Error.String(), req.String())
	}

	for _, t := range at {
		k := fmt.Sprintf("%dmember_tag_%s", uid, t)
		v := resp.Result.TagInfo[t]
		if t == UnifiedMarginTag && resp.Result.TagInfo[UnifiedTradingTag] == UnifiedStateSuccess {
			v = UnifiedStateSuccess
		}
		random := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(14400)
		if err = as.accountCache.Set([]byte(k), []byte(v), accountCacheExpireSeconds+random); err != nil {
			glog.Error(ctx, "set unified account invalid cache error", glog.String("key", k), glog.String("unified", v), glog.String("error", err.Error()))
		}
	}

	status := resp.Result.TagInfo[tag]
	if tag == UnifiedMarginTag && resp.Result.TagInfo[UnifiedTradingTag] == UnifiedStateSuccess {
		status = UnifiedStateSuccess
	}
	return status, nil
}

// MemberMessage member unified message
type MemberMessage struct {
	MemberID       int64                  `json:"member_id"`
	MemberIDs      []int64                `json:"sub_member_ids"`
	UpsertTagInfo  map[string]string      `json:"upsert_tag_info"`
	ExtInfo        map[string]interface{} `json:"ext_info"`
	RemoveTagsKeys []string               `json:"remove_tag_keys"`
}

func (as *AccountService) HandleMemberTagMessage(ctx context.Context, msg *gkafka.Message) {
	var mMsg MemberMessage
	if err := util.JsonUnmarshal(msg.Value, &mMsg); err != nil {
		glog.Error(ctx, "HandleMemberTagMessage Unmarshal error", glog.String("error", err.Error()))
		return
	}
	glog.Info(ctx, "member unified msg", glog.Int64("uid", mMsg.MemberID), glog.Any("uids", mMsg.MemberIDs),
		glog.Any("UpsertTagInfo", mMsg.UpsertTagInfo), glog.Int64("offset", msg.Offset))

	if st, ok := mMsg.UpsertTagInfo[copytradeUpgrade]; ok && st == copytradeUpgradeSuccess {
		ct, err := GetCopyTradeService(config.Global.UserServicePrivate)
		if err != nil {
			glog.Error(ctx, "member tag GetCopyTradeService error", glog.NamedError("err", err))
			galert.Error(ctx, "member tag GetCopyTradeService error", galert.WithField("err", err))
			return
		}
		ct.DeleteCopyTradeData(mMsg.MemberID)
		if v, ok := mMsg.ExtInfo[ownerMemberId]; ok {
			ct.DeleteCopyTradeData(cast.ToInt64(v))
		}
	}

	memberIDs := append(mMsg.MemberIDs, mMsg.MemberID)

	for _, memberID := range memberIDs {
		// uma和uta稍微特殊一点
		utaStatus, ok := mMsg.UpsertTagInfo[UnifiedTradingTag]
		if ok {
			// delete ua aid
			key := fmt.Sprintf("%dunified_trading1", memberID)
			glog.Debug(ctx, "unified trading id will del", glog.String("key", key))
			as.accountCache.Del([]byte(key))

			// delete ua tag
			key = fmt.Sprintf("%dmember_tag_%s", memberID, UnifiedTradingTag)
			glog.Debug(ctx, "unified trading tag will del", glog.String("key", key))
			as.accountCache.Del([]byte(key))
		}

		umaStatus, ok := mMsg.UpsertTagInfo[UnifiedMarginTag]
		if (ok && umaStatus == UnifiedStateSuccess) || utaStatus == UnifiedStateSuccess {
			// delete um aid
			key := fmt.Sprintf("%dunified_margin1", memberID)
			glog.Debug(ctx, "unified margin id will del", glog.String("key", key))
			as.accountCache.Del([]byte(key))

			// delete um tag
			key = fmt.Sprintf("%dmember_tag_%s", memberID, UnifiedMarginTag)
			glog.Debug(ctx, "unified margin tag will del", glog.String("key", key))
			as.accountCache.Del([]byte(key))
		}

		mt := readMemberTags()
		for _, tag := range mt {
			_, ok = mMsg.UpsertTagInfo[tag]
			if ok {
				key := fmt.Sprintf("%dmember_tag_%s", memberID, tag)
				glog.Debug(ctx, "member tag will del", glog.String("key", key))
				as.accountCache.Del([]byte(key))
			}
		}
	}
}

func accountOnErr(err *gkafka.ConsumerError) {
	if err != nil {
		galert.Error(context.Background(), "account consumer err:"+err.Error())
	}
}

type usDiagnose struct {
	cfg  zrpc.RpcClientConf
	kCfg kafka.UniversalClientConfig
	svc  *AccountService
}

func (o *usDiagnose) Key() string {
	return "user-service-private"
}

func (o *usDiagnose) Diagnose(ctx context.Context) (interface{}, error) {
	resp := make(map[string]interface{})
	resp["kafka"] = diagnosis.DiagnoseKafka(ctx, constant.EventMemberTagChange, o.kCfg)
	resp["grpc"] = diagnosis.DiagnoseGrpcDependency(ctx, o.cfg)
	return resp, nil
}
