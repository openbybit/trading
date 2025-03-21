package dynconfig

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
	"bgw/pkg/config_center"
	"bgw/pkg/config_center/nacos"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"
)

const (
	profileGroup   = "PROFILE_GROUP"
	brokerIdConfig = "broker_id_config.yaml"
)

var (
	brokerIdLoader         *BrokerIdLoader
	brokerIdLoaderOnceInit sync.Once
)

type BrokerRule interface {
	IsDeny(originFrom string, userBrokerId int, originSite string, userSite string) (bool, error)
}

// brokerIdCfg https://uponly.larksuite.com/wiki/OfU5wnDZticGXYkRQ4fuu16ssKg
type brokerIdCfg struct {
	XOriginFromMapRule []mappingRule `yaml:"x-origin-from-map-rule,omitempty"`
	GatewayDenyRule    []denyRule    `yaml:"gateway-deny-rule,omitempty"`
}

type mappingRule struct {
	Value    string `yaml:"value,omitempty"`
	BrokerId int    `yaml:"broker_id"`
}

type denyRule struct {
	OriginBrokerId  int           `yaml:"origin_broker_id,omitempty"`
	UserBrokerId    []int         `yaml:"user_broker_id,omitempty"`
	DenyStationType []stationRule `yaml:"deny_station_type,omitempty"`
}

type stationRule struct {
	OriginStationType int   `yaml:"origin_station_type,omitempty"`
	UserStationType   []int `yaml:"user_station_type,omitempty"`
}

type brokerRule struct {
	XOriginFromMapRule map[string]int    // origin id -> request broker id
	DenyTable          map[int]*denyInfo // request broker id -> denyInfo
}

type BrokerIdLoader struct {
	ctx        context.Context
	configure  config_center.Configure
	brokerRule atomic.Value // brokerRule
}

func GetBrokerIdLoader(ctx context.Context) (*BrokerIdLoader, error) {
	if brokerIdLoader != nil {
		return brokerIdLoader, nil
	}
	var err error
	brokerIdLoaderOnceInit.Do(func() {
		glog.Info(ctx, "GetBrokerIdLoader receive start signal")

		namespace := config.GetNamespace()
		if env.IsProduction() {
			namespace = constant.DEFAULT_NAMESPACE
		}
		nc, e := nacos.NewNacosConfigure(
			ctx,
			nacos.WithGroup(profileGroup),  // specified group
			nacos.WithNameSpace(namespace), // namespace isolation
		)
		if e != nil {
			err = e
			return
		}

		var br = &brokerRule{
			XOriginFromMapRule: map[string]int{},
			DenyTable:          map[int]*denyInfo{},
		}

		brokerMgr := &BrokerIdLoader{
			ctx:       context.Background(),
			configure: nc,
		}
		brokerMgr.brokerRule.Store(br)
		if e = nc.Listen(ctx, brokerIdConfig, brokerMgr); e != nil {
			err = e
			glog.Error(ctx, "GetBrokerIdLoader watch error", glog.String("error", e.Error()))
			return
		}
		brokerIdLoader = brokerMgr
		glog.Info(ctx, "GetBrokerIdLoader init success", glog.String("file", brokerIdConfig))
	})
	if err != nil {
		glog.Error(ctx, "GetBrokerIdLoader init error", glog.String("file", brokerIdConfig), glog.NamedError("err", err))
		galert.Error(ctx, "brokerId config init error", galert.WithField("err", err))
	}
	if brokerIdLoader == nil {
		glog.Error(ctx, "GetBrokerIdLoader init error", glog.String("file", brokerIdConfig))
		galert.Error(ctx, "brokerId config init error", galert.WithField("err", "nil brokerIdLoader"))
		return nil, fmt.Errorf("GetBrokerIdLoader error")
	}
	return brokerIdLoader, nil
}

func (b *BrokerIdLoader) decode(key, content string) (cfg *brokerIdCfg) {
	cfg = new(brokerIdCfg)
	err := util.YamlUnmarshalString(content, &cfg)
	if err != nil {
		glog.Error(b.ctx, "BrokerIdLoader YamlUnmarshalString error", glog.String("key", key), glog.String("content", content), glog.String("error", err.Error()))
		galert.Error(b.ctx, "BrokerIdLoader decode error", galert.WithField("key", key), galert.WithField("err", err))
		return nil
	}
	return
}

// OnEvent on event
func (b *BrokerIdLoader) OnEvent(event observer.Event) error {
	re, ok := event.(*observer.DefaultEvent)
	if !ok {
		return nil
	}
	glog.Info(b.ctx, "BrokerIdLoader OnEvent", glog.String("key", re.Key), glog.String("content", re.Value))
	if re.Value == "" {
		return nil
	}

	qv := b.decode(re.Key, re.Value)
	if qv == nil {
		return nil
	}

	switch re.Action {
	case observer.EventTypeAdd:
		b.setData(qv)
	case observer.EventTypeUpdate:
		b.setData(qv)
	case observer.EventTypeDel:
		glog.Warn(b.ctx, "BrokerIdLoader del data", glog.String("key", re.Key))
	}

	return nil
}

type denyInfo struct {
	User        map[int]struct{}         // user_broker_id list
	StationRule map[int]map[int]struct{} // origin_station_type -> []user_station_type
}

func newDenyInfo() *denyInfo {
	return &denyInfo{
		User:        map[int]struct{}{},
		StationRule: map[int]map[int]struct{}{},
	}
}

func (b *BrokerIdLoader) setData(qv *brokerIdCfg) {
	var br = &brokerRule{
		XOriginFromMapRule: map[string]int{},
		DenyTable:          map[int]*denyInfo{},
	}

	for _, rule := range qv.XOriginFromMapRule {
		if rule.Value == "" {
			continue
		}
		br.XOriginFromMapRule[rule.Value] = rule.BrokerId
	}

	for _, rule := range qv.GatewayDenyRule {
		var bks = make(map[int]struct{}, len(rule.UserBrokerId))
		for _, broker := range rule.UserBrokerId {
			bks[broker] = struct{}{}
		}
		di := newDenyInfo()
		di.User = bks
		var sts = make(map[int]map[int]struct{}, len(rule.DenyStationType))
		for _, st := range rule.DenyStationType {
			var usts = make(map[int]struct{}, len(st.UserStationType))
			for _, ust := range st.UserStationType {
				usts[ust] = struct{}{}
			}
			sts[st.OriginStationType] = usts
		}
		di.StationRule = sts
		br.DenyTable[rule.OriginBrokerId] = di
	}
	glog.Info(context.TODO(), "BrokerIdLoader brokerRule", glog.Any("rule", br))
	b.brokerRule.Store(br)
	galert.Info(b.ctx, "BrokerIdLoader brokerRule sync ok", galert.WithField("file", brokerIdConfig), galert.WithTitle("BrokerId Config"))
}

func (b *BrokerIdLoader) getData() *brokerRule {
	if v, ok := b.brokerRule.Load().(*brokerRule); ok {
		return v
	}
	return nil
}

// GetEventType remoting watch event
func (b *BrokerIdLoader) GetEventType() reflect.Type {
	return reflect.TypeOf(observer.DefaultEvent{})
}

// GetPriority get priority
func (b *BrokerIdLoader) GetPriority() int {
	return -1
}

func (b *BrokerIdLoader) IsDeny(originFrom string, userBrokerId int, originSite string, userSite string) (bool, error) {
	rule := b.getData()
	if rule == nil {
		return false, fmt.Errorf("BrokerIdLoader no data: %s, %d", originFrom, userBrokerId)
	}
	originBrokerId := rule.XOriginFromMapRule[originFrom]

	ubks, ok := rule.DenyTable[originBrokerId]
	if !ok || ubks == nil {
		return false, nil
	}
	if _, ok := ubks.User[userBrokerId]; ok {
		// block
		return true, nil
	}

	sts, ok := ubks.StationRule[cast.ToInt(originSite)]
	if !ok {
		return false, nil
	}
	if _, ok := sts[cast.ToInt(userSite)]; ok {
		// block
		return true, nil
	}
	return false, nil
}
