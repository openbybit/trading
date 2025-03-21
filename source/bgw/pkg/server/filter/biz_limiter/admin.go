package biz_limiter

import (
	"context"
	"errors"

	"code.bydev.io/fbu/gateway/gway.git/gapp"

	"bgw/pkg/common/constant"
)

func init() {
	// params order:app,uid,group,symbol
	// curl 'http://localhost:6480/admin?cmd=quota_loader&params=futures,4256593,order,v3'
	// curl 'http://localhost:6480/admin?cmd=quota_loader&params=spot,4256593,${group},${symbol},${key}'
	// curl 'http://localhost:6480/admin?cmd=quota_loader&params=option,4256593,${group}'
	gapp.RegisterAdmin("quota_loader", "query quota loader", onGetQuotaWithKey)
}

func onGetQuotaWithKey(args gapp.AdminArgs) (interface{}, error) {
	var app = args.GetStringAt(0)

	if app == "" {
		return "failed", errors.New("param app cannot be empty")
	}

	switch app {
	case constant.AppTypeFUTURES:
		return queryFutureQuota(args)
	case constant.AppTypeSPOT:
		return querySpotQuota(args)
	case constant.AppTypeOPTION:
		return queryOptionQuota(args)
	default:
		return "failed", errors.New("param invalid:app")
	}

}

func queryOptionQuota(args gapp.AdminArgs) (interface{}, error) {
	var app = args.GetStringAt(0)
	loader := newQuotaLoader(app)
	err := loader.init(context.Background())
	if err != nil {
		return "failed", err
	}
	var uid = args.GetInt64At(1)
	var group = args.GetStringAt(2)

	quota := loader.getQuota(context.Background(), uid, group)

	return quota, nil
}

func querySpotQuota(args gapp.AdminArgs) (interface{}, error) {
	var app = args.GetStringAt(0)
	loaderV2 := newQuotaLoaderV2(app)
	err := loaderV2.init(context.Background())
	if err != nil {
		return "failed", err
	}

	var uid = args.GetInt64At(1)
	var group = args.GetStringAt(2)
	var symbol = args.GetStringAt(3)
	var key = args.GetStringAt(4)

	rateParam := &rateParams{
		uid:    uid,
		group:  group,
		symbol: symbol,
		key:    key,
	}

	quota := loaderV2.getQuota(context.Background(), rateParam)
	return quota, nil
}

func queryFutureQuota(args gapp.AdminArgs) (interface{}, error) {
	manager, err := NewQuotaManager(context.Background())
	if err != nil {
		return "failed", err
	}

	var app = args.GetStringAt(0)
	var uid = args.GetInt64At(1)
	var group = args.GetStringAt(2)
	var symbol = args.GetStringAt(3)

	quota, err := manager.GetQuota(context.Background(), uid, app, group, symbol)
	if err != nil {
		return "failed", err
	}
	return quota, nil
}
