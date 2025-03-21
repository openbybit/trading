package biz_limiter

import (
	"bgw/pkg/common/constant"
	"bgw/pkg/test"
	"code.bydev.io/fbu/gateway/gway.git/gredis"
	"context"
	"errors"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/tj/assert"
	"testing"
	"time"
)

func TestGetAutoQuota(t *testing.T) {

	rctx, md := test.NewReqCtx()
	r := limitRule{
		Limit: gredis.Limit{},
		extractor: extractor{
			UID: false,
		},
	}
	l, s, err := getAutoQuota(rctx, "xxx", md, r)

	// !rule.UID
	assert.NoError(t, err)
	assert.Equal(t, 0, l.Rate)
	assert.Equal(t, 0, l.Burst)
	assert.Equal(t, time.Duration(0), l.Period)
	assert.Equal(t, "xxx", s)

	// !rule.EnableCustomRate
	r = limitRule{
		Limit: gredis.Limit{},
		extractor: extractor{
			UID: true,
		},
		EnableCustomRate: false,
	}
	l, s, err = getAutoQuota(rctx, "xxx", md, r)
	assert.NoError(t, err)
	assert.Equal(t, 0, l.Rate)
	assert.Equal(t, 0, l.Burst)
	assert.Equal(t, time.Duration(0), l.Period)
	assert.Equal(t, "xxx:0", s)

	// constant.AppTypeFUTURES
	r = limitRule{
		Limit: gredis.Limit{},
		extractor: extractor{
			UID: true,
		},
		EnableCustomRate: true,
	}
	rateLimitMgr = mockQuotaMgr{
		q: 100,
		e: nil,
	}
	p := gomonkey.ApplyGlobalVar(&rateLimitMgr, rateLimitMgr)
	l, s, err = getAutoQuota(rctx, constant.AppTypeFUTURES, md, r)
	assert.NoError(t, err)
	assert.Equal(t, 100, l.Rate)
	assert.Equal(t, 0, l.Burst)
	assert.Equal(t, constant.AppTypeFUTURES+"::0", s)

	l, s, err = getAutoQuota(rctx, "asas", md, r)
	assert.Equal(t, 0, l.Rate)
	assert.Equal(t, 0, l.Burst)
	assert.EqualError(t, err, "unkown app: asas, 0")

	p.Reset()
	rateLimitMgr = mockQuotaMgr{
		q: 20,
		e: errors.New("xxxxxx"),
	}
	p.ApplyGlobalVar(&rateLimitMgr, rateLimitMgr)
	l, s, err = getAutoQuota(rctx, constant.AppTypeFUTURES, md, r)
	assert.NoError(t, err)
	assert.Equal(t, 0, l.Rate)
	assert.Equal(t, 0, l.Burst)
	assert.Equal(t, constant.AppTypeFUTURES+"::0", s)

	p.Reset()

	rateLimitMgr = mockQuotaMgr{
		q: 20,
		e: nil,
	}
	p.ApplyGlobalVar(&rateLimitMgr, rateLimitMgr)
	l, s, err = getAutoQuota(rctx, constant.AppTypeSPOT, md, r)
	assert.NoError(t, err)
	assert.Equal(t, 0, l.Rate)
	assert.Equal(t, 0, l.Burst)
	assert.Equal(t, constant.AppTypeSPOT+":0", s)

	//m := sync.Map{}
	//m.Store(constant.AppTypeSPOT, &quotaLoaderV2{})
	//p.ApplyGlobalVar(&limitV2Loaders, &m)
	//l, s, err = getAutoQuota(rctx, constant.AppTypeSPOT, md, r)
	//assert.NoError(t, err)
	//assert.Equal(t, 0, l.Rate)
	//assert.Equal(t, 0, l.Burst)
	//assert.Equal(t, constant.AppTypeSPOT+":0", s)

	p.Reset()

}

type mockQuotaMgr struct {
	q int64
	e error
}

func (m mockQuotaMgr) GetQuota(ctx context.Context, uid int64, app, group, symbol string) (int64, error) {
	return m.q, m.e
}
