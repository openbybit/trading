package geoip

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	rgeo "code.bydev.io/fbu/gateway/gway.git/geo"
	"code.bydev.io/fbu/gateway/gway.git/glog"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/geoip"
)

func Init() {
	filter.Register(filter.GEOFilterKey, newGeo) // route filter
}

type geo struct {
	flags geoRule
}

func newGeo() filter.Filter {
	return &geo{}
}

// GetName returns the name of the filter
func (g *geo) GetName() string {
	return filter.GEOFilterKey
}

func (g *geo) Do(next types.Handler) types.Handler {
	return func(ctx *types.Ctx) (err error) {
		var data rgeo.GeoData

		md := metadata.MDFromContext(ctx)
		ip := md.Extension.RemoteIP
		db, err := geoip.NewGeoManager()
		if err != nil {
			return err
		}
		data, err = db.QueryCityAndCountry(ctx, ip)
		if err != nil || data == nil {
			glog.Error(ctx, "geo query geoData no data", glog.Any("rule", g.flags), glog.String("ip", ip))
			return next(ctx)
		}

		if err = g.bannedCountry(ctx, data.GetCountry().GetISO(), ip); err != nil {
			return
		}

		if g.flags.metadataCountry.needTransfer {
			g.setCountry(ctx, data, md)
		}

		if g.flags.metadataCity.needTransfer {
			g.setCity(ctx, data, md)
		}

		glog.Debug(ctx, "biz geo check pass", glog.String("ip", ip))
		return next(ctx)
	}
}

// Init remoteGeo flag parse ( --countries=CN,KR --metadata={"country":["iso","iso3","currencyCode","geonameid"],"city":["name","geonameid"]} )
func (g *geo) Init(ctx context.Context, args ...string) (err error) {
	if len(args) == 0 {
		return
	}

	gm, err := geoip.NewGeoManager()
	if err != nil || gm == nil {
		return fmt.Errorf("NewGeoManager error:%w", err)
	}

	err = g.parse(ctx, args)
	return
}

func (g *geo) parse(ctx context.Context, args []string) error {
	var (
		f             geoRule
		meta          geoMetadata
		banFlag       string
		countriesFlag string
	)

	parse := flag.NewFlagSet("geo", flag.ContinueOnError)
	parse.StringVar(&banFlag, "countries", "", "banned country list")
	parse.StringVar(&countriesFlag, "metadata", "", "transfer medata info list, json data")

	if err := parse.Parse(args[1:]); err != nil {
		return err
	}

	f.bannedCountries = strings.ToUpper(strings.TrimSpace(banFlag))
	if f.bannedCountries != "" {
		_ = geoip.InitIPWhitelist(context.Background())
	}
	if err := json.Unmarshal([]byte(countriesFlag), &meta); err != nil {
		glog.Error(ctx, "Unmarshal geoMetadata error", glog.String("error", err.Error()), glog.String("meta", countriesFlag))
		return berror.NewInterErr("geo param Unmarshal error", err.Error())
	}

	for _, v := range meta.Country {
		info := strings.ToLower(strings.TrimSpace(v))
		switch countryInfo(info) {
		case countryIso:
			f.metadataCountry.iso = true
		case countryIso3:
			f.metadataCountry.iso3 = true
		case countryFiat:
			f.metadataCountry.fiat = true
		case countryGeoID:
			f.metadataCountry.geoID = true
		}
	}

	if f.metadataCountry.IsOpen() {
		f.metadataCountry.needTransfer = true
	}

	for _, v := range meta.City {
		info := strings.ToLower(strings.TrimSpace(v))
		switch cityInfo(info) {
		case cityName:
			f.metadataCity.name = true
		case cityGeoID:
			f.metadataCity.geoID = true
		}
	}

	if f.metadataCity.IsOpen() {
		f.metadataCity.needTransfer = true
	}

	g.flags = f
	return nil
}

func (g *geo) setCountry(c *types.Ctx, data rgeo.GeoData, md *metadata.Metadata) {
	if g.flags.metadataCountry.iso {
		md.Extension.CountryISO = data.GetCountry().GetISO()
	}

	if g.flags.metadataCountry.iso3 {
		md.Extension.CountryISO3 = data.GetCountry().GetISO3()
	}

	if g.flags.metadataCountry.fiat {
		md.Extension.CountryGeoNameID = data.GetCountry().GetGeoNameID()
	}

	if g.flags.metadataCountry.geoID {
		md.Extension.CurrencyCode = data.GetCountry().GetCurrencyCode()
	}

	md.WithContext(c)
}

func (g *geo) setCity(c *types.Ctx, data rgeo.GeoData, md *metadata.Metadata) {
	if g.flags.metadataCity.name {
		md.Extension.CityName = util.Base64Encode(data.GetCity().GetName())
	}

	if g.flags.metadataCity.geoID {
		sub := data.GetCity().GetSubdivision()
		md.Extension.SubVisionId = int64(sub.GetGeoNameID())
		md.Extension.CityGeoNameID = data.GetCity().GetGeoNameID()
	}

	md.WithContext(c)
}

func (g *geo) bannedCountry(ctx context.Context, iso string, ip string) (err error) {
	if iso == "" {
		glog.Debug(ctx, "banned remoteGeo query geoData is nil", glog.String("ip", ip))
		return nil
	}

	if geoip.CheckIPWhitelist(ctx, ip) {
		return nil
	}

	if strings.Contains(g.flags.bannedCountries, iso) {
		return berror.ErrCountryBanned
	}

	return nil
}
