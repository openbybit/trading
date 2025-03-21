package openapi

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/sign"
	"code.bydev.io/fbu/gateway/gway.git/glog"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/geoip"
	ropenapi "bgw/pkg/service/openapi"
)

const logKeyClientIp = "client ip"

type Checker interface {
	GetAPITimestamp() string
	GetAPIRecvWindow() string
	GetAPIKey() string
	GetAPISign() string
	GetClientIP() string
	GetVersion() string
	VerifySign(ctx *types.Ctx, signTyp sign.Type, secret string) error
}

func (o *openapi) getCheckers(ctx *types.Ctx, md *metadata.Metadata, allowGuest, fallbackParse bool) (ret [2]Checker, err error) {
	apikey := string(ctx.Request.Header.Peek(constant.HeaderAPIKey))
	if apikey != "" {
		v3s, e := newV3Checker(ctx, apikey, md.Extension.RemoteIP, allowGuest, md.WssFlag)
		if e != nil {
			return ret, e
		}
		ret = v3s
	} else {
		if md.WssFlag {
			glog.Debug(ctx, "wss only support v3, raw data",
				glog.String("content-type", string(ctx.Request.Header.ContentType())),
				glog.String("headers", cast.UnsafeBytesToString(ctx.Request.Header.Header())),
				glog.String("body", cast.UnsafeBytesToString(ctx.Request.Body())),
				glog.String("query", cast.UnsafeBytesToString(ctx.URI().QueryString())),
			)
			return ret, berror.NewBizErr(errParams, "wss only support v3")
		}
		v2s, e := newV2Checker(ctx, md.Extension.RemoteIP, allowGuest, fallbackParse)
		if e != nil {
			return ret, e
		}
		ret = v2s
	}

	if allowGuest {
		return
	}

	expireTime, err := o.antiReplay(ctx, ret[0].GetAPITimestamp(), ret[0].GetAPIRecvWindow())
	if err != nil {
		return
	}
	md.Extension.ReqExpireTime = cast.ToString(expireTime)
	md.Extension.ReqExpireTimeE9 = expireTime
	return
}

func (o *openapi) checkBannedCountries(ctx *types.Ctx, rule *openapiRule, ip string) error {
	if rule.skipIpCheck {
		return nil
	}

	if geoip.CheckIPWhitelist(ctx, ip) {
		return nil
	}

	db, err := geoip.NewGeoManager()
	if err != nil {
		return fmt.Errorf("openapi NewGeoManager error: %w", err)
	}
	data, err := db.QueryCityAndCountry(ctx, ip)
	if err != nil || data == nil {
		return nil
	}
	iso := data.GetCountry().GetISO()
	if iso == "" {
		glog.Debug(ctx, "banned remoteGeo query geoData is nil", glog.String("ip", ip))
	} else {
		if strings.Contains(o.bc, iso) {
			return berror.ErrCountryBanned
		}
	}
	return nil
}

func (o *openapi) antiReplay(ctx *types.Ctx, apiTimestamp, window string) (int64, error) {
	var requestTime int64
	var allowTimeDiff int64 = 1000
	var recvWindow int64 = 5000

	current := time.Now().UnixNano() / int64(time.Millisecond)

	val, err := strconv.ParseInt(apiTimestamp, 10, 64)
	if err != nil {
		return 0, berror.NewBizErr(errParams, "openapi params error! req_timestamp invalid")
	}
	requestTime = val

	if window != "" {
		val, err = strconv.ParseInt(window, 10, 64)
		if err != nil {
			return 0, berror.NewBizErr(errParams, "openapi params error! recv_window invalid")
		}
		recvWindow = val
	}

	if requestTime >= current+allowTimeDiff || requestTime < current-recvWindow {
		glog.Debug(ctx, "antiReplay error", glog.Int64("requestTime", requestTime), glog.Int64("current", current),
			glog.Int64("recv", recvWindow), glog.Int64("right", current+allowTimeDiff), glog.Int64("left", current-recvWindow),
		)
		return 0, berror.NewBizErr(errTimestamp, fmt.Sprintf(errMsgTimestamp+"req_timestamp[%d],server_timestamp[%d],recv_window[%d]", requestTime, current, recvWindow))
	}

	return (requestTime + recvWindow) * 1e6, nil
}

func (o *openapi) checkIp(ctx *types.Ctx, md *metadata.Metadata, skipIpCheck bool, member *ropenapi.MemberLogin, ip string) (err error) {
	defer func() {
		if skipIpCheck && err != nil {
			glog.Info(ctx, "bind checkIp block", glog.String("apikey", member.LoginName), glog.Int64("uid", member.MemberId),
				glog.Int64("apikeyType", member.ExtInfo.ApiKeyType), glog.String("app-note", member.ExtInfo.Note), glog.String("ip", ip))
			if member.ExtInfo.ApiKeyType == 1 && getIpCheckMgr().CanSkipIpCheck(member.MemberId) {
				err = nil
			}
		}
	}()

	if member.ExtInfo.ApiKeyType == 2 {
		if md.Extension.Referer == "" && md.Extension.XReferer == "" {
			glog.Error(ctx, "openapi not pass refer or x-refer",
				glog.String("appId", member.ExtInfo.AppId),
				glog.Int64("uid", member.MemberId),
			)
		}
		if o.nacosLoader == nil {
			glog.Error(ctx, "openapi getIpWhiteList error, o.nacosLoader is nil", glog.String(logKeyClientIp, ip))
			return berror.ErrInvalidIP
		}
		whiteListIps, ok := o.nacosLoader.GetIpWhiteList(ctx, member.ExtInfo)
		if ok && (whiteListIps == "" || whiteListIps == "*" || strings.Contains(whiteListIps, ip)) {
			return nil
		}
		glog.Debug(ctx, "apikey ips", glog.String("ips", whiteListIps), glog.String(logKeyClientIp, ip))
		return berror.ErrInvalidIP
	} else {
		if member.ExtInfo.Ips == "[\"*\"]" {
			return nil
		}

		if strings.Contains(member.ExtInfo.Ips, ip) {
			return nil
		}
		glog.Debug(ctx, "apikey ips", glog.String("ips", member.ExtInfo.Ips), glog.String(logKeyClientIp, ip))
	}

	return berror.ErrInvalidIP
}

const (
	singleGroup = 1
	multiGroup  = 2
	allGroup    = 3
)

func (o *openapi) checkPermission(ctx *types.Ctx, permissions string, acl metadata.ACL) error {
	if acl.Group == constant.ResourceGroupInvalid || acl.Permission == constant.PermissionInvalid || acl.Permission == "" {
		glog.Debug(ctx, "openapi Permission ACL invalid", glog.Any("acl", acl))
		return berror.ErrRouteKeyInvalid
	}

	var permission [][]interface{}
	// 新数据
	if strings.HasPrefix(permissions, "[") {
		if err := util.JsonUnmarshalString(permissions, &permission); err != nil {
			glog.Debug(ctx, "checkPermission UnmarshalFromString error", glog.String("error", err.Error()), glog.String("pstr", permissions))
		}
	} else {
		permission = [][]interface{}{{permissions, false}}
	}

	glog.Debug(ctx, "permission", glog.Any("ACL", acl), glog.String("permission", permissions))
	// common api
	if acl.Group == "" {
		if acl.AllGroup && len(permission) > 0 {
			glog.Debug(ctx, "common api, all group")
			return o.checkDbPermission(ctx, acl, permission, allGroup)
		}
		glog.Debug(ctx, "common api, multi group")
		return o.checkDbPermission(ctx, acl, permission, multiGroup)
	}

	// standard api
	return o.checkDbPermission(ctx, acl, permission, singleGroup)
}

func (o *openapi) checkDbPermission(ctx context.Context, acl metadata.ACL, dbPermission [][]interface{}, t int) error {
	for _, item := range dbPermission {
		if len(item) < 2 {
			glog.Error(ctx, "openapi permission len error", glog.Any("permission", item))
			return berror.ErrOpenAPIPermission
		}
		switch t {
		case singleGroup:
			apiTypeStr, ok := item[0].(string)
			if !ok {
				glog.Error(ctx, "openapi group assert to string error", glog.Any("permission", dbPermission))
				return berror.ErrOpenAPIPermission
			}
			apiType := constant.GetRouteGroup(apiTypeStr)
			if apiType == acl.Group || apiType == constant.ResourceGroupAll { // 此处的All包含BlockTrade
				return o.checkReadOnly(ctx, item, acl.Permission)
			}
		case multiGroup:
			apiTypeStr, ok := item[0].(string)
			if !ok {
				glog.Error(ctx, "openapi group assert to string error", glog.Any("ACL", acl), glog.Any("permission", dbPermission))
				return berror.ErrOpenAPIPermission
			}
			apiType := constant.GetRouteGroup(apiTypeStr)
			for _, group := range acl.Groups {
				if apiType == group || apiType == constant.ResourceGroupAll { // 此处的All包含BlockTrade
					return o.checkReadOnly(ctx, item, acl.Permission)
				}
			}
		case allGroup:
			if err := o.checkReadOnly(ctx, item, acl.Permission); err == nil {
				return nil
			}
		}
	}
	return berror.ErrOpenAPIPermission
}

func (o *openapi) checkReadOnly(ctx context.Context, item []interface{}, permission string) error {
	// item[1] = true,表示只读
	readOnly, ok := item[1].(bool)
	if !ok {
		glog.Error(ctx, "openapi permission assert to bool error", glog.Any("permission", item))
		return berror.ErrOpenAPIPermission
	}
	if readOnly {
		if constant.PermissionRead != permission && constant.PermissionReadWrite != permission {
			return berror.ErrOpenAPIPermission
		}
	}
	// 可以访问该接口
	return nil
}
