package etcd

import (
	"context"
	"fmt"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/getcd"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gsechub"

	"bgw/pkg/common/constant"
	"bgw/pkg/config"
)

// NewConfigClient create new Client
func NewConfigClient(ctx context.Context) (getcd.Client, error) {
	rc := &config.Global.Etcd
	if rc == nil {
		return nil, fmt.Errorf("no etcd config")
	}

	password := rc.Password
	if rc.Password != "" {
		passwd, err := gsechub.Decrypt(password)
		if err != nil {
			glog.Info(ctx, "sechub.Decrypt etcd password error", glog.String("err", err.Error()))
			return nil, err
		}
		password = passwd
	}

	newClient, err := getcd.NewClient(ctx,
		getcd.WithName(constant.EtcdConfigKey),
		getcd.WithEndpoints(rc.GetAddresses()...),
		getcd.WithTimeout(rc.GetTimeout(config.DefaultTimeOut)),
		getcd.WithHeartbeat(cast.ToInt(rc.GetOptions("heatbeat", "1"))),
		getcd.WithUsername(rc.Username),
		getcd.WithPassword(password),
	)

	if err != nil {
		glog.Error(ctx, "new etcd client error", glog.String("error", err.Error()))
		return nil, err
	}
	return newClient, nil
}
