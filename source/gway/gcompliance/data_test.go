package gcompliance

import (
	"testing"

	compliance "code.bydev.io/cht/customer/kyc-stub.git/pkg/bybit/compliancewall/strategy/v1"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cache/lru"
)

func TestComplianceStrategyMatch(t *testing.T) {
	Convey("test complianceStrategy match", t, func() {
		cs := &complianceStrategy{}
		cs.strategies = map[string]map[string]map[string]*config{"creat_order": {"CN": {"login": &config{EndpointExec: "kyc"}}}}

		res, ok := cs.Match("creat_order", "CN", "login", "")
		So(ok, ShouldBeTrue)
		So(res, ShouldNotBeNil)
		So(res.EndpointExec, ShouldEqual, "kyc")

		res, ok = cs.Match("creat_order_2", "CN", "login", "")
		So(ok, ShouldBeFalse)
		So(res, ShouldBeNil)

		res, ok = cs.Match("creat_order", "EN", "login", "")
		So(ok, ShouldBeFalse)
		So(res, ShouldBeNil)

		res, ok = cs.Match("creat_order", "CN", "guest", "")
		So(ok, ShouldBeFalse)
		So(res, ShouldBeNil)
	})
}

func TestComplianceStrategy_Update(t *testing.T) {
	Convey("test complianceStrategy update and exist", t, func() {
		cs := newComplianceStrategy()
		cs.Update(map[string]map[string]map[string]*config{"creat_order": {"CN": {"login": &config{EndpointExec: "kyc"}}}})
		So(len(cs.strategies), ShouldEqual, 1)

		ok := cs.Exist("creat_order")
		So(ok, ShouldBeTrue)
	})
}

func TestBlackList_Update(t *testing.T) {
	Convey("test blackList update", t, func() {
		bl := newBlackList()
		bl.Set("bybit", "hkg")
		ok := bl.Contains("bybit")
		So(ok, ShouldBeTrue)
	})
}

var ui *UserInfo

// 240B
func BenchmarkUserinfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ui = &UserInfo{
			WhiteListStatus: false,
			KycStatus:       false,
			Country:         "HKG",
			KycLevel:        1,
			Groups: []*compliance.ComplianceUserItem{
				{
					Site:  BybitSiteID,
					Group: "groupA",
				},
				{
					Site:  "HKG",
					Group: "groupB",
				},
			},
		}
	}
}

var l lru.LRUCache

// about 77M
func BenchmarkLru(b *testing.B) {
	for i := 0; i < b.N; i++ {
		l, _ = lru.NewLRU(200000)
		for j := 0; j < 200000; j++ {
			l.Add(j, UserInfo{
				WhiteListStatus: false,
				KycStatus:       false,
				Country:         "HKG",
				KycLevel:        1,
				Groups: []*compliance.ComplianceUserItem{
					{
						Site:  BybitSiteID,
						Group: "groupA",
					},
					{
						Site:  "HKG",
						Group: "groupB",
					},
				},
			})
		}
	}
}
