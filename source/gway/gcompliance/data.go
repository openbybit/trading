package gcompliance

import (
	"sync"

	compliance "code.bydev.io/cht/customer/kyc-stub.git/pkg/bybit/compliancewall/strategy/v1"
)

// blackList is id black list
type blackList struct {
	sync.RWMutex
	list []interface{}
}

func newBlackList() *blackList {
	return &blackList{
		list: make([]interface{}, 0),
	}
}

func (w *blackList) Contains(id interface{}) bool {
	w.RLock()
	defer w.RUnlock()
	for _, v := range w.list {
		if v == id {
			return true
		}
	}
	return false
}

func (w *blackList) Set(ids ...interface{}) {
	w.Lock()
	w.list = ids
	w.Unlock()
}

func (w *blackList) IsEmpty() bool {
	w.RLock()
	defer w.RUnlock()

	return len(w.list) == 0
}

// UserInfo include white list info and kyc info
type UserInfo struct {
	WhiteListStatus bool
	KycStatus       bool
	Country         string
	KycLevel        int32
	Groups          []*compliance.ComplianceUserItem
}

// Result is result of hit compliance wall
type Result interface {
	GetEndPointExec() string         // 执行动作(弹窗、引导kyc等)
	GetEndPointsArgs() EndpointsArgs // 动作的额外参数(目前没有使用)
}

var emptyResult = &config{
	EndpointExec: "empty result",
	EndArgs:      EndpointsArgs{},
}

type config struct {
	EndpointExec string        // 执行动作（弹窗、引导kyc等)
	EndArgs      EndpointsArgs // 动作的额外参数
}

func (r *config) GetEndPointExec() string {
	return r.EndpointExec
}

func (r *config) GetEndPointsArgs() EndpointsArgs {
	return r.EndArgs
}

type EndpointsArgs struct {
}

// complianceStrategy is the memory data structure holding strategies
type complianceStrategy struct {
	sync.RWMutex
	strategies map[string]map[string]map[string]*config // scene => Country => userType => kyc_result
}

func newComplianceStrategy() *complianceStrategy {
	return &complianceStrategy{
		strategies: make(map[string]map[string]map[string]*config),
	}
}

func (k *complianceStrategy) Exist(scene string) bool {
	k.RLock()
	_, ok := k.strategies[scene]
	k.RUnlock()

	return ok
}

func (k *complianceStrategy) Get(scene string) map[string]map[string]*config {
	k.RLock()
	defer k.RUnlock()
	return k.strategies[scene]
}

func (k *complianceStrategy) Match(scene, location, userType, country string) (*config, bool) {
	k.RLock()
	defer k.RUnlock()
	return k.match(scene, location, userType, country)
}

func (k *complianceStrategy) match(scene, location, userType, country string) (*config, bool) {
	sceneConf, ok := k.strategies[scene]
	if !ok || sceneConf == nil {
		return nil, false
	}

	locationConf, ok := sceneConf[location]
	if !ok || locationConf == nil {
		if location != country {
			locationConf, ok = sceneConf[country]
		}
		if !ok || locationConf == nil {
			locationConf, ok = sceneConf[locationDefault]
			if !ok || locationConf == nil {
				return nil, false
			}
		}
	}

	res, ok := locationConf[userType]
	if res == nil {
		return nil, false
	}
	return res, ok
}

func (k *complianceStrategy) Update(strategies map[string]map[string]map[string]*config) {
	k.Lock()
	for scene, v := range strategies {
		k.strategies[scene] = v
	}
	k.Unlock()
}
