package dns

import (
	"github.com/agiledragon/gomonkey/v2"
	"github.com/miekg/dns"
	"github.com/smartystreets/goconvey/convey"
	"net"
	"reflect"
	"testing"
)

func TestAddressString(t *testing.T) {
	convey.Convey("TestAddressString", t, func() {
		addr := Address{
			Address: "foo",
			Port:    9999,
		}
		convey.So(addr.String(), convey.ShouldEqual, "foo:9999")
	})
}

func TestNewLookup(t *testing.T) {
	convey.Convey("TestNewLookup", t, func() {
		lookup := NewLookup("foo:9999")
		convey.So(lookup, convey.ShouldNotBeNil)
		convey.So(lookup.serverString, convey.ShouldEqual, "foo:9999")
	})
}

func TestLookupSRV(t *testing.T) {
	convey.Convey("TestLookupSRV", t, func() {
		lup := NewLookup("foo:9999")
		_, err := lup.LookupSRV("foo")
		convey.So(err, convey.ShouldNotBeNil)

		// 对lookupType方法进行打桩
		patch := gomonkey.ApplyPrivateMethod(reflect.TypeOf(lup), "lookupType", func(_ *lookup, _ string, _ string) (*dns.Msg, error) {
			msg := &dns.Msg{}
			msg.Answer = []dns.RR{&dns.SRV{Hdr: dns.RR_Header{Name: "test", Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: 0, Rdlength: 0}, Priority: 1, Weight: 1, Port: 8080, Target: "test"}}
			return msg, nil
		})
		defer patch.Reset()

		srves, err := lup.LookupSRV("foo")
		convey.So(err, convey.ShouldBeNil)
		convey.So(len(srves), convey.ShouldEqual, 1)
	})
}

func TestLookupA(t *testing.T) {
	convey.Convey("TestLookupA", t, func() {
		lup := NewLookup("foo:9999")
		_, err := lup.LookupA("foo")
		convey.So(err, convey.ShouldNotBeNil)

		// 对lookupType方法进行打桩
		patch := gomonkey.ApplyPrivateMethod(reflect.TypeOf(lup), "lookupType", func(_ *lookup, _ string, _ string) (*dns.Msg, error) {
			msg := &dns.Msg{}
			msg.Answer = []dns.RR{&dns.A{Hdr: dns.RR_Header{Name: "test", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0, Rdlength: 0}, A: net.ParseIP("")}}
			return msg, nil
		})
		defer patch.Reset()
		_, err = lup.LookupA("foo")
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestLookup_ParseAAnswer(t *testing.T) {
	convey.Convey("TestLookup_ParseAAnswer", t, func() {
		lookup := NewLookup("foo:9999")
		answer, err := lookup.parseAAnswer(&dns.Msg{})
		convey.So(answer, convey.ShouldEqual, "")
		convey.So(err, convey.ShouldNotBeNil)

		answer, err = lookup.parseAAnswer(&dns.Msg{Answer: []dns.RR{&dns.A{Hdr: dns.RR_Header{Name: "test", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0, Rdlength: 0}, A: net.ParseIP("")}}})
		convey.So(err, convey.ShouldBeNil)
		convey.So(answer, convey.ShouldNotBeNil)

		answer, err = lookup.parseAAnswer(&dns.Msg{Answer: []dns.RR{&dns.AVC{}}})
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(answer, convey.ShouldBeEmpty)

	})
}
