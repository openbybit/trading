package antireplay

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"github.com/hashicorp/go-version"
)

func Init() {
	filter.Register(filter.AntiReplayFilterKey, new())
}

const (
	errParams     = 10001
	errTimestamp  = 10002
	errorSign     = 10004
	errPermission = 10005
)

type antiReplay struct {
	antiReplayMgr AntiReplay
}

// new anti replay filter.
func new() filter.Filter {
	return &antiReplay{
		antiReplayMgr: newAntiReplayMgr(),
	}
}

func (a *antiReplay) GetName() string {
	return filter.AntiReplayFilterKey
}

// Do will do anti replay
func (a *antiReplay) Do(next types.Handler) types.Handler {
	return func(c *types.Ctx) error {
		if !a.antiReplayMgr.EnableSign() {
			glog.Debug(c, "disable anti replay sign")
			return next(c)
		}

		now := time.Now()

		md := metadata.MDFromContext(c)
		newVersion, err := version.NewVersion(md.Extension.AppVersion)
		if err != nil {
			return err
		}

		s, err := a.getParams(c)
		if err != nil {
			return err
		}

		enableAntiReplay := a.antiReplayMgr.EnableAntiReplay()
		glog.Debug(c, "anti replay switch", glog.Bool("replay", enableAntiReplay))
		if enableAntiReplay {
			diff, fastThanServer := a.antiReplayMgr.GetAntiReplayDiffTime()
			if err = s.antiReplay(c, diff, fastThanServer); err != nil {
				return err
			}
		}

		secret, err := a.antiReplayMgr.VerifyAccessKey(c, md.Extension.AppName, md.Extension.Platform, s.accessKey, newVersion)
		if err != nil {
			return berror.NewBizErr(errPermission, "auth request permission denied")
		}

		if err = s.verifySign(c, secret); err != nil {
			return err
		}
		glog.Debug(c, "anti replay cost", glog.Duration("cost", time.Since(now)))

		return next(c)
	}
}

// Init will init the anti replay filter
func (a *antiReplay) Init(ctx context.Context, args ...string) error {
	if len(args) == 0 {
		return nil
	}
	if err := a.antiReplayMgr.Init(ctx); err != nil {
		return err
	}

	return nil
}

func (a *antiReplay) getParams(c *types.Ctx) (*params, error) {
	requestTime := string(c.Request.Header.Peek(XBTimestamp))
	accessKey := string(c.Request.Header.Peek(XBAccessKey))
	sign := string(c.Request.Header.Peek(XBSignature))
	if requestTime == "" || sign == "" || accessKey == "" {
		return nil, berror.NewBizErr(errParams, fmt.Sprintf("replay params error, time[%s],key[%s],sign[%s]", requestTime, accessKey, sign))
	}

	var payload string
	if c.IsGet() {
		data := make(map[string]string)
		keyList := make([]string, 0, 5)
		c.QueryArgs().VisitAll(func(key, value []byte) {
			k := string(key)
			if _, ok := data[k]; ok {
				return
			}
			data[k] = string(value)
			keyList = append(keyList, k)
		})
		sort.Strings(keyList)
		for i, k := range keyList {
			keyList[i] = k + "=" + data[k]
		}
		val := strings.Join(keyList, "&")
		payload = util.Base64Encode(val)
	} else {
		payload = util.Base64EncodeByte(c.Request.Body())
	}

	return &params{requestTime: requestTime, accessKey: accessKey, sign: sign, payload: payload}, nil
}

type params struct {
	requestTime string
	accessKey   string
	sign        string
	payload     string // base64
}

func (p *params) antiReplay(ctx context.Context, diff, fastThanServer int64) error {
	current := time.Now().UnixMilli()

	requestTime, err := strconv.ParseInt(p.requestTime, 10, 64)
	if err != nil {
		return berror.NewBizErr(errParams, "antiReplay request time is invalid")
	}

	if current-requestTime > diff || requestTime-current > fastThanServer {
		glog.Debug(ctx, "antiReplay timestamp", glog.Int64("current", current), glog.Int64("requestTime", requestTime))
		return berror.NewBizErr(errTimestamp, "invalid request, please check your server timestamp")
	}

	return nil
}

func (p *params) verifySign(c *types.Ctx, secret string) error {
	var buf bytes.Buffer
	buf.WriteString(p.accessKey)
	buf.WriteString(p.payload)
	buf.WriteString(p.requestTime)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(buf.Bytes())
	glog.Debug(c, "payload", glog.String("payload", cast.UnsafeBytesToString(buf.Bytes())))
	sign := hex.EncodeToString(mac.Sum(nil))

	if !strings.EqualFold(sign, p.sign) {
		glog.Debug(c, "raw data", glog.Any("querystring", cast.UnsafeBytesToString(c.URI().QueryString())),
			glog.Any("body", cast.UnsafeBytesToString(c.Request.Body())))
		return berror.NewBizErr(errorSign, fmt.Sprintf("error sign! origin_string[%s]", cast.UnsafeBytesToString(buf.Bytes())))
	}
	return nil
}
