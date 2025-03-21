package geoipdb

import (
	"os"
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/filesystem"
	"github.com/smartystreets/goconvey/convey"
)

func TestGeo(t *testing.T) {
	convey.Convey("etl", t, func() {
		dir := "data/geoip"
		geo, err := New(dir, "R7pQu5jmBBa6hXal", WithTimeout(time.Second),
			WithAutoUpdate(true, 10*time.Second))
		convey.So(err, convey.ShouldBeNil)
		convey.So(geo, convey.ShouldNotBeNil)

		now := time.Now()
		checkDb(dir)
		t.Log(time.Since(now))
		time.Sleep(30 * time.Second)
	})
}

func TestDownload(t *testing.T) {
	convey.Convey("etl", t, func() {
		dir := "data/geoip"
		err := filesystem.MkdirAll(dir)
		convey.So(err, convey.ShouldBeNil)
		d := newDownloader("R7pQu5jmBBa6hXal", dir, time.Second)

		filename, tarName, err := d.Do(cityPrefix)
		convey.So(err, convey.ShouldBeNil)
		t.Log(filename, tarName)
		_ = os.RemoveAll("data")
	})
}
