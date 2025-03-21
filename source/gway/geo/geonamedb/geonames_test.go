package geonamedb

import (
	"os"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestGeoNames(t *testing.T) {
	convey.Convey("NewGeonames", t, func() {
		dir := os.Getenv("GEOIP_PATH")
		if dir == "" {
			dir = "../data/geoip/" // brew install geoipupdate
		}
		g, err := NewGeonames(dir)
		convey.So(err, convey.ShouldBeNil)
		convey.So(g, convey.ShouldNotBeNil)

		gr, err := g.QueryCityByGeoNameID(745044)
		convey.So(err, convey.ShouldBeNil)
		convey.ShouldEqual(gr.Name, "Istanbul")
		t.Log(gr.Name)
		gc, err := g.QueryCountryByGeoNameID(298795)
		convey.So(err, convey.ShouldBeNil)
		convey.ShouldEqual(gc.Country, "Turkey")
		t.Log(gc.Country)

		rst, err := g.QueryByGeoNameID(745044)
		convey.So(err, convey.ShouldBeNil)
		convey.ShouldEqual(rst.Name(), "Istanbul")
		t.Log(rst.String())

		_, err = g.QueryByGeoNameID(7450441111)
		convey.ShouldEqual(err, ErrNotFound)

		rst, err = g.QueryByGeoNameID(298795)
		convey.So(err, convey.ShouldBeNil)
		convey.ShouldEqual(rst.HasCity(), false)
		convey.ShouldEqual(rst.City(), nil)
		convey.ShouldEqual(rst.Country().Iso, "TR")

		gc, err = g.QueryCountryByISO("TR")
		convey.So(err, convey.ShouldBeNil)
		convey.ShouldEqual(gc.Country, "Turkey")
	})
}
