package openapi

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/bhttp"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/server/metadata"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gcore/sign"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	jsoniter "github.com/json-iterator/go"
)

const (
	errParams    = 10001
	errTimestamp = 10002
	errorSign    = 10004
)

const (
	errMsgTimestamp = "invalid request, please check your server timestamp or recv_window param. "
)

type v2Checker struct {
	apiTimestamp  string
	apiRecvWindow string
	apiKey        string
	apiSignature  string
	params        map[string]string
	postQueryData map[string]string
	postQueryFlag bool
	remoteIp      string
}

func newV2Checker(ctx *types.Ctx, ip string, allowGuest, queryFallback bool) (ret [2]Checker, err error) {
	var (
		params        map[string]string
		postQueryData map[string]string
		postUseQuery  bool // post use query flag
	)
	switch {
	case ctx.IsGet():
		params = make(map[string]string, 10)
		ctx.QueryArgs().VisitAll(func(key, value []byte) {
			params[string(key)] = string(value)
		})
	default:
		glog.Debug(ctx, "v2 raw data", glog.String("content-type", string(ctx.Request.Header.ContentType())),
			glog.String("body", cast.UnsafeBytesToString(ctx.Request.Body())),
			glog.String("query", cast.UnsafeBytesToString(ctx.URI().QueryString())),
		)
		switch {
		case len(ctx.Request.Body()) == 0:
			// querystring peek
			params = queryParse(ctx.URI().QueryString())
			// data standard
			postQueryData = make(map[string]string, 10)
			iter := func(key []byte, val []byte) {
				postQueryData[string(key)] = string(val)
			}
			ctx.QueryArgs().VisitAll(iter)
			body, err := jsoniter.Marshal(postQueryData)
			if err != nil {
				glog.Debug(ctx, "json Marshal error", glog.String("error", err.Error()))
				return ret, berror.NewBizErr(errParams, "invalid params")
			}
			postUseQuery = true
			metadata.ContextWithRequestHandledBody(ctx, body)
		case bytes.HasPrefix(ctx.Request.Header.ContentType(), bhttp.ContentTypePostForm):
			// body form peek
			params = make(map[string]string, 10)
			iter := func(key []byte, val []byte) {
				params[string(key)] = string(val)
			}
			ctx.PostArgs().VisitAll(iter)

			// merge query string
			if queryFallback {
				ctx.QueryArgs().VisitAll(func(key, value []byte) {
					params[string(key)] = string(value)
				})
				body, err := jsoniter.Marshal(params)
				if err != nil {
					glog.Debug(ctx, "json Marshal error", glog.String("error", err.Error()))
					return ret, berror.NewBizErr(errParams, "invalid params")
				}
				metadata.ContextWithRequestHandledBody(ctx, body)
			}
		case queryFallback && len(ctx.URI().QueryString()) > 0:
			// use query check sign, use json body to invoke
			params = make(map[string]string, 10)
			ctx.QueryArgs().VisitAll(func(key, value []byte) {
				params[string(key)] = string(value)
			})
		default:
			// json body peek
			params = make(map[string]string, 10)
			err = jsoniter.Unmarshal(ctx.Request.Body(), &params)
			if err != nil {
				glog.Debug(ctx, "json Unmarshal error", glog.String("error", err.Error()), glog.String("raw-body", cast.UnsafeBytesToString(ctx.Request.Body())))
				return ret, berror.NewBizErr(errParams, "body not json")
			}
		}
	}
	if symbol, ok := params["symbol"]; ok {
		glog.Debug(ctx, "symbol save on newV2Checker")
		types.ContextWithRequestSave(ctx, "symbol", symbol)
	}
	apiTimestamp := params["timestamp"]

	apiRecvWindow := params["recv_window"]
	if apiRecvWindow == "" {
		apiRecvWindow = params["recvWindow"]
	}
	apiKey := params["api_key"]
	ctx.Request.Header.Set(constant.HeaderAPIKey, apiKey)
	apiSignature := params["sign"]
	if allowGuest {
		ret[0] = &v2Checker{apiKey: apiKey, apiSignature: apiSignature, remoteIp: ip}
		return
	}

	if apiTimestamp == "" || apiKey == "" || apiSignature == "" {
		err = berror.NewBizErr(errParams, fmt.Sprintf("empty value: apiTimestamp[%s] apiKey[%s] apiSignature[%s]", apiTimestamp, apiKey, apiSignature))
		return
	}

	ret[0] = &v2Checker{
		apiTimestamp:  apiTimestamp,
		apiRecvWindow: apiRecvWindow,
		apiKey:        apiKey,
		apiSignature:  apiSignature,
		params:        params,
		postQueryData: postQueryData,
		postQueryFlag: postUseQuery,
		remoteIp:      ip,
	}

	makerApiKey := params["maker_api_key"]
	makerSign := params["maker_sign"]
	if makerSign != "" && makerApiKey != "" {
		ret[1] = &v2Checker{
			apiTimestamp:  apiTimestamp,
			apiRecvWindow: apiRecvWindow,
			apiKey:        makerApiKey,
			apiSignature:  makerSign,
			params:        params,
			postQueryData: postQueryData,
			postQueryFlag: postUseQuery,
			remoteIp:      ip,
		}
	}

	return
}

func (v2 *v2Checker) GetVersion() string {
	return "v2"
}

// GetAPITimestamp return the api timestamp
func (v2 *v2Checker) GetAPITimestamp() string {
	return v2.apiTimestamp
}

// GetAPIRecvWindow return the api receive window
func (v2 *v2Checker) GetAPIRecvWindow() string {
	return v2.apiRecvWindow
}

// GetAPIKey return the apikey
func (v2 *v2Checker) GetAPIKey() string {
	return v2.apiKey
}

// GetAPISign return the api sign
func (v2 *v2Checker) GetAPISign() string {
	return v2.apiSignature
}

// GetClientIP return the client ip
func (v2 *v2Checker) GetClientIP() string {
	return v2.remoteIp
}

// VerifySign verify the sign
func (v2 *v2Checker) VerifySign(ctx *types.Ctx, signTyp sign.Type, secret string) error {
	err := v2.checkSign(ctx, signTyp, secret, v2.params)
	if err == nil {
		return nil
	}
	if !ctx.IsGet() && !v2.postQueryFlag {
		glog.Debug(ctx, "raw data", glog.String("querystring", cast.UnsafeBytesToString(ctx.URI().QueryString())),
			glog.Any("body", cast.UnsafeBytesToString(ctx.Request.Body())))
		return err
	}

	glog.Debug(ctx, "do double openapi v2 sign check")

	if v2.postQueryFlag {
		if err := v2.checkSign(ctx, signTyp, secret, v2.postQueryData); err != nil {
			return err
		}
		return nil
	}
	// GET request, double check sign use origin query string
	params := queryParse(ctx.URI().QueryString())
	if err := v2.checkSign(ctx, signTyp, secret, params); err != nil {
		return err
	}
	return nil
}

func (v2 *v2Checker) checkSign(ctx *types.Ctx, signTyp sign.Type, secret string, params map[string]string) error {
	signParam := params["sign"]
	delete(params, "sign")
	keyList := make([]string, 0, len(params))
	for k := range params {
		keyList = append(keyList, k)
	}
	sort.Strings(keyList)
	for i, k := range keyList {
		keyList[i] = k + "=" + params[k]
	}
	payload := strings.Join(keyList, "&")

	if err := sign.Verify(signTyp, []byte(secret), []byte(payload), signParam); err != nil {
		glog.Debug(ctx, "verify sign fail, raw data", glog.String("payload", payload),
			glog.String("querystring", cast.UnsafeBytesToString(ctx.URI().QueryString())),
			glog.String("body", cast.UnsafeBytesToString(ctx.Request.Body())), glog.String("content-type", string(ctx.Request.Header.ContentType())))
		if env.IsProduction() {
			return berror.NewBizErr(errorSign, fmt.Sprintf("error sign! origin_string[%s]", payload))
		} else {
			return berror.NewBizErr(errorSign, fmt.Sprintf("error sign! origin_string[%s], sign=%s", payload, signParam))
		}
	}

	return nil
}
