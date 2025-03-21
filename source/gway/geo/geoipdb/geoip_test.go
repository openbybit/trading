package geoipdb

import (
	"os"
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/filesystem"
	"github.com/smartystreets/goconvey/convey"
)

func TestGeoIP_City(t *testing.T) {
	convey.Convey("New", t, func() {
		path := os.Getenv("GEOIP_PATH")
		if path == "" {
			path = "../data/geoip/" // brew install geoipupdate
		}

		gip, err := New(path, "R7pQu5jmBBa6hXal", WithTimeout(time.Second),
			WithAutoUpdate(true, 10*time.Second))
		convey.So(err, convey.ShouldBeNil)

		now := time.Now()
		checkDb(path)
		t.Log(time.Since(now))

		g, err := gip.GetCityInfo("85.97.6.42")
		convey.So(err, convey.ShouldBeNil)
		convey.So(g.GetGeoNameID(), convey.ShouldEqual, uint(745044))
		convey.So(g.GetNames()["en"], convey.ShouldEqual, "Istanbul")
		for _, s := range g.GetSubdivisions() {
			t.Log(s.GetGeoNameID(), s.GetNames(), s.GetIsoCode())
		}

		g, err = gip.GetCityInfo("1.16.0.2")
		convey.So(err, convey.ShouldBeNil)
		t.Log(g.GetCountryNames(), g.GetNames())

		g, err = gip.GetCityInfo("13.212.13.15")
		convey.So(err, convey.ShouldBeNil)
		t.Log(g.GetCountryNames(), g.GetNames(), g.GetCountryGeoNameID(), g.GetISO(), g.GetGeoNameID(),
			g.IsInEuropeanUnion(), g.GetSubdivisions())

		c, err := gip.GetCountryInfo("88.229.137.10")
		convey.So(err, convey.ShouldBeNil)
		convey.So(c.GetGeoNameID(), convey.ShouldEqual, uint(298795))
		convey.So(c.GetNames()["en"], convey.ShouldEqual, "Turkey")
		t.Log(c.GetISO(), c.GetNames(), c.GetGeoNameID(), c.IsInEuropeanUnion())

		err = gip.Close(cityLiteDataBase)
		convey.So(err, convey.ShouldBeNil)
		err = gip.CloseAll()
		convey.So(err, convey.ShouldBeNil)
	})
}

func checkDb(dir string) {
	for {
		fs, _ := filesystem.GetFilesInDir(dir, "mmdb")
		if len(fs) == 2 {
			break
		}
		time.Sleep(time.Second)
	}
}
