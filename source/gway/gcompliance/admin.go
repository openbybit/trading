package gcompliance

import (
	"context"
	"errors"

	compliance "code.bydev.io/cht/customer/kyc-stub.git/pkg/bybit/compliancewall/strategy/v1"
)

// GetUserInfo get user info from local cache.
func (w *wall) GetUserInfo(ctx context.Context, uid int64) (UserInfo, error) {
	var res UserInfo
	if !w.withCache {
		return res, nil
	}
	ui, ok := w.userInfos.Get(uid)
	if !ok {
		return res, errors.New("[compliance wall] user info does not exist")
	}

	res, ok = ui.(UserInfo)
	if !ok {
		return res, errors.New("[compliance wall] bad user info type")
	}

	return res, nil
}

// GetStrategy get strategy from local cache, res may be nil.
func (w *wall) GetStrategy(ctx context.Context, strategy string) map[string]map[string]*config {
	if !w.withCache {
		return nil
	}

	return w.cs.Get(strategy)
}

// RemoveUserInfo remove user info from local cache.
func (w *wall) RemoveUserInfo(ctx context.Context, uid int64) {
	if !w.withCache {
		return
	}
	w.userInfos.Remove(uid)
}

// QuerySiteConfig query all site config from local cache.
func (w *wall) QuerySiteConfig(ctx context.Context) map[string]*compliance.SitesConfigItemConfig {
	if !w.withCache {
		return nil
	}

	w.siteMutex.RLock()
	defer w.siteMutex.RUnlock()
	return w.siteCfg
}
