package compliance

import (
	"context"
	"flag"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcompliance"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/fbu/gateway/gway.git/gtrace"
	jsoniter "github.com/json-iterator/go"
	"gopkg.in/yaml.v2"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/config_center/nacos"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/geoip"
)

func Init() {
	filter.Register(filter.ComplianceWallFilterKey, New)
}

func New() filter.Filter {
	return &complianceWall{}
}

const (
	siteUAE = "ARE"
)

const alertComplianceTitle = "compliance city config"

var (
	Once sync.Once
)

type complianceWall struct {
	SceneCode           string
	ReduceOnlyKeys      []string
	BatchItemKey        string
	KycInfo             bool
	product             string
	UaeCheck            string
	UaeLeverageCheck    []string
	uaeSymbolCheck      *UaeSymbolCheck
	batchUaeSymbolCheck *UaeSymbolCheck
	SLT                 *SpotLeveragedToken
	multiScenes         map[string]string
}

type UaeSymbolCheck struct {
	Category    []string
	SymbolField string
}

type SpotLeveragedToken struct {
	Scene       string
	Category    string
	SymbolField string
}

func (c *complianceWall) GetName() string {
	return filter.ComplianceWallFilterKey
}

func (c *complianceWall) Init(ctx context.Context, args ...string) error {
	parse := flag.NewFlagSet("compliance", flag.ContinueOnError)
	var (
		reduceOnly       string
		SLT              string
		UaeLeverageCheck string
		usc              string
		busc             string
		multiScene       string
	)
	parse.StringVar(&c.SceneCode, "scene", "", "scene code")
	parse.StringVar(&reduceOnly, "reduceOnlyKey", "", "reduce only key")
	parse.StringVar(&c.BatchItemKey, "batchItemsKey", "", "batch item key")
	parse.BoolVar(&c.KycInfo, "kycInfo", false, "if kyc info need")
	parse.StringVar(&c.product, "product", "", "site info for which product")
	parse.StringVar(&UaeLeverageCheck, "uaeLeverageCheck", "", "check uae config")
	parse.StringVar(&usc, "uaeSymbolCheck", "", "check uae config")
	parse.StringVar(&busc, "batchUaeSymbolCheck", "", "check uae config for batch order")
	parse.StringVar(&SLT, "spotLeveragedToken", "", "SpotLeveragedToken config")
	parse.StringVar(&multiScene, "multiScenes", "", "multi scenes")
	if err := parse.Parse(args[1:]); err != nil {
		glog.Error(ctx, "compliance wall init failed", glog.String("error", err.Error()), glog.Any("args", args))
		return err
	}

	if UaeLeverageCheck != "" {
		c.UaeLeverageCheck = strings.Split(UaeLeverageCheck, ",")
	}

	if usc != "" {
		u := &UaeSymbolCheck{}
		_ = jsoniter.Unmarshal([]byte(usc), u)
		if u.SymbolField != "" {
			c.uaeSymbolCheck = u
		}
	}

	if busc != "" {
		u := &UaeSymbolCheck{}
		_ = jsoniter.Unmarshal([]byte(busc), u)
		if len(u.Category) > 0 {
			c.batchUaeSymbolCheck = u
		}
	}

	if SLT != "" {
		slt := &SpotLeveragedToken{}
		_ = jsoniter.Unmarshal([]byte(SLT), slt)
		if slt.Scene != "" && slt.SymbolField != "" {
			c.SLT = slt
		}
	}

	if multiScene != "" {
		m := make(map[string]string)
		_ = jsoniter.Unmarshal([]byte(multiScene), &m)
		if len(m) > 0 {
			c.multiScenes = m
		}
	}

	if reduceOnly != "" {
		c.ReduceOnlyKeys = strings.Split(reduceOnly, ",")
	}

	glog.Debug(ctx, "rules ", glog.String("scene", c.SceneCode),
		glog.String("reduceOnly", reduceOnly),
		glog.String("batchItem", c.BatchItemKey),
		glog.String("uaeLeverageCheck", UaeLeverageCheck),
		glog.String("uaeSymbolCheck", usc),
		glog.String("SpotLeveragedToken", SLT),
		glog.String("multiScenes", multiScene))

	Once.Do(func() {
		_ = initComplianceService()
		_ = buildListen(ctx)
	})

	if gm, err := geoip.NewGeoManager(); err != nil || gm == nil {
		galert.Error(ctx, fmt.Sprintf("complianceWall NewGeoManager error:%v", err))
	}
	return nil
}

func (c *complianceWall) Do(next types.Handler) types.Handler {
	return func(ctx *types.Ctx) error {
		md := metadata.MDFromContext(ctx)
		if md.IsDemoUID {
			glog.Debug(ctx, "demo member skip compliance wall", glog.Int64("uid", md.UID), glog.String("pid", md.ParentUID))
			return next(ctx)
		}

		siteID := md.SiteID
		brokerID := md.BrokerID
		uid := md.UID
		scene := c.getScene(ctx)
		source := md.Extension.Platform
		ip := md.Extension.RemoteIP
		iso3, sbuDivision, err := c.getGeoIP(ctx, ip)
		if err != nil {
			// if get country and sbuDivision failed, do not skip match
			gmetric.IncDefaultError("compliance_wall_geo_err", scene)
		}

		glog.Debug(ctx, "complianceWall_filter CheckStrategy requeset",
			glog.Int32("broker", brokerID),
			glog.String("site id", siteID),
			glog.String("ip", ip),
			glog.String("country", iso3),
			glog.String("sbuDivision", sbuDivision),
			glog.Int64("uid", uid),
			glog.String("scene", scene),
			glog.String("source", source),
			glog.String("user-site-id", md.UserSiteID))

		span, _ := gtrace.Begin(ctx, "compliance-wall")
		span.SetTag("uid", uid)
		span.SetTag("scene", scene)
		now := time.Now()
		res, hit, err := gw.CheckStrategy(ctx, brokerID, siteID, scene, uid, iso3, sbuDivision, source, md.UserSiteID)
		gmetric.ObserveDefaultLatencySince(now, "compliance", scene)
		gtrace.Finish(span)
		if err != nil {
			gmetric.IncDefaultError("compliance_wall_err", scene)
			glog.Error(ctx, "compliance wall failed",
				glog.String("scene", scene), glog.Int64("uid", uid), glog.String("err", err.Error()))
			return next(ctx)
		}

		if c.KycInfo {
			ui, err := gw.GetUserInfo(ctx, uid)
			if err == nil {
				md.KycCountry = ui.Country
				md.KycLevel = ui.KycLevel
			}
		}

		if c.product != "" {
			temp := siteID
			if siteID == "" || siteID == gcompliance.BybitSiteID {
				temp = md.UserSiteID
			}
			cfg, _, err := gw.GetSiteConfig(ctx, brokerID, uid, temp, c.product, md.UserSiteID)
			if err != nil {
				gmetric.IncDefaultError("compliance_wall_sitecfg_err", scene)
				glog.Error(ctx, "compliance wall get site config err", glog.String("scene", scene), glog.String("err", err.Error()))
			}
			glog.Debug(ctx, "site config", glog.String("id", siteID), glog.String("product", c.product),
				glog.String("cfg", cfg))
			md.SiteConf = cfg
		}

		if len(c.UaeLeverageCheck) > 0 {
			err := uaeLeverageCheck(ctx, brokerID, uid, siteID, c.UaeLeverageCheck, md)
			if err != nil {
				return err
			}
		}

		var symbolErr error
		if c.uaeSymbolCheck != nil {
			glog.Debug(ctx, "uae check", glog.Any("flag", c.uaeSymbolCheck), glog.String("siteID", siteID))
			symbolErr = uaeSymbolCheck(ctx, brokerID, uid, siteID, c.uaeSymbolCheck, md)
		}

		if c.batchUaeSymbolCheck != nil {
			glog.Debug(ctx, "uae check", glog.Any("flag", c.batchUaeSymbolCheck), glog.String("siteID", siteID))
			err = batchUaeSymbolCheck(ctx, brokerID, uid, siteID, c.batchUaeSymbolCheck, md)
			// 目前只有批量下撤单获取symbol失败会返回参数错误
			if err != nil {
				return err
			}
		}

		if !hit && symbolErr == nil {
			return next(ctx)
		}

		// reduce only
		if len(c.ReduceOnlyKeys) > 0 {
			for _, key := range c.ReduceOnlyKeys {
				v, _ := util.JsonGetBool(ctx.Request.Body(), key)
				if v {
					return next(ctx)
				}
			}
		}

		// batch item
		if c.BatchItemKey != "" {
			path := fmt.Sprintf("$.%s", c.BatchItemKey)
			l, _ := util.JsonpathGet(ctx.Request.Body(), path)
			batchItems, ok := l.([]interface{})
			if ok && len(batchItems) == 1 {
				v, ok := batchItems[0].(bool)
				if ok && v {
					return next(ctx)
				}
			}
		}

		if hit {
			d, _ := marshalComplianceResult(res)
			ctx.Response.SetBody(d)

			glog.Info(ctx, "complianceWall_filter CheckStrategy hit",
				glog.String("site id", md.SiteID),
				glog.String("user site id", md.UserSiteID),
				glog.String("ip", ip),
				glog.String("country", iso3),
				glog.String("sbuDivision", sbuDivision),
				glog.Int64("uid", uid),
				glog.String("scene", scene))
			return berror.ErrComplianceRuleTriggered
		}

		if symbolErr != nil {
			return symbolErr
		}

		return next(ctx)
	}
}

func (c *complianceWall) getGeoIP(ctx context.Context, ip string) (string, string, error) {
	db, err := geoip.NewGeoManager()
	if err != nil {
		return "", "", fmt.Errorf("complianceWall NewGeoManager error: %w", err)
	}

	d, err := db.QueryCityAndCountry(ctx, ip)
	if err != nil || d == nil {
		return "", "", err
	}

	sub := d.GetCity().GetSubdivision()
	id := strconv.FormatInt(int64(sub.GetGeoNameID()), 10)
	return d.GetCountry().GetISO3(), d.GetCountry().GetISO3() + "-" + id, nil
}

const file = "compliance_city_config"

func buildListen(ctx context.Context) error {
	nacosCfg, err := nacos.NewNacosConfigure(
		context.Background(),
		nacos.WithGroup(constant.BGW_GROUP),              // specified group
		nacos.WithNameSpace(constant.BGWConfigNamespace), // namespace isolation
	)
	if err != nil {
		glog.Error(ctx, "compliance city config listen NewNacosConfigure error", glog.String("error", err.Error()))
		return err
	}

	// listen nacos config
	l := &listener{}
	if err = nacosCfg.Listen(ctx, file, l); err != nil {
		galert.Error(ctx, "listen error"+err.Error(), galert.WithField("file", file), galert.WithTitle(alertComplianceTitle))
		return err
	}

	return nil
}

type cityCfg struct {
	Countries    []string `yaml:"countries"`
	SubDivisions []string `yaml:"subDivisions"`
}

type listener struct{}

func (l *listener) OnEvent(event observer.Event) error {
	e, ok := event.(*observer.DefaultEvent)
	if !ok {
		return nil
	}
	if e.Value == "" {
		return nil
	}

	cfg := &cityCfg{
		Countries:    make([]string, 0),
		SubDivisions: make([]string, 0),
	}

	if err := yaml.Unmarshal([]byte(e.Value), cfg); err != nil {
		galert.Error(context.Background(), "listener error"+err.Error(), galert.WithField("file", e.Key), galert.WithTitle(alertComplianceTitle))
		return nil
	}

	gw.SetCityConfig(cfg.Countries, cfg.SubDivisions)

	galert.Info(context.Background(), "build success", galert.WithField("file", e.Key), galert.WithTitle(alertComplianceTitle))

	return nil
}

func (l *listener) GetEventType() reflect.Type {
	return nil
}

func (l *listener) GetPriority() int {
	return 0
}
