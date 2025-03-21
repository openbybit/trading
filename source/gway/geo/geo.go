package geo

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/coocood/freecache"
	jsoniter "github.com/json-iterator/go"

	"code.bydev.io/fbu/gateway/gway.git/geo/geoipdb"
	"code.bydev.io/fbu/gateway/gway.git/geo/geonamedb"
)

const (
	defaultCacheSize               = 120 * 1024 * 1024
	cacheExpireSeconds             = 48 * 3600
	incompleteIpCacheExpireSeconds = 24 * 3600
)

var (
	once              sync.Once
	defaultGeoManager *geoManager
)

//go:generate mockgen -source=geo.go -destination=geo_mock.go -package=geo
type GeoManager interface {
	QueryCityAndCountry(ctx context.Context, ip string) (GeoData, error)
	QueryCache(ctx context.Context, ip string) string
	ClearCache(ctx context.Context, ip string)
}

type geoManager struct {
	geoIP    geoipdb.GeoIP
	geoNames geonamedb.GeoName
	cache    *freecache.Cache
	dir      string
}

// NewGeoManager create geo manager
func NewGeoManager(cfg Config) (GeoManager, error) {
	if cfg.DbStorePath == "" {
		cfg.DbStorePath = "data/geoip"
	}
	var err error
	if defaultGeoManager == nil {
		once.Do(func() {
			gm := &geoManager{
				dir: cfg.DbStorePath,
			}
			gm.cache = freecache.NewCache(defaultCacheSize)
			gm.geoIP, err = geoipdb.New(cfg.DbStorePath, cfg.License,
				gm.cache,
				geoipdb.WithTimeout(cfg.UpdateTimeout),
				geoipdb.WithAutoUpdate(cfg.AutoUpdate, cfg.UpdatePeriod))
			if err != nil {
				return
			}
			if gm.geoNames, err = geonamedb.NewGeonames(cfg.DbStorePath); err != nil {
				return
			}
			defaultGeoManager = gm
		})
	}
	return defaultGeoManager, nil
}

// QueryCityAndCountry query city and country
func (g *geoManager) QueryCityAndCountry(ctx context.Context, ip string) (GeoData, error) {
	cacheKey := "metadataGeo" + ip
	value, err := g.cache.Get([]byte(cacheKey))
	if err == nil {
		data := geoData{}
		err = jsoniter.Unmarshal(value, &data)
		if err == nil {
			return data, nil
		}
	}
	data, err := g.query(ctx, ip)
	if err != nil {
		return geoData{}, err
	}
	if !data.HasCountryInfo() && !data.HasCityInfo() {
		return geoData{}, nil
	}

	expireSeconds := 0
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	if !data.HasCountryInfo() || !data.HasCityInfo() {
		expireSeconds = incompleteIpCacheExpireSeconds + r.Intn(7200)
	} else {
		expireSeconds = cacheExpireSeconds + r.Intn(14400)
	}
	d, err := jsoniter.Marshal(data)
	if err != nil {
		return nil, err
	}
	err = g.cache.Set([]byte(cacheKey), d, expireSeconds)
	if err != nil {
		log.Printf("QueryCityAndCountry key: %s, size: %d,set cache error: %s\n", cacheKey, len(d), err)
	}
	return data, nil
}

func (g *geoManager) query(ctx context.Context, ip string) (GeoData, error) {
	cityInfo, err := g.geoIP.GetCityInfo(ip)
	if err != nil || cityInfo == nil {
		return nil, fmt.Errorf("geoIP GetCityInfo %w", err)
	}
	data := geoData{
		HasCountry: cityInfo.GetCountryGeoNameID() > 0,
		HasCity:    cityInfo.GetGeoNameID() > 0,
	}
	if cityInfo.GetCountryGeoNameID() > 0 {
		c := country{
			GeoNameId: int64(cityInfo.GetCountryGeoNameID()),
			Iso:       cityInfo.GetISO(),
		}
		geoCountry, e := g.queryCountryInfo(ctx, cityInfo.GetCountryGeoNameID())
		if e == nil && geoCountry != nil {
			if geoCountry.GetGeoNameID() > 0 {
				c.GeoNameId = geoCountry.GetGeoNameID()
			}
			if geoCountry.GetISO() != "" {
				c.Iso = geoCountry.GetISO()
			}
			if geoCountry.GetISO3() != "" {
				c.Iso3 = geoCountry.GetISO3()
			}
			if geoCountry.GetCurrencyCode() != "" {
				c.CurrencyCode = geoCountry.GetCurrencyCode()
			}
		}
		data.Country = c
	}

	if cityInfo.GetGeoNameID() > 0 {
		c := city{
			GeoNameID:    int64(cityInfo.GetGeoNameID()),
			Name:         cityInfo.GetNames()["en"],
			Subdivisions: cityInfo.GetSubdivisions(),
		}
		geoCity, e := g.queryCityInfo(ctx, cityInfo.GetGeoNameID())
		if e != nil || geoCity == nil {
			data.City = c
			return data, nil
		}
		if c.Name == "" && geoCity.GetName() != "" {
			c.Name = geoCity.GetName()
		}
		if c.GeoNameID <= 0 && geoCity.GetGeoNameID() > 0 {
			c.GeoNameID = geoCity.GetGeoNameID()
		}
		data.City = c
	}
	return data, nil
}

// queryCountryInfo query country info
func (g *geoManager) queryCountryInfo(ctx context.Context, id uint) (Country, error) {
	geoCountry, err := g.geoNames.QueryCountryByGeoNameID(int64(id))
	if err != nil {
		return nil, err
	}
	if geoCountry == nil || geoCountry.Iso == "" {
		return country{}, nil
	}

	return country{
		Iso:          geoCountry.Iso,
		Iso3:         geoCountry.Iso_3,
		GeoNameId:    geoCountry.GeoNameId,
		CurrencyCode: geoCountry.CurrencyCode,
	}, nil
}

// queryCityInfo query city info
func (g *geoManager) queryCityInfo(ctx context.Context, id uint) (City, error) {
	geoCity, err := g.geoNames.QueryCityByGeoNameID(int64(id))
	if err != nil {
		return nil, err
	}
	if geoCity == nil || geoCity.GeoNameId <= 0 {
		return city{}, nil
	}

	return city{
		GeoNameID: geoCity.GeoNameId,
		Name:      geoCity.Name,
	}, nil
}

func (g *geoManager) QueryCache(ctx context.Context, ip string) string {
	cacheKey := "metadataGeo" + ip
	value, err := g.cache.Get([]byte(cacheKey))
	if err == nil {
		return string(value)
	}
	return ""
}

// ClearCache if ip == "", clear all cache
func (g *geoManager) ClearCache(ctx context.Context, ip string) {
	if ip == "" {
		g.cache.Clear()
	}
	cacheKey := "metadataGeo" + ip
	_ = g.cache.Del([]byte(cacheKey))
}
