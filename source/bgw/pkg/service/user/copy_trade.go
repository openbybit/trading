package user

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"bgw/pkg/diagnosis"

	"code.bydev.io/frameworks/byone/kafka"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/ggrpc"
	"code.bydev.io/fbu/gateway/gway.git/gkafka"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/frameworks/byone/zrpc"
	"git.bybit.com/svc/stub/pkg/pb/api/consts/euser"
	"git.bybit.com/svc/stub/pkg/pb/api/user"
	"github.com/coocood/freecache"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/kafkaconsume"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
	"bgw/pkg/service"
)

const (
	defaultCopyTradeCacheSize   = 96
	copyTradeCacheExpireSeconds = 120 * 3600
	success                     = "SUCCESS"
)

var (
	copyTradeService     *CopyTradeService
	copyTradeOnce        sync.Once
	copyTradeConsumeOnce sync.Once
)

type CopyTrade struct {
	LeaderID        int64   `json:"leader_id,omitempty"`
	LeaderIDs       []int64 `json:"leader_ids,omitempty"`
	FollowerID      int64   `json:"follower_id,omitempty"`
	FollowerIDs     []int64 `json:"follower_ids,omitempty"`
	TargetID        int64   `json:"target_id,omitempty"`
	IsUpgradeLeader bool    `json:"is_upgrade_leader,omitempty"`
	ParentID        int64   `json:"parent_id,omitempty"`
}

type CopyTradeInfo struct {
	AllowGuest bool `json:"allowGuest,omitempty"`
}

func (c CopyTradeInfo) Parse(ctx context.Context, origin string) (*CopyTradeInfo, error) {
	if origin == "" {
		return nil, nil
	}
	if err := util.JsonUnmarshalString(origin, &c); err != nil {
		glog.Info(ctx, "openapi copyTradeInfo JsonUnmarshalString error", glog.String("error", err.Error()))
		return nil, err
	}
	return &c, nil
}

type CopyTradeIface interface {
	GetCopyTradeData(ctx context.Context, uid int64) (ids *CopyTrade, err error)
	DeleteCopyTradeData(uid int64)
}

type CopyTradeService struct {
	client         ggrpc.ClientConnInterface
	copytradeCache *freecache.Cache
}

// GetCopyTradeService returns a copy trade service
func GetCopyTradeService(userService zrpc.RpcClientConf) (CopyTradeIface, error) {
	var err error
	copyTradeOnce.Do(func() {
		rpcClient, err := zrpc.NewClient(userService, zrpc.WithDialOptions(service.DefaultDialOptions...))
		if err != nil {
			glog.Errorf(context.Background(), "copytrade dial user-service-private fail, error=%v", err)
			galert.Error(context.Background(), "copytrade dial user-service-private fail", galert.WithField("error", err))
			return
		}

		copyTradeService = &CopyTradeService{
			client: rpcClient.Conn(),
		}

		// copy trade cache
		size := config.Global.Data.CacheSize.CopyTradeCacheSize
		if size < defaultCopyTradeCacheSize {
			size = defaultCopyTradeCacheSize
		}
		copyTradeService.copytradeCache = freecache.NewCache(size * 1024 * 1024)

		_ = diagnosis.Register(&cpDiagnose{
			cfg:  userService,
			kCfg: config.Global.KafkaCli,
			svc:  copyTradeService,
		})
		registerCopytradeAdmin()
	})
	if copyTradeService == nil {
		gmetric.IncDefaultError("copytrade", "empty_copytrade_service")
		err = fmt.Errorf("copyTradeService empty: %w", err)
		glog.Error(context.TODO(), "copyTradeService empty", glog.String("err", err.Error()))
		return nil, err
	}

	return copyTradeService, err
}

func (c *CopyTradeService) GetCopyTradeData(ctx context.Context, memberID int64) (*CopyTrade, error) {
	// check the cache
	val, ok := c.getCopyTradeData(memberID)
	if ok {
		glog.Debug(ctx, "CopyTradeData cache hit", glog.Int64("uid", memberID), glog.Any("CopyTradeData", val))
		return val, nil
	}

	glog.Debug(ctx, "CopyTradeData cache not hit", glog.Int64("uid", memberID))
	data, err := c.queryCopyTradeData(ctx, memberID)
	if err != nil {
		return nil, err
	}

	// set the cache
	if err = c.setCopyTradeData(memberID, data); err != nil {
		glog.Error(ctx, "setCopyTradeData error", glog.String("err", err.Error()))
	}

	return data, nil
}

func (c *CopyTradeService) queryCopyTradeData(ctx context.Context, memberID int64) (*CopyTrade, error) {
	copyTradeConsumeOnce.Do(func() {
		c.consumeCopyTradeData()
	})

	copyTradeInfo := &CopyTrade{} // leaderID,followerID,targetID

	req := &user.QueryRelationByMemberRequest{
		MemberIds:          []int64{memberID},
		MemberRelationType: euser.MemberRelationType_MEMBER_RELATION_TYPE_UNSPECIFIED,
		IsFilter:           []euser.SubMemberStatus{euser.SubMemberStatus_SUB_MEMBER_STATUS_NORMAL},
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := user.NewMemberInternalClient(c.client).GetRelationByMemberIDCommon(ctx, req)
	if err != nil {
		return nil, berror.NewUpStreamErr(berror.UpstreamErrUserServiceInvokeFailed, "QueryRelationByMember error", err.Error(), req.String())
	}
	if resp.Error != nil {
		return nil, berror.NewUpStreamErr(berror.UpstreamErrUserServiceInvokeFailed, "GetRelationByMemberIDCommon resp error", resp.Error.String(), req.String())
	}

	relation, ok1 := resp.Result[memberID]
	if !ok1 {
		glog.Info(ctx, "QueryRelationByMember not data", glog.Int64("uid", memberID))
		return copyTradeInfo, nil
	}

	hasOldLeader := false
	for _, m := range relation.MemberRelations {
		if m.OwnerMemberId > 0 && m.OwnerMemberId != memberID && m.TargetMemberId != memberID {
			continue
		}
		if m.MemberRelationType == euser.MemberRelationType_MEMBER_RELATION_TYPE_COPY_TRADE_LEADER {
			hasOldLeader = true
		}
	}
	for _, m := range relation.MemberRelations {
		if m.OwnerMemberId > 0 && m.OwnerMemberId != memberID && m.TargetMemberId != memberID {
			continue
		}
		isOldLeader := m.MemberRelationType == euser.MemberRelationType_MEMBER_RELATION_TYPE_COPY_TRADE_LEADER
		isUpgradeLeader := m.CopytradeUpgrade == success
		isFollower := m.MemberRelationType == euser.MemberRelationType_MEMBER_RELATION_TYPE_COPY_TRADE_FOLLOWER
		if isFollower {
			copyTradeInfo.FollowerIDs = append(copyTradeInfo.FollowerIDs, m.TargetMemberId)
		} else {
			// 如果是母账号登录
			if memberID == m.MemberId {
				if isOldLeader {
					copyTradeInfo.LeaderIDs = append(copyTradeInfo.LeaderIDs, m.TargetMemberId)
					copyTradeInfo.ParentID = m.OwnerMemberId
					if m.OwnerMemberId == 0 {
						copyTradeInfo.ParentID = m.MemberId
					}
					copyTradeInfo.IsUpgradeLeader = isUpgradeLeader
				}
			} else {
				if hasOldLeader {
					if isOldLeader {
						copyTradeInfo.LeaderIDs = append(copyTradeInfo.LeaderIDs, m.TargetMemberId)
						copyTradeInfo.ParentID = m.OwnerMemberId
						if m.OwnerMemberId == 0 {
							copyTradeInfo.ParentID = m.MemberId
						}
						copyTradeInfo.IsUpgradeLeader = isUpgradeLeader
					}
				} else if isUpgradeLeader {
					copyTradeInfo.LeaderIDs = append(copyTradeInfo.LeaderIDs, m.TargetMemberId)
					copyTradeInfo.ParentID = m.OwnerMemberId
					if m.OwnerMemberId == 0 {
						copyTradeInfo.ParentID = m.MemberId
					}
					copyTradeInfo.IsUpgradeLeader = isUpgradeLeader
				}
			}
		}
	}

	// sort leader list
	sort.Slice(copyTradeInfo.LeaderIDs, func(i, j int) bool {
		return copyTradeInfo.LeaderIDs[i] < copyTradeInfo.LeaderIDs[j]
	})
	if len(copyTradeInfo.LeaderIDs) > 0 {
		copyTradeInfo.LeaderID = copyTradeInfo.LeaderIDs[0]
	}

	// sort follower list
	sort.Slice(copyTradeInfo.FollowerIDs, func(i, j int) bool {
		return copyTradeInfo.FollowerIDs[i] < copyTradeInfo.FollowerIDs[j]
	})
	if len(copyTradeInfo.FollowerIDs) > 0 {
		copyTradeInfo.FollowerID = copyTradeInfo.FollowerIDs[0]
	}

	// set target
	copyTradeInfo.TargetID = copyTradeInfo.LeaderID
	if copyTradeInfo.TargetID <= 0 {
		copyTradeInfo.TargetID = copyTradeInfo.FollowerID
	}

	return copyTradeInfo, nil
}

// get copy trade user data from cache
func (c *CopyTradeService) getCopyTradeData(memberID int64) (resp *CopyTrade, ok bool) {
	key := c.getCopyTradeCacheKey(memberID)
	val, err := c.copytradeCache.Get([]byte(key))
	if err != nil {
		return
	}
	resp = &CopyTrade{}
	if err := util.JsonUnmarshal(val, resp); err != nil {
		return
	}
	return resp, true
}

// set copy trade user data from cache
func (c *CopyTradeService) setCopyTradeData(memberID int64, relation *CopyTrade) error {
	key := c.getCopyTradeCacheKey(memberID)
	if val, err := util.JsonMarshal(relation); err != nil {
		return err
	} else {
		random := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(14400)
		return c.copytradeCache.Set([]byte(key), val, copyTradeCacheExpireSeconds+random)
	}
}

// DeleteCopyTradeData del copy trade user data from cache
func (c *CopyTradeService) DeleteCopyTradeData(memberID int64) {
	key := c.getCopyTradeCacheKey(memberID)
	c.copytradeCache.Del([]byte(key))
}

func (c *CopyTradeService) consumeCopyTradeData() {
	kafkaconsume.AsyncHandleKafkaMessage(context.Background(), constant.EventSpecialSubMemberCreate,
		config.Global.KafkaCli, c.handleCopyTradeData, onErr)
}

const (
	copytradeUpgrade        = "copytrade_upgrade"
	ownerMemberId           = "owner_member_id"
	copytradeUpgradeSuccess = "SUCCESS"
)

type SpecialSubMemberCreateMsg struct {
	MemberID           int64  `json:"member_id"`            // 主账号
	SubMemberID        int64  `json:"sub_member_id"`        // 子账号
	MemberRelationType int32  `json:"member_relation_type"` // 子账号类型
	RequestID          string `json:"request_id"`           // 请求唯一标识
	Status             string `json:"status"`               // 操作状态 success、fail
}

func (c *CopyTradeService) handleCopyTradeData(ctx context.Context, message *gkafka.Message) {
	var msg SpecialSubMemberCreateMsg
	if err := util.JsonUnmarshal(message.Value, &msg); err != nil {
		glog.Error(ctx, "HandleCopyTradeData Unmarshal error", glog.String("error", err.Error()))
		return
	}
	glog.Info(ctx, "copytrade msg", glog.Any("data", msg), glog.Int64("offset", message.Offset))

	if euser.MemberRelationType(msg.MemberRelationType) != euser.MemberRelationType_MEMBER_RELATION_TYPE_COPY_TRADE_LEADER &&
		euser.MemberRelationType(msg.MemberRelationType) != euser.MemberRelationType_MEMBER_RELATION_TYPE_COPY_TRADE_FOLLOWER {
		return
	}

	key := c.getCopyTradeCacheKey(msg.MemberID)
	c.copytradeCache.Del([]byte(key))
}

func (c *CopyTradeService) getCopyTradeCacheKey(memberID int64) string {
	return fmt.Sprintf("%dcopytrade", memberID)
}

func onErr(err *gkafka.ConsumerError) {
	if err != nil {
		galert.Error(context.Background(), "copytrade consumer err "+err.Error())
	}
}

type cpDiagnose struct {
	cfg  zrpc.RpcClientConf
	kCfg kafka.UniversalClientConfig
	svc  *CopyTradeService
}

func (o *cpDiagnose) Key() string {
	return "copytrade"
}

func (o *cpDiagnose) Diagnose(ctx context.Context) (interface{}, error) {
	resp := make(map[string]interface{})
	resp["kafka"] = diagnosis.DiagnoseKafka(ctx, constant.EventSpecialSubMemberCreate, o.kCfg)
	resp["grpc"] = diagnosis.DiagnoseGrpcDependency(ctx, o.cfg)
	return resp, nil
}
