package user

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/glog"

	"bgw/pkg/common/constant"
)

func (as *AccountService) GetUnifiedMarginAccountID(ctx context.Context, uid int64, bizType int) (int64, error) {
	key := fmt.Sprintf("%dunified_margin%d", uid, bizType)
	aid, err := as.accountCache.Get([]byte(key))
	if err == nil {
		glog.Debug(ctx, "unified margin cache hit", glog.String("key", key))
		return cast.BytesToInt64(aid), nil
	}
	glog.Debug(ctx, "unified margin cache not hit", glog.String("key", key))

	status, err := as.QueryMemberTag(ctx, uid, UnifiedMarginTag)
	if err != nil {
		return 0, err
	}
	if !(status == UnifiedStateSuccess) {
		// not unified member_id, save cache and return, the value is 0
		random := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(14400)
		if err = as.accountCache.Set([]byte(key), cast.Int64ToBytes(0), accountCacheExpireSeconds+random); err != nil {
			glog.Error(ctx, "set unified margin invalid cache error", glog.String("key", key), glog.String("unified", status), glog.String("error", err.Error()))
		}
		return 0, nil
	}

	id, err := as.getAccountID(ctx, uid, constant.IdxAppTypeUNIFIED, int32(bizType))
	if err != nil {
		return 0, err
	}
	if err = as.accountCache.Set([]byte(key), cast.Int64ToBytes(id), 0); err != nil {
		glog.Error(ctx, "set unified margin valid cache error", glog.String("key", key), glog.String("unified", status), glog.String("error", err.Error()))
	}

	return id, nil
}

func (as *AccountService) GetUnifiedTradingAccountID(ctx context.Context, uid int64, bizType int) (int64, error) {
	key := fmt.Sprintf("%dunified_trading%d", uid, bizType)
	aid, err := as.accountCache.Get([]byte(key))
	if err == nil {
		glog.Debug(ctx, "unified trading cache hit", glog.String("key", key))
		return cast.BytesToInt64(aid), nil
	}
	glog.Debug(ctx, "unified trading cache not hit", glog.String("key", key))

	status, err := as.QueryMemberTag(ctx, uid, UnifiedTradingTag)
	if err != nil {
		return 0, err
	}
	if !(status == UnifiedStateSuccess) {
		random := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(14400)
		if err = as.accountCache.Set([]byte(key), cast.Int64ToBytes(0), accountCacheExpireSeconds+random); err != nil {
			glog.Error(ctx, "set unified trading invalid cache error", glog.String("key", key), glog.String("unified", status), glog.String("error", err.Error()))
		}
		return 0, nil
	}

	id, err := as.getAccountID(ctx, uid, constant.IdxAppTypeUNIFIED, int32(bizType))
	if err != nil {
		return 0, err
	}
	if err = as.accountCache.Set([]byte(key), cast.Int64ToBytes(id), 0); err != nil {
		glog.Error(ctx, "set unified trading valid cache error", glog.String("key", key), glog.String("unified", status), glog.String("error", err.Error()))
	}

	return id, nil
}
