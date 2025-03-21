package geoip

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMetadataCountry_IsOpen(t *testing.T) {
	Convey("test is open", t, func() {
		m := &metadataCountry{}
		o := m.IsOpen()
		So(o, ShouldBeFalse)

		m = &metadataCountry{iso: true}
		o = m.IsOpen()
		So(o, ShouldBeTrue)

		m = &metadataCountry{iso3: true}
		o = m.IsOpen()
		So(o, ShouldBeTrue)

		m = &metadataCountry{fiat: true}
		o = m.IsOpen()
		So(o, ShouldBeTrue)

		m = &metadataCountry{geoID: true}
		o = m.IsOpen()
		So(o, ShouldBeTrue)

		n := &metadataCity{}
		o = n.IsOpen()
		So(o, ShouldBeFalse)

		n = &metadataCity{geoID: true}
		o = n.IsOpen()
		So(o, ShouldBeTrue)

		n = &metadataCity{name: true}
		o = n.IsOpen()
		So(o, ShouldBeTrue)
	})
}
