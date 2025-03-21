package openapi

import (
	"bytes"
	"fmt"

	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gcore/sign"
	"code.bydev.io/fbu/gateway/gway.git/glog"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
)

type v3Checker struct {
	apiTimestamp  string
	apiRecvWindow string
	apiKey        string
	apiSignature  string
	apiPayload    []byte
	remoteIp      string
}

func newV3Checker(ctx *types.Ctx, apiKey, ip string, allowGuest, isWss bool) (ret [2]Checker, err error) {
	var (
		apiPayload     []byte
		apiTimestamp   = string(ctx.Request.Header.Peek(constant.HeaderAPITimeStamp))
		apiRecvWindow  = string(ctx.Request.Header.Peek(constant.HeaderAPIRecvWindow))
		apiSignature   = string(ctx.Request.Header.Peek(constant.HeaderAPISign))
		makerSignature = string(ctx.Request.Header.Peek(constant.HeaderAPIMAKERSIGN))   // block_trade
		makerAPIKey    = string(ctx.Request.Header.Peek(constant.HeaderAPIMAKERAPIKEY)) // block_trade
	)

	if !ctx.IsGet() && !ctx.IsPost() {
		err = berror.WithMessage(berror.ErrParams, "method error")
		return
	}

	if allowGuest {
		ret[0] = &v3Checker{apiKey: apiKey, apiSignature: apiSignature, remoteIp: ip}
		return
	}

	if apiTimestamp == "" || apiKey == "" {
		err = berror.WithMessage(berror.ErrParams, fmt.Sprintf("empty value: apiTimestamp[%s] apiKey[%s]", apiTimestamp, apiKey))
		return
	}
	if !isWss && apiSignature == "" {
		err = berror.WithMessage(berror.ErrParams, fmt.Sprintf("empty value: apiTimestamp[%s] apiKey[%s] apiSignature[%s]", apiTimestamp, apiKey, apiSignature))
		return
	}

	if ctx.IsGet() {
		apiPayload = ctx.URI().QueryString()
	} else {
		apiPayload = ctx.Request.Body()
	}

	ret[0] = &v3Checker{apiTimestamp, apiRecvWindow,
		apiKey, apiSignature, apiPayload, ip}

	if makerSignature != "" && makerAPIKey != "" {
		ret[1] = &v3Checker{apiTimestamp, apiRecvWindow,
			makerAPIKey, makerSignature, apiPayload, ip}
	}

	return
}

func (v3 *v3Checker) GetVersion() string {
	return "v3"
}

// GetAPITimestamp return the api timestamp
func (v3 *v3Checker) GetAPITimestamp() string {
	return v3.apiTimestamp
}

// GetAPIRecvWindow return the api receive window
func (v3 *v3Checker) GetAPIRecvWindow() string {
	return v3.apiRecvWindow
}

// GetAPIKey return the apikey
func (v3 *v3Checker) GetAPIKey() string {
	return v3.apiKey
}

// GetAPISign return the api sign
func (v3 *v3Checker) GetAPISign() string {
	return v3.apiSignature
}

// GetClientIP return the client ip
func (v3 *v3Checker) GetClientIP() string {
	return v3.remoteIp
}

// VerifySign verify the sign
func (v3 *v3Checker) VerifySign(ctx *types.Ctx, signTyp sign.Type, secret string) error {
	var buf bytes.Buffer
	buf.WriteString(v3.apiTimestamp)
	buf.WriteString(v3.apiKey)
	buf.WriteString(v3.apiRecvWindow)
	buf.Write(v3.apiPayload)

	payload := buf.Bytes()
	signParam := v3.apiSignature
	if err := sign.Verify(signTyp, []byte(secret), payload, signParam); err != nil {
		glog.Debugf(ctx, "verify sign fail, payload: %s, sign: %s, secret: %s", payload, v3.apiSignature, secret)
		if env.IsProduction() {
			return berror.NewBizErr(errorSign, fmt.Sprintf("error sign! origin_string[%s]", payload))
		} else {
			return berror.NewBizErr(errorSign, fmt.Sprintf("error sign! origin_string[%s], sign=%s", payload, signParam))
		}
	}

	return nil
}
