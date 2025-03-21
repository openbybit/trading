package biz_limiter

import (
	"fmt"
	"strings"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gredis"
)

func getAutoQuota(ctx *types.Ctx, app string, md *metadata.Metadata, rule limitRule) (gredis.Limit, string, error) {
	rule.Symbol = false
	ks := rule.Values(ctx, v3Symbol)
	keys := append([]string{app}, ks...)
	key := strings.Join(keys, ":") // redis key

	if !rule.UID {
		return rule.Limit, key, nil
	}

	if !rule.EnableCustomRate {
		return rule.Limit, key, nil
	}

	var quota int
	switch app {
	case constant.AppTypeFUTURES:
		rates, err := rateLimitMgr.GetQuota(service.GetContext(ctx), md.UID, app, rule.Group, v3Symbol)
		if err != nil {
			glog.Error(ctx, "rateLimitMgr.GetQuota error", glog.String("error", err.Error()))
		} else {
			quota = int(rates)
		}
		rule.Path = true
		rule.Group = ""
		ks := rule.Values(ctx, v3Symbol)
		keys := append([]string{app}, ks...)
		key = strings.Join(keys, ":") // uid + path
	case constant.AppTypeSPOT:
		value, ok := limitV2Loaders.Load(app)
		if !ok {
			glog.Error(ctx, "load loader failed", glog.String("app", app), glog.String("key", key))
		} else {
			loader, ok := value.(*quotaLoaderV2)
			if !ok {
				glog.Error(ctx, "get loader failed", glog.String("app", app), glog.String("key", key))
			} else {
				quota = loader.getQuotaByUid(ctx, key, md.UID)
			}
		}
	case constant.AppTypeOPTION:
		key = getOptionKey(md.UID, rule.Group)
		quota = getUnifiedQuota(ctx, md.UID, rule.Group)
	default:
		return gredis.Limit{}, "", fmt.Errorf("unkown app: %s, %d", app, md.UID)
	}
	// use user's quota first
	if quota > 0 {
		rule.Rate = quota
	}
	return rule.Limit, key, nil
}
