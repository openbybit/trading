package gcompliance

import (
	"context"
	"errors"
	"strconv"
	"testing"

	compliance "code.bydev.io/cht/customer/kyc-stub.git/pkg/bybit/compliancewall/strategy/v1"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/atomic"
	"google.golang.org/grpc"

	"code.bydev.io/fbu/gateway/gway.git/gcompliance/mock"
	"code.bydev.io/fbu/gateway/gway.git/ggrpc/pool"
)

var mockErr = errors.New("mock err")

func Test_convert(t *testing.T) {
	Convey("test convert", t, func() {
		configs := &compliance.ComplianceConfigItem{}
		configs.SceneCode = "creat_order"
		for i := 0; i < 3; i++ {
			straCfg := &compliance.ComplianceConfigItem_StrategyConfig{}
			straCfg.CountryCode = "Country" + strconv.Itoa(i)
			for j := 0; j < 3; j++ {
				contryCfg := &compliance.ComplianceConfigItem_StrategyConfig_CountryConfig{}
				contryCfg.UserType = "usertype" + strconv.Itoa(j)
				contryCfg.DispositionResult = "result" + strconv.Itoa(j)
				straCfg.CountryConfig = append(straCfg.CountryConfig, contryCfg)
			}
			straCfg.CountryConfig = append(straCfg.CountryConfig, nil)
			configs.StrategyConfig = append(configs.StrategyConfig, straCfg)
		}
		configs.StrategyConfig = append(configs.StrategyConfig, nil)

		res := convert(configs, nil)
		So(len(res), ShouldEqual, 1)
	})
}

func Test_NewWall(t *testing.T) {
	Convey("test new wall", t, func() {

		sd := func(ctx context.Context, registry, namespace, group string) (addrs []string) {
			return []string{"127.0.0.1:6480"}
		}

		Convey("test registry", func() {
			rc := NewRegistryCfg("service_1", "ns", "g1", sd)
			w, err := NewWall(rc, true)
			So(err, ShouldBeNil)
			So(w, ShouldNotBeNil)

			rc = NewRegistryCfg("", "ns", "g1", sd)
			w, err = NewWall(rc, true)
			So(err, ShouldNotBeNil)
			So(w, ShouldBeNil)
		})

		Convey("test addr", func() {
			ac := NewAddrCfg("127.0.0.1:6480")
			w, err := NewWall(ac, true)
			So(err, ShouldBeNil)
			So(w, ShouldNotBeNil)

			ac = NewAddrCfg("")
			w, err = NewWall(ac, true)
			So(err, ShouldNotBeNil)
			So(w, ShouldBeNil)
		})

		Convey("test wrong type", func() {
			cfg := RemoteCfg(nil)
			w, err := NewWall(cfg, true)
			So(err, ShouldNotBeNil)
			So(w, ShouldBeNil)
		})
	})
}

func TestWall_GetServiceRoundRobin(t *testing.T) {
	Convey("test GetServiceRoundRobin", t, func() {

		Convey("test nil discovery", func() {
			w := &wall{
				discovery: nil,
			}

			_, err := w.GetServiceRoundRobin()
			So(err, ShouldNotBeNil)
		})

		Convey("test zero instance", func() {
			sd := func(ctx context.Context, registry, namespace, group string) (addrs []string) {
				return []string{}
			}

			w := &wall{
				discovery: sd,
				index:     atomic.NewInt32(0),
			}
			_, err := w.GetServiceRoundRobin()
			So(err, ShouldNotBeNil)
		})

		Convey("test RoundRobin", func() {
			sd := func(ctx context.Context, registry, namespace, group string) (addrs []string) {
				return []string{"addr1", "addr2", "addr3"}
			}

			w := &wall{
				discovery: sd,
				index:     atomic.NewInt32(0),
			}

			addr, err := w.GetServiceRoundRobin()
			So(err, ShouldBeNil)
			So(addr, ShouldEqual, "addr2")

			addr, err = w.GetServiceRoundRobin()
			So(err, ShouldBeNil)
			So(addr, ShouldEqual, "addr3")

			addr, err = w.GetServiceRoundRobin()
			So(err, ShouldBeNil)
			So(addr, ShouldEqual, "addr1")
		})

	})
}

func TestWall_GetComplianceConn(t *testing.T) {
	Convey("test GetComplianceConn", t, func() {

		Convey("test addr", func() {
			ctrl := gomock.NewController(t)
			mockPools := pool.NewMockPools(ctrl)
			mockPools.EXPECT().GetConn(context.Background(), "127.0.0.1:6480").Return(nil, nil)

			w := &wall{
				address:  "127.0.0.1:6480",
				connPool: mockPools,
			}
			_, err := w.GetComplianceConn()
			So(err, ShouldBeNil)
		})

		Convey("test addr, get conn err", func() {
			ctrl := gomock.NewController(t)
			mockPools := pool.NewMockPools(ctrl)
			mockPools.EXPECT().GetConn(context.Background(), "127.0.0.1:6480").Return(nil, mockErr)

			w := &wall{
				address:  "127.0.0.1:6480",
				connPool: mockPools,
			}
			_, err := w.GetComplianceConn()
			So(err, ShouldEqual, mockErr)
		})

		Convey("test get registry err", func() {
			sd := func(ctx context.Context, registry, namespace, group string) (addrs []string) {
				return []string{}
			}

			w := &wall{
				discovery: sd,
				index:     atomic.NewInt32(0),
			}
			_, err := w.GetComplianceConn()
			So(err, ShouldNotBeNil)
		})

		Convey("test registry", func() {
			sd := func(ctx context.Context, registry, namespace, group string) (addrs []string) {
				return []string{"addr"}
			}

			ctrl := gomock.NewController(t)
			mockPools := pool.NewMockPools(ctrl)
			mockPools.EXPECT().GetConn(context.Background(), "addr").Return(nil, nil)

			w := &wall{
				discovery: sd,
				index:     atomic.NewInt32(0),
				connPool:  mockPools,
			}
			_, err := w.GetComplianceConn()
			So(err, ShouldBeNil)
		})
	})
}

func TestWall_GetComplianceConfig(t *testing.T) {
	Convey("test get compliance config", t, func() {

		Convey("test get conn err", func() {
			w := &wall{}
			_, err := w.GetComplianceConfig(context.Background(), 0, "secen", 1234, false)
			So(err, ShouldNotBeNil)
		})

		Convey("test get config", func() {

			ctrl := gomock.NewController(t)
			mockConn := pool.NewMockConn(ctrl)
			mockConn.EXPECT().Client().Return(nil)
			mockConn.EXPECT().Close().Return(nil)

			mockGetConn := func() (pool.Conn, error) {
				return mockConn, nil
			}

			patch := gomonkey.ApplyFunc((*wall).GetComplianceConn, mockGetConn)
			defer patch.Reset()

			mockClient := mock.NewMockComplianceAPIClient(ctrl)
			mockClient.EXPECT().GetComplianceConfig(gomock.Any(), gomock.Any()).Return(nil, nil)
			mockNewClient := func(grpc.ClientConnInterface) compliance.ComplianceAPIClient {
				return mockClient
			}

			patch2 := gomonkey.ApplyFunc(compliance.NewComplianceAPIClient, mockNewClient)
			defer patch2.Reset()

			w := &wall{}
			_, err := w.GetComplianceConfig(context.Background(), 0, "secen", 1234, false)
			So(err, ShouldBeNil)
		})
	})
}

func TestWall_CheckStrategy(t *testing.T) {
	Convey("test check strategy", t, func() {

		Convey("test remote check", func() {
			mockGetConfig := func(w *wall, ctx context.Context, brokerID int32, scene string, uid int64, updateStrategy bool) (*compliance.GetComplianceConfigResp, error) {

				if uid == 0 {
					resp := &compliance.GetComplianceConfigResp{}
					resp.BrokerIds = []int32{0}
					resp.Sites = []string{BybitSiteID}
					return resp, nil
				}

				if uid == 1 {
					return nil, mockErr
				}

				if uid == 2 {
					resp := &compliance.GetComplianceConfigResp{}
					resp.IsWhitelist = true
					return resp, nil
				}

				if uid == 3 {
					resp := &compliance.GetComplianceConfigResp{}
					return resp, nil
				}

				if uid == 4 {
					resp := &compliance.GetComplianceConfigResp{}
					resp.BrokerIds = []int32{0}
					resp.Sites = []string{BybitSiteID}
					return resp, nil
				}

				if uid == 5 {
					resp := &compliance.GetComplianceConfigResp{}
					resp.BrokerIds = []int32{0}
					resp.Sites = []string{BybitSiteID}
					resp.IsKyc = true
					return resp, nil
				}

				if uid == 6 {
					resp := &compliance.GetComplianceConfigResp{}
					resp.BrokerIds = []int32{0}
					resp.Sites = []string{BybitSiteID}

					configs := &compliance.ComplianceConfigItem{}
					configs.SceneCode = "creat_order"

					straCfg := &compliance.ComplianceConfigItem_StrategyConfig{}
					straCfg.CountryCode = "CN"

					contryCfg := &compliance.ComplianceConfigItem_StrategyConfig_CountryConfig{}
					contryCfg.UserType = "USER_TYPE_KYC"
					contryCfg.DispositionResult = "result"
					straCfg.CountryConfig = append(straCfg.CountryConfig, contryCfg)

					straCfg.CountryConfig = append(straCfg.CountryConfig, nil)
					configs.StrategyConfig = append(configs.StrategyConfig, straCfg)

					configs.StrategyConfig = append(configs.StrategyConfig, nil)
					resp.SceneItem = configs
					resp.SceneItems = []*compliance.ComplianceConfigItem{configs}
					return resp, nil
				}

				if uid == 7 {
					resp := &compliance.GetComplianceConfigResp{}
					resp.BrokerIds = []int32{0}
					resp.Sites = []string{BybitSiteID}
					resp.IsKyc = true
					resp.KycCountry = "CN"

					configs := &compliance.ComplianceConfigItem{}
					configs.SceneCode = "creat_order"

					straCfg := &compliance.ComplianceConfigItem_StrategyConfig{}
					straCfg.CountryCode = "CN"

					contryCfg := &compliance.ComplianceConfigItem_StrategyConfig_CountryConfig{}
					contryCfg.UserType = "USER_TYPE_KYC"
					contryCfg.DispositionResult = "result"
					straCfg.CountryConfig = append(straCfg.CountryConfig, contryCfg)
					configs.StrategyConfig = append(configs.StrategyConfig, straCfg)

					configs.StrategyConfig = append(configs.StrategyConfig, nil)
					resp.SceneItem = configs
					resp.SceneItems = []*compliance.ComplianceConfigItem{configs}
					return resp, nil
				}

				if uid == 8 {
					resp := &compliance.GetComplianceConfigResp{}
					resp.BrokerIds = []int32{0}
					resp.Sites = []string{BybitSiteID}
					resp.IsKyc = true
					resp.KycCountry = "CN"

					configs := &compliance.ComplianceConfigItem{}
					configs.SceneCode = "creat_order"

					straCfg := &compliance.ComplianceConfigItem_StrategyConfig{}
					straCfg.CountryCode = "CN"

					contryCfg := &compliance.ComplianceConfigItem_StrategyConfig_CountryConfig{}
					contryCfg.UserType = "USER_TYPE_KYC"
					contryCfg.DispositionResult = "PASS"
					straCfg.CountryConfig = append(straCfg.CountryConfig, contryCfg)

					straCfg.CountryConfig = append(straCfg.CountryConfig, nil)
					configs.StrategyConfig = append(configs.StrategyConfig, straCfg)

					configs.StrategyConfig = append(configs.StrategyConfig, nil)
					resp.SceneItem = configs
					resp.SceneItems = []*compliance.ComplianceConfigItem{configs}

					ug := &compliance.ComplianceUserItem{
						Site:  BybitSiteID,
						Group: "A",
					}
					resp.UserItems = []*compliance.ComplianceUserItem{ug}
					return resp, nil
				}

				if uid == 10 {
					resp := &compliance.GetComplianceConfigResp{}
					resp.BrokerIds = []int32{0}
					resp.Sites = []string{BybitSiteID}
					resp.IsKyc = true
					resp.KycCountry = "CN"

					configs := &compliance.ComplianceConfigItem{}
					configs.SceneCode = "creat_order"

					straCfg := &compliance.ComplianceConfigItem_StrategyConfig{}
					straCfg.CountryCode = "CN"

					contryCfg := &compliance.ComplianceConfigItem_StrategyConfig_CountryConfig{}
					contryCfg.UserType = "USER_TYPE_KYC"
					contryCfg.DispositionResult = "PASS"
					straCfg.CountryConfig = append(straCfg.CountryConfig, contryCfg)

					straCfg.CountryConfig = append(straCfg.CountryConfig, nil)
					configs.StrategyConfig = append(configs.StrategyConfig, straCfg)

					configs.StrategyConfig = append(configs.StrategyConfig, nil)
					resp.SceneItem = configs
					resp.SceneItems = []*compliance.ComplianceConfigItem{configs}
					return resp, nil
				}

				return nil, mockErr
			}
			patch := gomonkey.ApplyFunc((*wall).GetComplianceConfig, mockGetConfig)
			defer patch.Reset()

			w := &wall{}

			_, _, err := w.CheckStrategy(context.Background(), 0, BybitSiteID, "scene", 1, "Country", "", SourceApp)
			So(err, ShouldEqual, mockErr)

			_, hit, err := w.CheckStrategy(context.Background(), 0, BybitSiteID, "scene", 2, "Country", "", SourceApp)
			So(err, ShouldBeNil)
			So(hit, ShouldBeFalse)

			_, hit, err = w.CheckStrategy(context.Background(), 0, BybitSiteID, "scene", 3, "Country", "", SourceApp)
			So(err, ShouldBeNil)
			So(hit, ShouldBeFalse)

			_, hit, err = w.CheckStrategy(context.Background(), 0, BybitSiteID, "scene", 4, "Country", "", SourceApp)
			So(err, ShouldBeNil)
			So(hit, ShouldBeFalse)

			_, hit, err = w.CheckStrategy(context.Background(), 0, BybitSiteID, "scene", 5, "Country", "", SourceApp)
			So(err, ShouldBeNil)
			So(hit, ShouldBeFalse)

			_, hit, err = w.CheckStrategy(context.Background(), 0, BybitSiteID, "scene", 0, "Country", "", SourceApp)
			So(err, ShouldBeNil)
			So(hit, ShouldBeFalse)

			_, hit, err = w.CheckStrategy(context.Background(), 0, BybitSiteID, "creat_order", 6, "CN", "", SourceApp)
			So(err, ShouldBeNil)
			So(hit, ShouldBeFalse)

			_, hit, err = w.CheckStrategy(context.Background(), 0, BybitSiteID, "creat_order", 6, "EN", "", SourceApp)
			So(err, ShouldBeNil)
			So(hit, ShouldBeFalse)

			_, hit, err = w.CheckStrategy(context.Background(), 0, BybitSiteID, "creat_order", 7, "EN", "", SourceApp)
			So(err, ShouldBeNil)
			So(hit, ShouldBeTrue)

			_, hit, err = w.CheckStrategy(context.Background(), 0, "", "creat_order", 7, "EN", "", SourceApp)
			So(err, ShouldBeNil)
			So(hit, ShouldBeTrue)

			_, hit, err = w.CheckStrategy(context.Background(), 0, BybitSiteID, "creat_order", 8, "EN", "", SourceApp)
			So(err, ShouldBeNil)
			So(hit, ShouldBeFalse)

			w.SetCityConfig([]string{"CN"}, []string{"BJ"})
			_, hit, err = w.CheckStrategy(context.Background(), 0, BybitSiteID, "creat_order", 8, "CN", "BJ", SourceApp)
			So(err, ShouldBeNil)
			So(hit, ShouldBeFalse)
		})

		Convey("test local check", func() {
			mockGetConfig := func(w *wall, ctx context.Context, brokerID int32, scene string, uid int64, updateStrategy bool) (*compliance.GetComplianceConfigResp, error) {
				resp := &compliance.GetComplianceConfigResp{}
				resp.BrokerIds = []int32{0}
				resp.Sites = []string{BybitSiteID}
				resp.IsKyc = true
				resp.KycCountry = "CN"

				configs := &compliance.ComplianceConfigItem{}
				configs.SceneCode = "creat_order"

				straCfg := &compliance.ComplianceConfigItem_StrategyConfig{}
				straCfg.CountryCode = "CN"

				contryCfg := &compliance.ComplianceConfigItem_StrategyConfig_CountryConfig{}
				contryCfg.UserType = "USER_TYPE_KYC"
				contryCfg.DispositionResult = "PASS"
				straCfg.CountryConfig = append(straCfg.CountryConfig, contryCfg)

				straCfg.CountryConfig = append(straCfg.CountryConfig, nil)
				configs.StrategyConfig = append(configs.StrategyConfig, straCfg)

				configs.StrategyConfig = append(configs.StrategyConfig, nil)
				resp.SceneItem = configs
				return resp, nil
			}

			patch := gomonkey.ApplyFunc((*wall).GetComplianceConfig, mockGetConfig)
			defer patch.Reset()

			ac := NewAddrCfg("127.0.0.1:6480")
			w, _ := NewWall(ac, true)
			_, hit, err := w.CheckStrategy(context.Background(), 0, BybitSiteID, "creat_order", 10, "CN", "", SourceApp)
			So(err, ShouldBeNil)
			So(hit, ShouldBeFalse)

			_, hit, err = w.CheckStrategy(context.Background(), 0, "", "creat_order", 10, "CN", "", SourceApp)
			So(err, ShouldBeNil)
			So(hit, ShouldBeFalse)

			w.SetCityConfig([]string{"CN"}, []string{"BJ"})
			_, hit, err = w.CheckStrategy(context.Background(), 0, BybitSiteID, "creat_order", 10, "CN", "BJ", SourceApp)
			So(err, ShouldBeNil)
			So(hit, ShouldBeFalse)

			_, hit, err = w.CheckStrategy(context.Background(), 0, BybitSiteID, "creat_order", 10, "CN", "SH", SourceApp)
			So(err, ShouldBeNil)
			So(hit, ShouldBeFalse)

			_, hit, err = w.CheckStrategy(context.Background(), 0, "HKG", "creat_order", 10, "CN", "SH", SourceApp)
			So(err, ShouldBeNil)
			So(hit, ShouldBeFalse)
		})
	})
}

func TestWall_SetCityConfig(t *testing.T) {
	Convey("test wall SetCityConfig", t, func() {
		ac := NewAddrCfg("127.0.0.1:6480")
		w, _ := NewWall(ac, true)
		w.SetCityConfig([]string{"CN"}, []string{"BJ"})
	})
}
