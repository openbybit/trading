package geoip

import (
	"context"
	"sync"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gapp"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/filesystem"
	"code.bydev.io/fbu/gateway/gway.git/geo"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gsechub"

	"bgw/pkg/config"
)

const (
	licenseKey   = "license_key"
	updatePeriod = "update_period"
	update       = "update"

	geoIPKey = "geo"
)

var (
	defaultGeoMgr geo.GeoManager
	once          sync.Once
)

// NewGeoManager get geo manager
func NewGeoManager() (geo.GeoManager, error) {
	var err error
	once.Do(func() {
		rc := config.Global.Geo
		path := config.Global.Data.Geo
		if path == "" {
			path = "data/geoip"
		}
		if err = filesystem.MkdirAll(path); err != nil {
			return
		}

		geoIp2License := rc.GetOptions(licenseKey, "")
		if geoIp2License != "" {
			geoIp2License, err = gsechub.Decrypt(geoIp2License)
			if err != nil {
				glog.Info(context.TODO(), "sechub.Decrypt geoip password error", glog.String("err", err.Error()))
				return
			}
		}
		cfg := geo.Config{
			DbStorePath: path,
			AutoUpdate:  cast.StringToBool(rc.GetOptions(update, "")),
		}
		if cfg.UpdatePeriod, err = time.ParseDuration(rc.GetOptions(updatePeriod, "24h")); err != nil {
			glog.Info(context.TODO(), "ParseDuration geoip error", glog.String("err", err.Error()))
			return
		}

		cfg.License = geoIp2License
		gm, err := geo.NewGeoManager(cfg)
		if err != nil {
			glog.Info(context.TODO(), "NewGeoManager error", glog.String("err", err.Error()))
			return
		}

		defaultGeoMgr = gm
		registerAdmin()
	})
	if err != nil {
		return nil, err
	}

	return defaultGeoMgr, nil
}

func registerAdmin() {
	if defaultGeoMgr == nil {
		return
	}
	// curl 'http://localhost:6480/admin?cmd=geoQueryCache&params={{ip}}'
	gapp.RegisterAdmin("geoQueryCache", "query geo cache", OnQueryGeoCache)
	// curl 'http://localhost:6480/admin?cmd=geoClearCache&params={{ip}}' if ip == "",clear all cache
	gapp.RegisterAdmin("geoClearCache", "geoClearCache", OnClearGeoCache)
	// curl 'http://localhost:6480/admin?cmd=geoQuery&params={{ip}}'
	gapp.RegisterAdmin("geoQuery", "geoQuery ", OnQueryGeo)
}

func OnQueryGeoCache(args gapp.AdminArgs) (interface{}, error) {
	ip := args.GetStringAt(0)
	val := defaultGeoMgr.QueryCache(context.Background(), ip)
	return val, nil
}

func OnClearGeoCache(args gapp.AdminArgs) (interface{}, error) {
	ip := args.GetStringAt(0)
	defaultGeoMgr.ClearCache(context.Background(), ip)
	return "success", nil
}

func OnQueryGeo(args gapp.AdminArgs) (interface{}, error) {
	ip := args.GetStringAt(0)
	return defaultGeoMgr.QueryCityAndCountry(context.Background(), ip)
}
