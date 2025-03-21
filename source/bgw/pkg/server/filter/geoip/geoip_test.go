package geoip

import (
	"context"
	"fmt"
	"testing"

	rgeo "code.bydev.io/fbu/gateway/gway.git/geo"
	"code.bydev.io/fbu/gateway/gway.git/geo/geoipdb"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/geoip"
)

var mockErr = fmt.Errorf("mock err")

func TestGeo_Do(t *testing.T) {
	Convey("test geo do", t, func() {
		Init()
		_ = newGeo()
		g := &geo{}
		n := g.GetName()
		err := g.Init(context.Background())
		So(err, ShouldBeNil)
		So(n, ShouldEqual, filter.GEOFilterKey)

		next := func(ctx *types.Ctx) error {
			return nil
		}
		handler := g.Do(next)

		ctx := &types.Ctx{}
		patch := gomonkey.ApplyFunc(geoip.NewGeoManager, func() (rgeo.GeoManager, error) {
			return nil, mockErr
		})
		err = g.Init(context.Background(), "geo")
		So(err, ShouldNotBeNil)
		err = handler(ctx)
		So(err, ShouldEqual, mockErr)
		patch.Reset()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockMgr := rgeo.NewMockGeoManager(ctrl)
		mockMgr.EXPECT().QueryCityAndCountry(gomock.Any(), "127.0.0.1").Return(nil, mockErr)
		mockMgr.EXPECT().QueryCityAndCountry(gomock.Any(), "127.0.0.2").Return(&mockGeo{}, nil)

		patch = gomonkey.ApplyFunc(geoip.NewGeoManager, func() (rgeo.GeoManager, error) {
			return mockMgr, nil
		})
		defer patch.Reset()

		err = g.Init(context.Background(), "geo")
		So(err, ShouldNotBeNil)

		md := metadata.MDFromContext(ctx)
		md.Extension.RemoteIP = "127.0.0.1"
		err = handler(ctx)
		So(err, ShouldBeNil)

		g.flags.metadataCountry.needTransfer = true
		g.flags.metadataCity.needTransfer = true
		md.Extension.RemoteIP = "127.0.0.2"
		err = handler(ctx)
		So(err, ShouldBeNil)
	})
}

type mockGeo struct{}

func (m *mockGeo) HasCountryInfo() bool {
	return true
}
func (m *mockGeo) HasCityInfo() bool {
	return true
}
func (m *mockGeo) GetCountry() rgeo.Country {
	return &mockCountry{}
}
func (m *mockGeo) GetCity() rgeo.City {
	return nil
}

type mockCountry struct{}

func (m *mockCountry) GetGeoNameID() int64 {
	return 0
}
func (m *mockCountry) GetISO() string {
	return "CN"
}
func (m *mockCountry) GetISO3() string {
	return ""
}
func (m *mockCountry) GetCurrencyCode() string {
	return ""
}

func TestGeo_parse(t *testing.T) {
	Convey("test geo parse", t, func() {
		g := &geo{}

		err := g.parse(context.Background(), []string{"geo", "--wrongAgrs=1"})
		So(err, ShouldNotBeNil)

		patch := gomonkey.ApplyFunc(geoip.InitIPWhitelist, func(ctx context.Context) error { return nil })
		defer patch.Reset()
		err = g.parse(context.Background(), []string{"geo", "--countries=CN,KR", "--metadata={\"country\":[\"iso\",\"iso3\",\"currencyCode\",\"geonameid\"],\"city\":[\"name\",\"geonameid\"]}"})
		So(err, ShouldBeNil)
	})
}

func TestGeo_setCountry(t *testing.T) {
	Convey("test geo set country", t, func() {
		g := &geo{}
		g.flags.metadataCountry.iso = true
		g.flags.metadataCountry.iso3 = true
		g.flags.metadataCountry.fiat = true
		g.flags.metadataCountry.geoID = true

		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)
		g.setCountry(ctx, &mockGeo{}, md)

	})
}

func TestGeo_BannedCountry(t *testing.T) {
	Convey("test bannedCountry", t, func() {
		g := &geo{}
		err := g.bannedCountry(context.Background(), "", "")
		So(err, ShouldBeNil)

		gomonkey.ApplyFunc(geoip.CheckIPWhitelist, func(ctx context.Context, ip string) bool {
			if ip == "127.0.0.1" {
				return true
			}

			return false
		})
		err = g.bannedCountry(context.Background(), "CN", "127.0.0.1")
		So(err, ShouldBeNil)

		err = g.bannedCountry(context.Background(), "CN", "127.0.0.2")
		So(err, ShouldBeNil)

		g.flags.bannedCountries = "CN"
		err = g.bannedCountry(context.Background(), "CN", "127.0.0.2")
		So(err, ShouldNotBeNil)
	})
}

func TestGeo_SetCity(t *testing.T) {
	Convey("test setCity", t, func() {
		g := &geo{}
		g.flags.metadataCity.name = true
		g.flags.metadataCity.geoID = true

		g.setCity(&types.Ctx{}, &mockGeoData{}, &metadata.Metadata{})
	})
}

type mockGeoData struct{}

func (m *mockGeoData) HasCountryInfo() bool {
	return true
}

func (m *mockGeoData) HasCityInfo() bool {
	return true
}

func (m *mockGeoData) GetCountry() rgeo.Country {
	return nil
}

func (m *mockGeoData) GetCity() rgeo.City {
	return &mockCity{}
}

type mockCity struct{}

func (m *mockCity) GetGeoNameID() int64 {
	return 0
}
func (m *mockCity) GetName() string {
	return ""
}
func (m *mockCity) GetSubdivisions() []geoipdb.Subdivisions {
	return nil
}
func (m *mockCity) GetSubdivision() geoipdb.Subdivisions {
	return geoipdb.Subdivisions{}
}
