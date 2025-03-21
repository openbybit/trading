package geo

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/filesystem"
	"code.bydev.io/fbu/gateway/gway.git/geo/geoipdb"
	"code.bydev.io/fbu/gateway/gway.git/geo/geonamedb"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/coocood/freecache"
	"github.com/golang/mock/gomock"
	jsoniter "github.com/json-iterator/go"
	"github.com/smartystreets/goconvey/convey"
)

func TestGeo(t *testing.T) {
	convey.Convey("NewGeoManager", t, func() {
		cfg := Config{
			License:       "R7pQu5jmBBa6hXal",
			UpdateTimeout: 10 * time.Second,
			UpdatePeriod:  10 * time.Minute,
			AutoUpdate:    true,
			DbStorePath:   "data/geoip/",
		}
		g, err := NewGeoManager(cfg)
		convey.So(err, convey.ShouldBeNil)
		convey.So(g, convey.ShouldNotBeNil)

		gd, err := g.QueryCityAndCountry(context.TODO(), "95.183.78.43")
		convey.So(err, convey.ShouldBeNil)
		t.Log(gd.HasCityInfo(), gd.HasCountryInfo(), gd.GetCity(), gd.GetCountry(), gd.GetCity().GetSubdivisions())
		convey.ShouldEqual(len(gd.GetCity().GetSubdivisions()), 1)

		gd, err = g.QueryCityAndCountry(context.TODO(), "85.97.6.42")
		convey.So(err, convey.ShouldBeNil)
		convey.ShouldEqual(len(gd.GetCity().GetSubdivisions()), 1)
	})
}

func TestNewGeoManager(t *testing.T) {
	convey.Convey("NewGeoManager", t, func() {
		convey.Convey("MkdirAll error", func() {
			cfg := Config{}
			patch := gomonkey.ApplyFunc(filesystem.MkdirAll, func(path string) error {
				return errors.New("mkdirall error")
			})
			defer patch.Reset()
			manager, err := NewGeoManager(cfg)
			convey.So(err, convey.ShouldResemble, errors.New("mkdirall error"))
			convey.So(manager, convey.ShouldBeNil)
		})

		convey.Convey("MkdirAll no error", func() {
			patch := gomonkey.ApplyFunc(filesystem.MkdirAll, func(path string) error {
				return nil
			})
			defer patch.Reset()
			manager, err := NewGeoManager(Config{})
			convey.So(err, convey.ShouldBeNil)
			convey.So(manager, convey.ShouldNotBeNil)
		})
	})
}

func TestQueryCityInfo(t *testing.T) {
	convey.Convey("QueryCityInfo", t, func() {
		ctrl := gomock.NewController(t)
		convey.Convey("QueryCityByGeonameID1", func() {
			geonameDb := geonamedb.NewMockGeoName(ctrl)
			geonameDb.EXPECT().QueryCityByGeoNameID(gomock.Any()).Return(&geonamedb.GeoNameCity{Name: "beijing", GeoNameId: 123456}, nil)
			patch := gomonkey.ApplyFunc(NewGeoManager, func(ctx context.Context, cfg Config) (GeoManager, error) {
				gm := &geoManager{
					dir: cfg.DbStorePath,
				}
				var err error
				gm.cache = freecache.NewCache(defaultCacheSize)
				gm.geoIP, err = geoipdb.New(cfg.DbStorePath, cfg.License, freecache.NewCache(defaultCacheSize),
					geoipdb.WithTimeout(cfg.UpdateTimeout),
					geoipdb.WithAutoUpdate(cfg.AutoUpdate, cfg.UpdatePeriod))
				convey.So(err, convey.ShouldBeNil)
				gm.geoNames = geonameDb
				return gm, nil
			})
			defer patch.Reset()
			manager, err := NewGeoManager(Config{})
			convey.So(err, convey.ShouldBeNil)
			convey.So(manager, convey.ShouldNotBeNil)
		})
	})
}

func TestQueryCountryInfo(t *testing.T) {
	convey.Convey("QueryCountryInfo", t, func() {
		ctrl := gomock.NewController(t)
		convey.Convey("QueryCountryInfo1", func() {
			geonameDb := geonamedb.NewMockGeoName(ctrl)
			patch := gomonkey.ApplyFunc(NewGeoManager, func(cfg Config) (GeoManager, error) {
				gm := &geoManager{
					dir: cfg.DbStorePath,
				}
				gm.cache = freecache.NewCache(defaultCacheSize)
				var err error
				gm.geoIP, err = geoipdb.New(cfg.DbStorePath, cfg.License, freecache.NewCache(defaultCacheSize),
					geoipdb.WithTimeout(cfg.UpdateTimeout),
					geoipdb.WithAutoUpdate(cfg.AutoUpdate, cfg.UpdatePeriod))
				convey.So(err, convey.ShouldBeNil)
				gm.geoNames = geonameDb
				return gm, nil
			})
			defer patch.Reset()
			geonameDb.EXPECT().QueryCountryByGeoNameID(gomock.Any()).Return(&geonamedb.GeonameCountry{
				GeoNameId:    123456,
				Iso:          "iso",
				Iso_3:        "iso3",
				CurrencyCode: "currencyCode",
			}, nil)
			manager, err := NewGeoManager(Config{})
			convey.So(err, convey.ShouldBeNil)
			convey.So(manager, convey.ShouldNotBeNil)
		})
	})
}

func TestQuery(t *testing.T) {
	convey.Convey("Query", t, func() {
		ctrl := gomock.NewController(t)
		ctx := context.Background()
		geoIPDbCity := geoipdb.NewMockCity(ctrl)
		geoIP := geoipdb.NewMockGeoIP(ctrl)

		gm := geoManager{
			geoIP: geoIP,
		}
		convey.Convey("GetCityInfo", func() {
			geoIP.EXPECT().GetCityInfo(gomock.Any()).Return(nil, errors.New("get city info failed"))
			data, err := gm.query(ctx, "127.0.0.1")
			convey.So(err, convey.ShouldResemble, errors.New("get city info failed"))
			convey.So(data, convey.ShouldResemble, geoData{})
		})

		convey.Convey("citygeonameid==0, countrygeonameid==0", func() {
			geoIP.EXPECT().GetCityInfo(gomock.Any()).Return(geoIPDbCity, nil)
			geoIPDbCity.EXPECT().GetGeoNameID().Return(uint(0)).MaxTimes(2)
			geoIPDbCity.EXPECT().GetCountryGeoNameID().Return(uint(0)).MaxTimes(2)
			data, err := gm.query(ctx, "127.0.0.2")
			convey.So(err, convey.ShouldBeNil)
			convey.So(data, convey.ShouldResemble, geoData{})
		})
		convey.Convey("citygeonameid > 0, countrygeonameid == 0", func() {
			convey.Convey("QueryCityInfo: nil, err", func() {
				patch := gomonkey.ApplyMethod(&gm, "QueryCityInfo", func(_ *geoManager, ctx context.Context, id uint) (City, error) {
					return nil, errors.New("query city info failed")
				})
				defer patch.Reset()
				geoIP.EXPECT().GetCityInfo(gomock.Any()).Return(geoIPDbCity, nil)
				geoIPDbCity.EXPECT().GetGeoNameID().Return(uint(123)).MaxTimes(3)
				geoIPDbCity.EXPECT().GetCountryGeoNameID().Return(uint(0)).MaxTimes(2)
				data, err := gm.query(ctx, "127.0.0.3")
				convey.So(err, convey.ShouldResemble, errors.New("query city info failed"))
				convey.So(data, convey.ShouldResemble, geoData{})
			})

			convey.Convey("QueryCityInfo: nil, nil", func() {
				// nil, nil
				patchOne := gomonkey.ApplyMethod(&gm, "QueryCityInfo", func(_ *geoManager, ctx context.Context, id uint) (City, error) {
					return nil, nil
				})
				defer patchOne.Reset()
				patchTwo := gomonkey.ApplyMethod(&gm, "QueryCountryInfo", func(_ *geoManager, ctx context.Context, id uint) (Country, error) {
					return nil, nil
				})
				defer patchTwo.Reset()
				geoIP.EXPECT().GetCityInfo(gomock.Any()).Return(geoIPDbCity, nil)
				geoIPDbCity.EXPECT().GetGeoNameID().Return(uint(123)).MaxTimes(3)
				geoIPDbCity.EXPECT().GetCountryGeoNameID().Return(uint(0)).MaxTimes(2)
				data, err := gm.query(ctx, "127.0.0.4")
				convey.So(err, convey.ShouldBeNil)
				convey.So(data, convey.ShouldResemble, geoData{})
			})
			convey.Convey("QueryCityInfo: data, nil", func() {
				convey.Convey("name != nil && geoNameID > 0", func() {
					patch := gomonkey.ApplyMethod(&gm, "QueryCityInfo", func(_ *geoManager, ctx context.Context, id uint) (City, error) {
						return city{GeoNameID: 123456, Name: "beijing"}, nil
					})
					defer patch.Reset()
					geoIP.EXPECT().GetCityInfo(gomock.Any()).Return(geoIPDbCity, nil)
					geoIPDbCity.EXPECT().GetGeoNameID().Return(uint(123)).MaxTimes(3)
					geoIPDbCity.EXPECT().GetCountryGeoNameID().Return(uint(0)).MaxTimes(2)
					data, err := gm.query(ctx, "127.0.0.5")
					convey.So(err, convey.ShouldBeNil)
					convey.So(data, convey.ShouldResemble, geoData{
						HasCity: true,
						City:    city{GeoNameID: 123456, Name: "beijing"},
					})
				})

				convey.Convey("name == nil && geoNameID <= 0", func() {
					patch := gomonkey.ApplyMethod(&gm, "QueryCityInfo", func(_ *geoManager, ctx context.Context, id uint) (City, error) {
						return city{}, nil
					})
					defer patch.Reset()
					geoIP.EXPECT().GetCityInfo(gomock.Any()).Return(geoIPDbCity, nil)
					geoIPDbCity.EXPECT().GetGeoNameID().Return(uint(123)).MaxTimes(4)
					geoIPDbCity.EXPECT().GetNames().Return(map[string]string{"en": "shanghai"})
					geoIPDbCity.EXPECT().GetCountryGeoNameID().Return(uint(0)).MaxTimes(2)
					data, err := gm.query(ctx, "127.0.0.5")
					convey.So(err, convey.ShouldBeNil)
					convey.So(data, convey.ShouldResemble, geoData{
						HasCity: true,
						City:    city{GeoNameID: 123, Name: "shanghai"},
					})
				})

				convey.Convey("citygeonameid > 0, countrygeonameid > 0 && QueryCountryInfo: data, nil", func() {
					patchOne := gomonkey.ApplyMethod(&gm, "QueryCityInfo", func(_ *geoManager, ctx context.Context, id uint) (City, error) {
						return city{}, nil
					})
					defer patchOne.Reset()
					patchTwo := gomonkey.ApplyMethod(&gm, "QueryCountryInfo", func(_ *geoManager, ctx context.Context, id uint) (Country, error) {
						return country{}, nil
					})
					defer patchTwo.Reset()
					geoIP.EXPECT().GetCityInfo(gomock.Any()).Return(geoIPDbCity, nil)
					geoIPDbCity.EXPECT().GetGeoNameID().Return(uint(123)).MaxTimes(4)
					geoIPDbCity.EXPECT().GetNames().Return(map[string]string{"en": "shanghai"})
					geoIPDbCity.EXPECT().GetCountryGeoNameID().Return(uint(456)).MaxTimes(3)
					data, err := gm.query(ctx, "127.0.0.5")
					convey.So(err, convey.ShouldBeNil)
					convey.So(data, convey.ShouldResemble, geoData{
						HasCity:    true,
						HasCountry: true,
						City:       city{GeoNameID: 123, Name: "shanghai"},
						Country:    country{},
					})
				})

				convey.Convey("citygeonameid > 0, countrygeonameid > 0 && QueryCountryInfo: nil, err", func() {
					patch := gomonkey.ApplyMethod(&gm, "QueryCountryInfo", func(_ *geoManager, ctx context.Context, id uint) (Country, error) {
						return nil, errors.New("query country info failed")
					})
					defer patch.Reset()
					geoIP.EXPECT().GetCityInfo(gomock.Any()).Return(geoIPDbCity, nil)
					geoIPDbCity.EXPECT().GetGeoNameID().Return(uint(123)).MaxTimes(3)
					geoIPDbCity.EXPECT().GetCountryGeoNameID().Return(uint(456)).MaxTimes(3)
					data, err := gm.query(ctx, "127.0.0.5")
					convey.So(err, convey.ShouldResemble, errors.New("query country info failed"))
					convey.So(data, convey.ShouldResemble, geoData{})
				})
			})
		})
	})
}

func TestQueryCityAndCountry(t *testing.T) {
	convey.Convey("QueryCountryInfo", t, func() {
		ctx := context.Background()
		convey.Convey("cache hit", func() {
			cache := freecache.NewCache(defaultCacheSize)
			rawData := geoData{
				HasCountry: false,
				HasCity:    false,
			}
			data, _ := jsoniter.Marshal(rawData)
			_ = cache.Set([]byte("metadataGeo127.0.0.127"), data, 60)
			patch := gomonkey.ApplyFunc(NewGeoManager, func(ctx context.Context, cfg Config) (GeoManager, error) {
				gm := &geoManager{
					dir: cfg.DbStorePath,
				}
				gm.cache = cache
				return gm, nil
			})
			defer patch.Reset()
			manager, _ := NewGeoManager(Config{})
			queryData, err := manager.QueryCityAndCountry(ctx, "127.0.0.127")
			convey.So(err, convey.ShouldBeNil)
			convey.So(queryData, convey.ShouldResemble, geoData{
				HasCountry: false,
				HasCity:    false,
			})

			convey.Convey("query return err", func() {
				patch = gomonkey.ApplyPrivateMethod(reflect.TypeOf(manager), "query",
					func(_ *geoManager, ctx context.Context, ip string) (GeoData, error) {
						return geoData{}, errors.New("query failed")
					})
				queryData, err = manager.QueryCityAndCountry(ctx, "127.0.0.128")
				convey.So(err, convey.ShouldResemble, errors.New("query failed"))
				convey.So(queryData, convey.ShouldResemble, geoData{})
			})

			convey.Convey("query return geoData1", func() {
				patch = gomonkey.ApplyPrivateMethod(reflect.TypeOf(manager), "query",
					func(_ *geoManager, ctx context.Context, ip string) (GeoData, error) {
						return geoData{
							HasCountry: false,
							HasCity:    false,
						}, nil
					})
				queryData, err = manager.QueryCityAndCountry(ctx, "127.0.0.129")
				convey.So(err, convey.ShouldBeNil)
				convey.So(queryData, convey.ShouldResemble, geoData{})
			})

			convey.Convey("query return geoData2", func() {
				patch = gomonkey.ApplyPrivateMethod(reflect.TypeOf(manager), "query",
					func(_ *geoManager, ctx context.Context, ip string) (GeoData, error) {
						return geoData{
							HasCountry: true,
							HasCity:    false,
							Country: country{
								Iso: "iso",
							},
						}, nil
					})
				queryData, err = manager.QueryCityAndCountry(ctx, "127.0.0.130")
				convey.So(err, convey.ShouldBeNil)
				convey.So(queryData, convey.ShouldResemble, geoData{
					HasCountry: true,
					HasCity:    false,
					Country: country{
						Iso: "iso",
					}})
			})
		})
	})
}

func TestMarshal(t *testing.T) {
	convey.Convey("marshal", t, func() {
		convey.Convey("marshal", func() {
			data := geoData{
				HasCity:    false,
				HasCountry: false,
			}
			byteData, err := jsoniter.Marshal(data)
			convey.So(err, convey.ShouldBeNil)

			d := &geoData{}
			err = jsoniter.Unmarshal(byteData, d)
			convey.So(err, convey.ShouldBeNil)
			convey.So(*d, convey.ShouldResemble, data)
		})

		convey.Convey("marshal1", func() {
			data := geoData{
				HasCity:    true,
				HasCountry: false,
				City: city{
					GeoNameID: 123,
					Name:      "beijing",
				},
			}
			byteData, err := jsoniter.Marshal(data)
			convey.So(err, convey.ShouldBeNil)

			d := &geoData{}
			err = jsoniter.Unmarshal(byteData, d)
			convey.So(err, convey.ShouldBeNil)
			convey.So(*d, convey.ShouldResemble, data)
		})

		convey.Convey("marshal2", func() {
			data := geoData{
				HasCity:    true,
				HasCountry: true,
				City: city{
					GeoNameID: 123,
					Name:      "beijing",
				},
				Country: country{
					GeoNameId:    123456,
					Iso:          "iso",
					Iso3:         "iso3",
					CurrencyCode: "currencyCode",
				},
			}
			byteData, err := jsoniter.Marshal(data)
			convey.So(err, convey.ShouldBeNil)

			d := &geoData{}
			err = jsoniter.Unmarshal(byteData, d)
			convey.So(err, convey.ShouldBeNil)
			convey.So(*d, convey.ShouldResemble, data)
		})
	})
}
