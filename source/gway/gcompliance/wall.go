package gcompliance

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	compliance "code.bydev.io/cht/customer/kyc-stub.git/pkg/bybit/compliancewall/strategy/v1"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/atomic"
	"google.golang.org/grpc"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cache/lru"
	"code.bydev.io/fbu/gateway/gway.git/ggrpc/pool"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
)

const (
	userTypeNotLogin = "USER_TYPE_GUEST"
	userTypeNotKyc   = "USER_TYPE_LOGIN"
	userTypeKyc      = "USER_TYPE_KYC"

	locationDefault = "all"
	pass            = "PASS"

	BybitSiteID = "BYBIT"

	SourceWeb     = "WEB"
	SourceApp     = "APP"
	SourceOpenapi = "OPENAPI"
)

var _ Wall = (*wall)(nil)

//go:generate mockgen -source=wall.go -destination=wall_mock.go -package=gcompliance
type Wall interface {
	CheckStrategy(ctx context.Context, brokerID int32, siteID, scene string, uid int64, country, subDivision, source, userSiteID string) (res Result, hit bool, err error)
	GetSiteConfig(ctx context.Context, brokerID int32, uid int64, siteID, product, userSiteID string) (string, *compliance.SitesConfigItemConfig, error)

	GetUserInfo(ctx context.Context, uid int64) (UserInfo, error)
	GetStrategy(ctx context.Context, strategy string) map[string]map[string]*config
	RemoveUserInfo(ctx context.Context, uid int64)
	QuerySiteConfig(ctx context.Context) map[string]*compliance.SitesConfigItemConfig

	SetCityConfig(countries, subDivisions []string)

	HandleUserWhiteListEvent([]byte) error
	HandleUserKycEvent([]byte) error
	HandleStrategyEvent([]byte) error
	HandleSiteConfigEvent([]byte) error
}

type wall struct {
	// connection with server
	registry  string // registry of compliance wall server, for service discovery
	group     string // group of compliance wall server, for service discovery
	namespace string // namespace of compliance wall server, for service discovery
	discovery Discovery
	index     *atomic.Int32 // use for service select

	address string // set address directly, not use service discovery.

	connPool pool.Pools

	conn grpc.ClientConnInterface // set grpc connection directly, usually set byone zrpc conn

	withCache bool // if local cache is needed.

	// local data
	brokerBL  *blackList
	siteBL    *blackList
	userInfos lru.LRUCache
	cs        *complianceStrategy

	status    *atomic.Int32
	siteMutex sync.RWMutex
	jsonCfg   map[string]string                            // site.product => json cfg
	siteCfg   map[string]*compliance.SitesConfigItemConfig // site.product => cfg

	locationMutex sync.RWMutex        // lock to protect countries and subDivisions
	countries     map[string]struct{} // corresponding countries of subDivisions
	subDivisions  map[string]struct{} // req from these subDivisions should match strategy with subDivision
}

func NewWall(rc RemoteCfg, withCache bool) (Wall, error) {
	w := &wall{
		status:       atomic.NewInt32(0),
		connPool:     pool.NewPools(),
		jsonCfg:      make(map[string]string),
		siteCfg:      make(map[string]*compliance.SitesConfigItemConfig),
		countries:    make(map[string]struct{}),
		subDivisions: make(map[string]struct{}),
	}

	if withCache {
		cache, _ := lru.NewLRU(400000)
		w.userInfos = cache
		w.brokerBL = newBlackList()
		w.siteBL = newBlackList()
		w.cs = newComplianceStrategy()
		w.withCache = true
	}
	switch rc.(type) {
	case *registryCfg:
		w.registry = rc.Registry()
		w.namespace = rc.Namespace()
		w.group = rc.Group()
		w.discovery = rc.Discovery()
		w.index = atomic.NewInt32(0)
		if w.registry == "" || w.namespace == "" ||
			w.group == "" || w.discovery == nil {
			return nil, fmt.Errorf("registryCfg error, registry :%s, namespace: %s, group: %s", w.registry, w.namespace, w.group)
		}
	case *addrCfg:
		w.address = rc.Addr()
		if w.address == "" {
			return nil, errors.New("bad address config")
		}
	case *connCfg:
		w.conn = rc.Conn()
		if w.conn == nil {
			return nil, errors.New("empty grpc conn")
		}
	default:
		return nil, errors.New("unknown remote cfg")
	}
	return w, nil
}

// 1. CheckStrategy check if req match compliance wall strategy. uid should be 0 if no login.
// 2. brokerID代表用户归属，siteID代表流量入口.
func (w *wall) CheckStrategy(ctx context.Context, brokerID int32, siteID, scene string, uid int64, country, sbuDivision, source, userSiteID string) (res Result, hit bool, err error) {
	if scene == "" {
		return emptyResult, false, nil
	}
	source = getSource(source)
	if !w.withCache {
		return w.remoteCheck(ctx, brokerID, siteID, scene, uid, country, sbuDivision, source, userSiteID)
	}
	return w.localCheck(ctx, brokerID, siteID, scene, uid, country, sbuDivision, source, userSiteID)
}

func (w *wall) remoteCheck(ctx context.Context, brokerID int32, siteID, scene string, uid int64, country, sbuDivision, source, userSiteID string) (Result, bool, error) {
	resp, err := w.GetComplianceConfig(ctx, brokerID, scene, uid, true, userSiteID)
	if err != nil || resp == nil {
		return emptyResult, false, err
	}
	if resp.IsWhitelist {
		return emptyResult, false, nil
	}
	blackHit := false
	// 兼容逻辑，site为空时走原来的broker id的判断逻辑
	if siteID != "" {
		for _, id := range resp.GetSites() {
			if siteID == id {
				blackHit = true
				break
			}
		}
	} else {
		for _, id := range resp.GetBrokerIds() {
			if brokerID == id {
				blackHit = true
				break
			}
		}
		siteID = BybitSiteID
	}
	if !blackHit {
		return emptyResult, false, nil
	}

	strategies := convert(resp.SceneItems...)

	// 多场景
	scenes := getScenes(w.getScene(scene, siteID), source)

	var userType string
	if uid <= 0 {
		userType = userTypeNotLogin
	} else {
		if resp.IsKyc {
			userType = userTypeKyc
			country = resp.KycCountry
		} else {
			userType = userTypeNotKyc
		}
	}

	// 多用户类型
	uts := make([]string, 0)
	for _, ug := range resp.GetUserItems() {
		if ug.Site == siteID {
			uts = append(uts, ug.Group)
		}
	}
	uts = append(uts, userType)

	location := w.getLocation(country, sbuDivision)

	for _, s := range scenes {
		for _, ut := range uts {
			sceneConf, ok := strategies[s]
			if !ok {
				continue
			}
			locationConf, ok := sceneConf[location]
			if !ok {
				if location == sbuDivision {
					locationConf, ok = sceneConf[country]
				}
				if !ok {
					locationConf, ok = sceneConf[locationDefault]
					if !ok {
						continue
					}
				}
			}

			value, ok := locationConf[ut]
			if ok {
				if value.GetEndPointExec() == pass {
					return emptyResult, false, nil
				} else {
					return value, true, nil
				}
			}
		}
	}
	return emptyResult, false, nil

}

func (w *wall) localCheck(ctx context.Context, brokerID int32, siteID, scene string, uid int64, country, sbuDivision, source, userSiteID string) (res Result, hit bool, err error) {
	ui, err := w.checkCache(ctx, brokerID, scene, uid, userSiteID)
	if err != nil {
		return emptyResult, false, err
	}

	// 兼容逻辑，site为空时走原来的broker id的判断逻辑
	if siteID != "" {
		if !w.siteBL.Contains(siteID) {
			return emptyResult, false, nil
		}
	} else {
		if !w.brokerBL.Contains(brokerID) {
			return emptyResult, false, nil
		}
		siteID = BybitSiteID
	}

	// 多场景
	scenes := getScenes(w.getScene(scene, siteID), source)

	var userType string
	if uid <= 0 {
		userType = userTypeNotLogin
	} else {
		// uid white list
		if ui.WhiteListStatus {
			return emptyResult, false, nil
		}
		if ui.KycStatus {
			userType = userTypeKyc
			country = ui.Country
		} else {
			userType = userTypeNotKyc
		}
	}
	// 多用户类型
	uts := w.getUserTypes(uid, userType, siteID)

	location := w.getLocation(country, sbuDivision)

	for _, s := range scenes {
		for _, ut := range uts {
			res, hit = w.cs.Match(s, location, ut, country)
			if hit {
				if res.GetEndPointExec() == pass {
					return emptyResult, false, nil
				} else {
					return res, true, nil
				}
			}
		}
	}

	return emptyResult, false, nil
}

// checkCache check if local cache need sync data with compliance server
func (w *wall) checkCache(ctx context.Context, brokerID int32, scene string, uid int64, userSiteID string) (UserInfo, error) {
	var (
		updateSite     = w.siteBL.IsEmpty() // if siteID black list need sync
		updateUser     = false
		updateStrategy = !w.cs.Exist(scene) && scene != "" // if scene strategy need sync
		ui             UserInfo
	)
	// if user info need sync
	if uid > 0 {
		v, ok := w.userInfos.Get(uid)
		if !ok || v == nil {
			updateUser = true
		} else {
			ui = v.(UserInfo)
		}
	}

	if updateSite || updateUser || updateStrategy {
		resp, err := w.GetComplianceConfig(ctx, brokerID, scene, uid, updateStrategy, userSiteID)
		if err != nil {
			return ui, err
		}

		// 两个同步操作
		if updateSite {
			temp := make([]interface{}, 0)
			for _, v := range resp.GetSites() {
				temp = append(temp, v)
			}
			w.siteBL.Set(temp...)
			temp = make([]interface{}, 0)
			for _, v := range resp.GetBrokerIds() {
				temp = append(temp, v)
			}
			w.brokerBL.Set(temp...)
		}

		if updateUser {
			ui = UserInfo{
				WhiteListStatus: resp.IsWhitelist,
				KycStatus:       resp.IsKyc,
				Country:         resp.KycCountry,
				KycLevel:        int32(resp.KycLevel),
				Groups:          resp.UserItems,
			}
			w.userInfos.Add(uid, ui)
			// user info update
			gmetric.IncDefaultCounter("compliance", "user_update")
		}

		if updateStrategy {
			// 兜底
			if len(resp.SceneItems) == 0 {
				w.cs.Update(map[string]map[string]map[string]*config{scene: nil})
			} else {
				w.cs.Update(convert(resp.SceneItems...))
			}
		}
	}

	return ui, nil
}

func (w *wall) GetComplianceConfig(ctx context.Context, brokerID int32, scene string, uid int64, updateStrategy bool, userSiteID string) (*compliance.GetComplianceConfigResp, error) {
	var conn grpc.ClientConnInterface

	if w.conn != nil {
		conn = w.conn
	} else {
		pConn, err := w.GetComplianceConn()
		if err != nil {
			return nil, err
		}
		defer func() { _ = pConn.Close() }()
		conn = pConn.Client()
	}

	req := &compliance.GetComplianceConfigReq{
		MemberId:            uid,
		SceneCode:           scene,
		IsObtainSceneConfig: updateStrategy,
		BrokerId:            int64(brokerID),
		UserSiteId:          userSiteID,
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	return compliance.NewComplianceAPIClient(conn).GetComplianceConfig(ctx, req)
}

func (w *wall) GetSiteConfig(ctx context.Context, brokerID int32, uid int64, siteID, product, userSiteID string) (string, *compliance.SitesConfigItemConfig, error) {
	ui, err := w.checkCache(ctx, brokerID, "", uid, userSiteID)
	if err != nil {
		return "", nil, err
	}
	if !w.brokerBL.Contains(brokerID) {
		return "", nil, nil
	}
	if ui.WhiteListStatus {
		return "", nil, nil
	}
	key := fmt.Sprintf("%s.%s", siteID, product)
	if w.status.Load() == 1 {
		w.siteMutex.RLock()
		defer w.siteMutex.RUnlock()
		return w.jsonCfg[key], w.siteCfg[key], nil
	}

	jsonRes, res, err := w.GetComplianceSiteConfig(ctx)
	if err != nil {
		return "", nil, err
	}

	w.siteMutex.Lock()
	defer w.siteMutex.Unlock()

	if w.status.Load() == 1 {
		return w.jsonCfg[key], w.siteCfg[key], nil
	}
	log.Printf("site config, %v", jsonRes)
	w.jsonCfg = jsonRes
	w.siteCfg = res
	w.status.CompareAndSwap(0, 1)
	return w.jsonCfg[key], w.siteCfg[key], nil
}

func (w *wall) GetComplianceSiteConfig(ctx context.Context) (map[string]string, map[string]*compliance.SitesConfigItemConfig, error) {
	var conn grpc.ClientConnInterface

	if w.conn != nil {
		conn = w.conn
	} else {
		pConn, err := w.GetComplianceConn()
		if err != nil {
			return nil, nil, err
		}
		defer func() { _ = pConn.Close() }()
		conn = pConn.Client()
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*1)
	defer cancel()

	req := &compliance.SitesConfigIn{}
	resp, err := compliance.NewComplianceAPIClient(conn).GetComplianceSiteConfig(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	// site.product => cfg
	jsonRes := make(map[string]string)
	res := make(map[string]*compliance.SitesConfigItemConfig)
	for _, cfg := range resp.Item {
		if cfg == nil {
			continue
		}
		key := fmt.Sprintf("%s.%s", cfg.GetSite(), cfg.GetProduct())
		c := cfg.GetConfig()
		if c == nil {
			continue
		}
		val, err := jsoniter.Marshal(c)
		if err != nil {
			continue
		}
		jsonRes[key] = string(val)
		res[key] = c
	}

	return jsonRes, res, nil
}

func (w *wall) GetComplianceConn() (pool.Conn, error) {
	var addr string
	if w.address != "" {
		addr = w.address
	} else {
		a, err := w.GetServiceRoundRobin()
		if err != nil {
			return nil, err
		}
		addr = a
	}

	conn, err := w.connPool.GetConn(context.Background(), addr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (w *wall) GetServiceRoundRobin() (addr string, err error) {
	if w.discovery == nil {
		return "", errors.New("discovery not available")
	}
	addrs := w.discovery(context.Background(), w.registry, w.namespace, w.group)
	if len(addrs) == 0 {
		return "", errors.New("[discovery]no instances found")
	}

	cur := int(w.index.Inc())
	cur = cur % len(addrs)

	return addrs[cur], nil
}

func (w *wall) SetCityConfig(countries, subDivisions []string) {
	cou := make(map[string]struct{}, len(countries))
	sds := make(map[string]struct{}, len(subDivisions))

	for _, v := range countries {
		cou[v] = struct{}{}
	}

	for _, v := range subDivisions {
		sds[v] = struct{}{}
	}

	w.locationMutex.Lock()
	w.countries = cou
	w.subDivisions = sds
	w.locationMutex.Unlock()
}

// 扩展区域的需求，特定区域以区域参与策略匹配
func (w *wall) getLocation(country, subDivision string) string {
	w.locationMutex.RLock()
	defer w.locationMutex.RUnlock()

	_, ok := w.countries[country]
	if !ok {
		return country
	}
	_, ok = w.subDivisions[subDivision]
	if !ok {
		return country
	}
	return subDivision
}

// 主站(siteID)保持不变，后续的多站点需求扩展为 scene.{siteID}
func (w *wall) getScene(scene string, siteID string) string {
	if siteID == BybitSiteID || siteID == "" {
		return scene
	}
	return fmt.Sprintf("%s.%s", scene, siteID)
}

func (w *wall) getUserTypes(uid int64, ut, site string) []string {
	res := make([]string, 0)
	ui, err := w.GetUserInfo(context.Background(), uid)
	if err != nil {
		res = append(res, ut)
		return res
	}

	for _, g := range ui.Groups {
		if g.Site == site {
			res = append(res, g.Group)
		}
	}
	res = append(res, ut)
	return res
}

func convert(configs ...*compliance.ComplianceConfigItem) map[string]map[string]map[string]*config {
	strategies := make(map[string]map[string]map[string]*config)
	for _, cfg := range configs {
		if cfg == nil {
			continue
		}
		locationCfg := make(map[string]map[string]*config)
		for _, country := range cfg.StrategyConfig {
			if country == nil {
				continue
			}
			userCfg := make(map[string]*config)
			for _, u := range country.CountryConfig {
				if u == nil {
					continue
				}
				userCfg[u.UserType] = &config{EndpointExec: u.DispositionResult}
			}
			locationCfg[country.CountryCode] = userCfg
		}
		strategies[cfg.SceneCode] = locationCfg
	}
	return strategies
}

// 多场景
func getScenes(scene, source string) []string {
	scenes := make([]string, 0)
	if (source == SourceWeb) || (source == SourceApp) || (source == SourceOpenapi) {
		scenes = append(scenes, fmt.Sprintf("%s.%s", scene, source))
	}
	// 顺序要求
	scenes = append(scenes, scene)
	return scenes
}
