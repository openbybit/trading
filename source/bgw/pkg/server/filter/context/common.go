package context

import (
	"bytes"
	"regexp"
	"strings"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	gmetadata "bgw/pkg/server/metadata"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gtrace"
	"git.bybit.com/svc/mod/pkg/bplatform"
	"git.bybit.com/svc/stub/pkg/svc/cmd"
	"google.golang.org/grpc/metadata"
)

const all = "ALL"

type mdCarrier interface {
	Metadata() metadata.MD
}

var reLangTag = regexp.MustCompile("(?i)^([a-z]{2})(?:-([a-z]{2}))?$")

var innerOutboundHeaders = map[string]struct{}{
	constant.BgwAPIResponseCodes:      {},
	constant.BgwAPIResponseMessages:   {},
	constant.BgwAPIResponseExtMaps:    {},
	constant.BgwAPIResponseExtInfos:   {},
	constant.BgwAPIResponseStatusCode: {},
	constant.BgwAPIResponseFlag:       {},
}

var innerInboundHeaders = map[string]struct{}{
	constant.Guid:                         {},
	constant.DeviceID:                     {},
	strings.ToLower(constant.XOriginFrom): {},
	strings.ToLower(constant.BrokerID):    {},
	strings.ToLower(constant.CallOrigin):  {},
	constant.Baggage:                      {},
	constant.Platform:                     {},
	constant.XAKMTraceID:                  {},
	strings.ToLower(constant.XReferer):    {},
	strings.ToLower(constant.Referer):     {},
	strings.ToLower(constant.ReqInitAtE9): {},
	strings.ToLower(constant.XClientTag):  {},
	strings.ToLower(constant.UserAgent):   {},
}

func parseBaggage(c *types.Ctx, md *gmetadata.Metadata) {
	const baggageKey = "baggage"
	baggage := gtrace.SpanFromContext(c).BaggageItem(baggageKey)
	md.Intermediate.Baggage = baggage
	if env.IsProduction() {
		return
	}
	var laneEnv string
	tags := strings.Split(baggage, ",")
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if !strings.Contains(tag, constant.XLaneEnv) {
			continue
		}
		envs := strings.SplitN(tag, "=", 2)
		if len(envs) == 2 {
			laneEnv = envs[1]
		}
		break
	}
	md.Intermediate.LaneEnv = laneEnv
}

func parseCommon(c *types.Ctx, md *gmetadata.Metadata) {
	hmeta := &cmd.HTTPRequestMeta{
		Method:    util.DecodeHeaderValue(c.Request.Header.Method()),
		Uri:       util.DecodeHeaderValue(c.Request.URI().RequestURI()),
		Host:      util.DecodeHeaderValue(c.Request.Host()),
		UserAgent: util.DecodeHeaderValue(c.Request.Header.UserAgent()),
		Referer:   util.DecodeHeaderValue(c.Request.Header.Referer()),
		Platform:  util.DecodeHeaderValue(c.Request.Header.Peek(constant.Platform)),
	}
	md.Path = string(c.Path())
	md.BrokerID = int32(cast.Atoi(string(c.Request.Header.Peek(constant.BrokerID))))
	md.SiteID = string(c.Request.Header.Peek(constant.SiteID))
	md.Method = hmeta.Method
	md.Extension.URI = hmeta.Uri
	md.Extension.Host = hmeta.Host
	md.Extension.UserAgent = hmeta.UserAgent
	md.Extension.Referer = hmeta.Referer
	md.Extension.OriginPlatform = hmeta.Platform
	pc := bplatform.ParseClientWithAppName(hmeta)
	md.Extension.AppVersion = pc.Version
	md.Extension.AppName = pc.AppName
	if pc.Client == bplatform.AndroidAPP || pc.Client == bplatform.IOSAPP {
		md.Extension.AppVersionCode = parseVersionCode(c)
	}
	md.Extension.OpFrom = string(pc.Client)
	md.Extension.Platform, md.Extension.EPlatform = parsePlatform(pc.Client)
	md.Extension.OpPlatform, md.Extension.EOpPlatform = parseOpPlatform(pc.Client)
}

func parseLang(c *types.Ctx) string {
	lang := util.DecodeHeaderValue(c.Request.Header.Peek(constant.Lang))
	if lang == "" {
		alang := c.Request.Header.Peek(constant.AcceptLanguage)

		match := reLangTag.FindSubmatch(alang)
		if len(match) == 0 {
			return ""
		}
		sb := &strings.Builder{}
		sb.Write(bytes.ToLower(match[1]))
		if len(match[2]) > 0 {
			sb.WriteRune('-')
			sb.Write(bytes.ToLower(match[2]))
		}

		return sb.String()
	}
	return lang
}
func parsePlatform(client bplatform.Client) (string, int32) {
	platform := client.CMDPlatform()
	if platform == 0 {
		return "", 0
	}
	return string(client), int32(platform)
}

func parseOpPlatform(client bplatform.Client) (string, int32) {
	op := client.OPPlatform()
	if op == 0 {
		return "", 0
	}
	return string(client), int32(op)
}

func parseVersionCode(c *types.Ctx) string {
	vc := c.Request.Header.Peek("Versioncode")
	for _, b := range vc {
		if b < '0' || b > '9' {
			return ""
		}
	}

	return string(vc)
}
