package manual_intervent

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/config_center"
	"bgw/pkg/config_center/nacos"
	"bgw/pkg/server/filter/biz_limiter/rate"
	"bgw/pkg/server/metadata"
	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/container"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/frameworks/byone/core/threading"
	"github.com/armon/go-radix"
	jsoniter "github.com/json-iterator/go"
)

const (
	defaultRadixTreeValue    = 1
	cacheCleanInterval       = 24 * time.Hour
	manualInterveneFile      = "manual_intervene.json"
	cacheKeyFormat           = "ruleType:%s_%s"
	requestUrlCacheKeyFormat = "ruleType:%s_httpMethod:%s_path:%s"
)

type InterveneLimiter struct {
	observer.EmptyListener

	once         sync.Once
	configCenter config_center.Configure

	lock                   sync.RWMutex
	enableRuleTypesInOrder []string
	limiterCacheMap        map[string]limiter
	pathRadixTree          *radix.Tree
}

func (il *InterveneLimiter) Init(ctx context.Context) {
	il.once.Do(func() {
		_ = il.doInit(ctx)
	})
}

func (il *InterveneLimiter) doInit(ctx context.Context) error {
	il.pathRadixTree = radix.New()
	var err error
	if il.configCenter, err = nacos.NewNacosConfigure(
		context.Background(),
		nacos.WithGroup(constant.BGW_GROUP),              // specified group
		nacos.WithNameSpace(constant.BGWConfigNamespace), // namespace isolation
	); err != nil {
		glog.Error(ctx, "[global interveneLimiter]new nacos configure failed", glog.String("err", err.Error()))
		galert.Error(ctx, "[global interveneLimiter]new nacos configure failed", galert.WithField("err", err))
		return err
	}

	if e := il.configCenter.Listen(ctx, manualInterveneFile, il); e != nil {
		glog.Error(ctx, "[global interveneLimiter]listen config failed", glog.String("err", e.Error()))
		galert.Error(ctx, "[global interveneLimiter]listen config failed", galert.WithField("err", e))
		return err
	}

	il.startPeriodicCacheClean()
	return nil
}

// OnEvent config listen event handle
func (il *InterveneLimiter) OnEvent(event observer.Event) error {
	e, ok := event.(*observer.DefaultEvent)
	if !ok || e.Value == "" {
		glog.Error(context.Background(), "[global interveneLimiter]event is nil or value is empty", glog.Bool("ok", ok))
		galert.Error(context.Background(), "[global interveneLimiter]event is nil or value is empty", galert.WithField("ok", ok))
		return errors.New("event is nil or value is empty")
	}

	//e.value change to json config
	cfg := &config{}
	if err := jsoniter.Unmarshal([]byte(e.Value), cfg); err != nil {
		glog.Error(context.Background(), "[global interveneLimiter]unmarshal config failed", glog.String("err", err.Error()))
		galert.Error(context.Background(), "[global interveneLimiter]unmarshal config failed", galert.WithField("err", err))
		return err
	}

	glog.Info(context.Background(), "[global interveneLimiter]config", glog.Any("config", cfg))

	if err := cfg.validConfig(); err != nil {
		glog.Error(context.Background(), "[global interveneLimiter]config validate failed", glog.String("err", err.Error()))
		galert.Error(context.Background(), "[global interveneLimiter]config validate failed", galert.WithField("err", err))
		return nil
	}

	glog.Info(context.Background(), "[global interveneLimiter]config validate success")

	il.loadRules(cfg)
	return nil
}

func (il *InterveneLimiter) loadRules(interventionConfig *config) {

	if !interventionConfig.Enable || len(interventionConfig.Rules) == 0 {
		il.lock.Lock()
		defer il.lock.Unlock()
		il.limiterCacheMap = nil
		return
	}

	limiterMap := make(map[string]limiter)
	var ruleTypeInOrder []string
	ruleTypeSet := container.NewSet()
	for _, rule := range interventionConfig.Rules {
		if !rule.Enable || !rule.matchEffectiveEnvName() || !rule.EffectPeriod.inPeriod() {
			continue
		}

		ruleCacheMap := il.buildCacheAndInsertRadixTree(rule)
		for k, v := range ruleCacheMap {
			limiterMap[k] = v
		}

		if ruleTypeSet.Contains(rule.RuleType) {
			continue
		}
		ruleTypeSet.Add(rule.RuleType)
		ruleTypeInOrder = append(ruleTypeInOrder, rule.RuleType)
	}

	sort.Slice(ruleTypeInOrder, func(i, j int) bool {
		return ruleTypeMap[ruleTypeInOrder[i]] < ruleTypeMap[ruleTypeInOrder[j]]
	})

	il.lock.Lock()
	defer il.lock.Unlock()
	il.enableRuleTypesInOrder = ruleTypeInOrder
	il.limiterCacheMap = limiterMap
}

func (il *InterveneLimiter) buildCacheAndInsertRadixTree(rule *rule) map[string]limiter {
	limiterMap := make(map[string]limiter)
	startTimeInUTC, _ := time.Parse(timeLayOut, rule.EffectPeriod.StartDateInUTC)
	endTimeInUTC, _ := time.Parse(timeLayOut, rule.EffectPeriod.EndDateInUTC)

	switch rule.RuleType {
	case clientIpRuleType, requestHostRuleType, clientOpFromRule:
		for _, ext := range rule.ExtData {
			var key string
			switch rule.RuleType {
			case clientIpRuleType:
				key = ext.ClientIp
			case requestHostRuleType:
				key = ext.RequestHost
			case clientOpFromRule:
				key = ext.ClientOpFrom
			}
			cacheKey := buildCacheKey(rule.RuleType, key)
			limiterMap[cacheKey] = limiter{startInUTC: startTimeInUTC, endInUTC: endTimeInUTC, ruleType: rule.RuleType,
				limit: rate.NewLimiter(rate.Limit(rule.Limit), rule.Limit)}
		}

		return limiterMap
	case requestUrlRule:
		for _, ext := range rule.ExtData {
			limit := rule.Limit
			requestUrl := ext.RequestUrl
			if requestUrl.Limit >= 0 {
				limit = requestUrl.Limit
			}

			il.pathRadixTree.Insert(requestUrl.Path, defaultRadixTreeValue)

			if requestUrl.HttpMethod == "*" {
				for _, method := range allowedMethodsSet.Values() {
					cacheKey := buildRequestUrlCacheKeyIfOnly(rule.RuleType, method.(string), requestUrl.Path)
					limiterMap[cacheKey] = limiter{startInUTC: startTimeInUTC, endInUTC: endTimeInUTC, ruleType: rule.RuleType,
						limit: rate.NewLimiter(rate.Limit(limit), limit)}
				}
				continue
			}

			cacheKey := buildRequestUrlCacheKeyIfOnly(rule.RuleType, requestUrl.HttpMethod, requestUrl.Path)
			limiterMap[cacheKey] = limiter{startInUTC: startTimeInUTC, endInUTC: endTimeInUTC, ruleType: rule.RuleType,
				limit: rate.NewLimiter(rate.Limit(limit), limit)}
		}

		return limiterMap
	default:
		return limiterMap
	}

}

func buildCacheKey(ruleType string, key string) string {
	return fmt.Sprintf(cacheKeyFormat, ruleType, key)
}

func buildRequestUrlCacheKeyIfOnly(ruleType string, httpMethod string, path string) string {
	return buildRequestUrlCacheKey(ruleType, httpMethod, path, nil)
}

func buildRequestUrlCacheKey(ruleType string, httpMethod string, path string, pathRadixTree *radix.Tree) string {
	if pathRadixTree == nil {
		return fmt.Sprintf(requestUrlCacheKeyFormat, ruleType, httpMethod, path)
	}

	prefix, _, exist := pathRadixTree.LongestPrefix(path)
	if !exist {
		return fmt.Sprintf(requestUrlCacheKeyFormat, ruleType, httpMethod, path)
	}

	return fmt.Sprintf(requestUrlCacheKeyFormat, ruleType, httpMethod, prefix)
}

func (il *InterveneLimiter) Intervene(ctx *types.Ctx) bool {
	md := metadata.MDFromContext(ctx)

	il.lock.RLock()
	defer il.lock.RUnlock()
	if il.limiterCacheMap == nil || il.enableRuleTypesInOrder == nil || len(il.enableRuleTypesInOrder) == 0 {
		return false
	}

	for _, v := range il.enableRuleTypesInOrder {

		var cacheKey string
		switch v {
		case clientIpRuleType:
			cacheKey = buildCacheKey(clientIpRuleType, md.GetClientIP())
		case requestHostRuleType:
			cacheKey = buildCacheKey(requestHostRuleType, md.Extension.Host)
		case clientOpFromRule:
			cacheKey = buildCacheKey(clientOpFromRule, md.Extension.OpFrom)
		case requestUrlRule:
			cacheKey = buildRequestUrlCacheKey(requestUrlRule, md.Method, md.Path, il.pathRadixTree)
		default:
			continue
		}

		currentLimiter := il.limiterCacheMap[cacheKey]
		now := time.Now().UTC()
		if now.After(currentLimiter.endInUTC) || now.Before(currentLimiter.startInUTC) {
			continue
		}
		return !currentLimiter.limit.Allow()
	}

	return false
}

func (il *InterveneLimiter) startPeriodicCacheClean() {
	ticker := time.NewTicker(cacheCleanInterval)
	threading.GoSafe(func() {
		for {
			select {
			case <-ticker.C:
				il.doCleanCache()
			}
		}
	})
}

func (il *InterveneLimiter) doCleanCache() {
	glog.Info(context.Background(), "[global interveneLimiter]cleanCache")
	rules := make([]string, len(ruleTypeMap))

	il.lock.Lock()
	defer il.lock.Unlock()
	for k, v := range il.limiterCacheMap {
		now := time.Now().UTC()
		if now.After(v.endInUTC) {
			glog.Info(context.Background(), "[global interveneLimiter]cleanCache delete", glog.String("cacheKey", k))
			delete(il.limiterCacheMap, k)
		}

		if now.After(v.startInUTC) && now.Before(v.endInUTC) {
			rules = append(rules, v.ruleType)
		}
	}

	sort.Slice(rules, func(i, j int) bool {
		return ruleTypeMap[rules[i]] < ruleTypeMap[rules[j]]
	})

	il.enableRuleTypesInOrder = rules
	glog.Info(context.Background(), "[global interveneLimiter]cleanCache success", glog.Any("enableRuleTypeSet", il.enableRuleTypesInOrder))
}
