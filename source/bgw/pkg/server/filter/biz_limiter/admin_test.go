package biz_limiter

import (
	"bgw/pkg/common/constant"
	"code.bydev.io/fbu/gateway/gway.git/gapp"
	"context"
	"errors"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/tj/assert"
	"reflect"
	"testing"
)

func TestOnGetQuotaWithKey(t *testing.T) {
	resp, err := onGetQuotaWithKey(gapp.AdminArgs{
		Params: []string{""},
	})
	assert.Equal(t, "failed", resp)
	assert.Error(t, err, "param app cannot be empty")

	resp, err = onGetQuotaWithKey(gapp.AdminArgs{
		Params: []string{"123"},
	})
	assert.Equal(t, "failed", resp)
	assert.Error(t, err, "param invalid:app")

	p := gomonkey.ApplyFuncReturn(queryFutureQuota, 100, nil)

	resp, err = onGetQuotaWithKey(gapp.AdminArgs{
		Params: []string{constant.AppTypeFUTURES},
	})
	assert.Equal(t, 100, resp)
	assert.NoError(t, err)

	p.ApplyFuncReturn(querySpotQuota, 100, nil)

	resp, err = onGetQuotaWithKey(gapp.AdminArgs{
		Params: []string{constant.AppTypeSPOT},
	})
	assert.Equal(t, 100, resp)
	assert.NoError(t, err)

	p.ApplyFuncReturn(queryOptionQuota, 100, nil)

	resp, err = onGetQuotaWithKey(gapp.AdminArgs{
		Params: []string{constant.AppTypeOPTION},
	})
	assert.Equal(t, 100, resp)
	assert.NoError(t, err)

	p.Reset()
}

func TestQueryOptionQuota(t *testing.T) {
	qv2 := newQuotaLoader("123")
	p := gomonkey.ApplyFunc(newQuotaLoader, func(app string) *quotaLoader {
		assert.Equal(t, "123", app)
		return qv2
	})
	p.ApplyPrivateMethod(reflect.TypeOf(qv2), "init", func(ctx context.Context) error {
		return errors.New("ddd")
	})
	f, err := queryOptionQuota(gapp.AdminArgs{
		Params: []string{"123", "1000", "ggg", "122"},
	})
	assert.EqualError(t, err, "ddd")
	assert.Equal(t, "failed", f)
	p.Reset()
	qv2 = newQuotaLoader("123")
	p = p.ApplyFunc(newQuotaLoader, func(app string) *quotaLoader {
		assert.Equal(t, "123", app)
		return qv2
	})
	p.ApplyPrivateMethod(reflect.TypeOf(qv2), "init", func(ctx context.Context) error {
		return nil
	})
	p.ApplyPrivateMethod(reflect.TypeOf(qv2), "getQuota", func(ctx context.Context, uid int64, group string) int {
		return 100
	})
	f, err = queryOptionQuota(gapp.AdminArgs{
		Params: []string{"123", "1000", "ggg", "122", "22"},
	})
	assert.NoError(t, err)
	assert.Equal(t, 100, f)
	p.Reset()
}

func TestQuerySpotQuota(t *testing.T) {
	qv2 := newQuotaLoaderV2("123")
	p := gomonkey.ApplyFunc(newQuotaLoaderV2, func(app string) *quotaLoaderV2 {
		assert.Equal(t, "123", app)
		return qv2
	})
	p.ApplyPrivateMethod(reflect.TypeOf(qv2), "init", func(ctx context.Context) error {
		return errors.New("ddd")
	})
	f, err := querySpotQuota(gapp.AdminArgs{
		Params: []string{"123", "1000", "ggg", "122"},
	})
	assert.EqualError(t, err, "ddd")
	assert.Equal(t, "failed", f)
	p.Reset()
	qv2 = newQuotaLoaderV2("123")
	p = p.ApplyFunc(newQuotaLoaderV2, func(app string) *quotaLoaderV2 {
		assert.Equal(t, "123", app)
		return qv2
	})
	p.ApplyPrivateMethod(reflect.TypeOf(qv2), "init", func(ctx context.Context) error {
		return nil
	})
	p.ApplyPrivateMethod(reflect.TypeOf(qv2), "getQuota", func(ctx context.Context, pp *rateParams) int {
		return 100
	})
	f, err = querySpotQuota(gapp.AdminArgs{
		Params: []string{"123", "1000", "ggg", "122", "22"},
	})
	assert.NoError(t, err)
	assert.Equal(t, 100, f)
	p.Reset()
}

func TestQuerySpotQuota2(t *testing.T) {
	qv2 := newQuotaLoaderV2("123")
	p := gomonkey.ApplyFunc(newQuotaLoaderV2, func(app string) *quotaLoaderV2 {
		assert.Equal(t, "123", app)
		return qv2
	})
	p.ApplyPrivateMethod(reflect.TypeOf(qv2), "init", func(ctx context.Context) error {
		return nil
	})
	p.ApplyPrivateMethod(reflect.TypeOf(qv2), "getQuota", func(ctx context.Context, pp *rateParams) int {
		return 100
	})
	f, err := querySpotQuota(gapp.AdminArgs{
		Params: []string{"123", "1000", "ggg", "122", "22"},
	})
	assert.NoError(t, err)
	assert.Equal(t, 100, f)
	p.Reset()
}

func TestQueryFutureQuota(t *testing.T) {
	p := gomonkey.ApplyFuncReturn(NewQuotaManager, nil, errors.New("eee"))
	f, err := queryFutureQuota(gapp.AdminArgs{})
	assert.EqualError(t, err, "eee")
	assert.Equal(t, "failed", f)
	p.Reset()

	p.ApplyFuncReturn(NewQuotaManager, mockQuotaManager{q: 0, err: errors.New("eee")}, nil)
	f, err = queryFutureQuota(gapp.AdminArgs{})
	assert.EqualError(t, err, "eee")
	assert.Equal(t, "failed", f)
	p.Reset()

	p.ApplyFuncReturn(NewQuotaManager, mockQuotaManager{
		qFn: func(_ context.Context, uid int64, app, group, symbol string) (int64, error) {
			assert.Equal(t, int64(1000), uid)
			assert.Equal(t, "11", app)
			assert.Equal(t, "ggg", group)
			assert.Equal(t, "122", symbol)
			return 100, nil
		}}, nil)
	f, err = queryFutureQuota(gapp.AdminArgs{
		Params: []string{"11", "1000", "ggg", "122"},
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(100), f)
	p.Reset()

}

type mockQuotaManager struct {
	q   int64
	err error
	qFn func(_ context.Context, _ int64, _, _, _ string) (int64, error)
}

func (m mockQuotaManager) GetQuota(ctx context.Context, a int64, b, c, d string) (int64, error) {
	if m.qFn != nil {
		return m.qFn(ctx, a, b, c, d)
	}
	return m.q, m.err
}
