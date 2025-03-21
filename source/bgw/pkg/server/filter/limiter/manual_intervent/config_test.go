package manual_intervent

import (
	"errors"
	"net/url"
	"strings"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
)

func TestInterventionRule_checkExtData(t *testing.T) {
	convey.Convey("TestInterventionRule_checkExtData", t, func() {
		r := rule{}
		err := r.checkExtData()
		convey.So(err, convey.ShouldNotBeNil)

		r.ExtData = []*extData{}
		err = r.checkExtData()
		convey.So(err, convey.ShouldNotBeNil)

		r.RuleType = "unknown"
		err = r.checkExtData()
		convey.So(err, convey.ShouldNotBeNil)

		r = rule{
			RuleType: clientIpRuleType,
			ExtData:  []*extData{{ClientIp: "127.0.0.1/24"}},
		}
		err = r.checkExtData()
		convey.So(err, convey.ShouldNotBeNil)

		r.ExtData = []*extData{{ClientIp: "127.0.0.1"}}
		err = r.checkExtData()
		convey.So(err, convey.ShouldBeNil)

		r = rule{
			RuleType: requestHostRuleType,
			ExtData:  []*extData{{RequestHost: "~~~"}},
		}
		err = r.checkExtData()
		convey.So(err, convey.ShouldNotBeNil)

		r.ExtData = []*extData{{RequestHost: "www.baidu.com"}}
		err = r.checkExtData()
		convey.So(err, convey.ShouldBeNil)

		r = rule{
			RuleType: clientOpFromRule,
			ExtData:  []*extData{{ClientOpFrom: ""}},
		}
		err = r.checkExtData()
		convey.So(err, convey.ShouldNotBeNil)
		r.ExtData = []*extData{{ClientOpFrom: "1234567890123456789012345678901234567890123456789012345678901234567890"}}
		err = r.checkExtData()
		convey.So(err, convey.ShouldNotBeNil)

		r.ExtData = []*extData{{ClientOpFrom: "pcweb"}}
		err = r.checkExtData()
		convey.So(err, convey.ShouldBeNil)

		r = rule{
			RuleType: requestUrlRule,
			ExtData:  []*extData{{RequestUrl: requestUrl{HttpMethod: "TEST", Path: "http://www.baidu.com", Limit: 1}}},
		}
		err = r.checkExtData()
		convey.So(err, convey.ShouldNotBeNil)

		applyFunc := gomonkey.ApplyFunc(url.Parse, func(rawURL string) (*url.URL, error) {
			return nil, errors.New("mock")
		})
		r.ExtData = []*extData{{RequestUrl: requestUrl{HttpMethod: "GET", Path: "example.com:8080/path?name=John", Limit: 1}}}
		err = r.checkExtData()
		convey.So(err, convey.ShouldNotBeNil)
		applyFunc.Reset()

		r.ExtData = []*extData{{RequestUrl: requestUrl{HttpMethod: "GET", Path: "http://www.baidu.com", Limit: -1}}}
		err = r.checkExtData()
		convey.So(err, convey.ShouldNotBeNil)

		r.ExtData = []*extData{{RequestUrl: requestUrl{HttpMethod: "GET", Path: "/v3/order/create", Limit: 1}}}
		err = r.checkExtData()
		convey.So(err, convey.ShouldBeNil)

		r.ExtData = []*extData{{RequestUrl: requestUrl{HttpMethod: "*", Path: "/v3/order/create", Limit: 1}}}
		err = r.checkExtData()
		convey.So(err, convey.ShouldBeNil)

		r.ExtData = []*extData{{RequestUrl: requestUrl{HttpMethod: "GET", Path: "/v3/order/*", Limit: 1}}}
		err = r.checkExtData()
		convey.So(err, convey.ShouldBeNil)

		r = rule{
			RuleType: "illegal",
			ExtData:  []*extData{{RequestUrl: requestUrl{HttpMethod: "TEST", Path: "http://www.baidu.com", Limit: 1}}},
		}
		err = r.checkExtData()
		convey.So(err, convey.ShouldNotBeNil)
	})
}

func TestInterventionRule_validateEffectiveEnvName(t *testing.T) {
	convey.Convey("TestInterventionRule_validateEffectiveEnvName", t, func() {
		patches := gomonkey.ApplyFunc(env.ProjectEnvName, func() string { return "test" })
		defer patches.Reset()

		rule := rule{}
		matchEffectiveEnvName := rule.matchEffectiveEnvName()
		convey.So(matchEffectiveEnvName, convey.ShouldBeFalse)

		applyFunc := gomonkey.ApplyFunc(strings.Split, func(s, sep string) []string {
			return nil
		})
		matchEffectiveEnvName = rule.matchEffectiveEnvName()
		convey.So(matchEffectiveEnvName, convey.ShouldBeFalse)
		applyFunc.Reset()

		rule.EffectiveEnvName = "test"
		matchEffectiveEnvName = rule.matchEffectiveEnvName()
		convey.So(matchEffectiveEnvName, convey.ShouldBeTrue)

		rule.EffectiveEnvName = "unify-test-1,bybit-test-1"
		matchEffectiveEnvName = rule.matchEffectiveEnvName()
		convey.So(matchEffectiveEnvName, convey.ShouldBeFalse)
	})
}

func TestStandardPeriod_isValid(t *testing.T) {
	convey.Convey("TestStandardPeriod_isValid", t, func() {
		period := StandardPeriod{}
		err := period.isValid()
		convey.So(err, convey.ShouldNotBeNil)

		period = StandardPeriod{StartDateInUTC: "2023-10-17 00:00:00", EndDateInUTC: "00:00:00"}
		err = period.isValid()
		convey.So(err, convey.ShouldNotBeNil)

		period = StandardPeriod{StartDateInUTC: "2023-10-17 00:00:00", EndDateInUTC: "2023-10-15 00:00:00"}
		err = period.isValid()
		convey.So(err, convey.ShouldBeNil)

		period = StandardPeriod{StartDateInUTC: "2023-10-17 00:00:00", EndDateInUTC: "2026-10-18 00:00:00"}
		err = period.isValid()
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestStandardPeriod_inPeriod(t *testing.T) {
	convey.Convey("TestStandardPeriod_inPeriod", t, func() {

		period := StandardPeriod{StartDateInUTC: "2023-10-17 ", EndDateInUTC: "2026-10-18 00:00:00"}
		inPeriod := period.inPeriod()
		convey.So(inPeriod, convey.ShouldBeFalse)

		period = StandardPeriod{StartDateInUTC: "2023-10-17 00:00:00", EndDateInUTC: "2026-10-18"}
		inPeriod = period.inPeriod()
		convey.So(inPeriod, convey.ShouldBeFalse)

		period = StandardPeriod{StartDateInUTC: "2023-10-17 00:00:00", EndDateInUTC: "2026-10-18 00:00:00"}
		inPeriod = period.inPeriod()
		convey.So(inPeriod, convey.ShouldBeTrue)

		period = StandardPeriod{StartDateInUTC: "2023-10-17 00:00:00", EndDateInUTC: "2023-10-18 00:00:00"}
		inPeriod = period.inPeriod()
		convey.So(inPeriod, convey.ShouldBeFalse)
	})
}
