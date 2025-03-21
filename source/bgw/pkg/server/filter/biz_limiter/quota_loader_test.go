package biz_limiter

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/coocood/freecache"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/tj/assert"

	"bgw/pkg/common/util"
	"bgw/pkg/config_center/etcd"
)

func TestCommonQuota(t *testing.T) {
	qm := newQuotaLoader("123")
	assert.Equal(t, reflect.TypeOf(observer.DefaultEvent{}), qm.GetEventType())
	assert.Equal(t, -1, qm.GetPriority())

	ql := newQuotaLoaderV2("option")
	assert.Equal(t, "/BGW_data/quota_v2/xxx", ql.getKey("xxx"))
}

func TestDeleteCache(t *testing.T) {
	qm := newQuotaLoader("123")

	quotaCache = freecache.NewCache(1)
	k := qm.getCacheKey(100, "")
	quotaCache.Set(k, []byte("123"), 1000000000)
	qm.deleteCache(&quotaValue{UID: 100})
	assert.Equal(t, int64(0), quotaCache.EntryCount())
}

func BenchmarkQuotaLoaderGet(b *testing.B) {
	ql := newQuotaLoaderV2("option")
	_ = ql.init(context.Background())
	qv := &quotaValue{
		UID:    12345,
		Quota:  111,
		AID:    12345,
		Symbol: "BTCUSD",
	}
	value, _ := util.JsonMarshal(qv)
	err := ql.configure.Put(context.Background(), "12345", string(value))
	assert.Nil(b, err)
	for i := 0; i < b.N; i++ {
		ql.getQuotaWithKey(context.TODO(), "12345")
	}
}

func TestQuotaLoaderGet(t *testing.T) {
	gmetric.Init("test")
	ql := newQuotaLoaderV2("option")
	_ = ql.init(context.Background())
	qv := &quotaValue{
		UID:    12345,
		Quota:  111,
		AID:    12345,
		Symbol: "BTCUSD",
	}
	value, _ := util.JsonMarshal(qv)
	err := ql.configure.Put(context.Background(), "12345", string(value))
	assert.Nil(t, err)

	v, b := ql.getQuotaWithKey(context.TODO(), "12345")
	assert.True(t, b)
	assert.EqualValues(t, qv.Quota, v)

	t.Log(v)
}

func TestGetUnifiedQuota(t *testing.T) {
	Convey("test getUnifiedQuota", t, func() {
		res := getUnifiedQuota(context.Background(), 123, "group")
		So(res, ShouldEqual, 0)

		ql := &quotaLoader{}
		key := ql.getCacheKey(123, "group")
		quotaCache = freecache.NewCache(10000)
		_ = quotaCache.Set(key, []byte("0"), 0)
		limiterRules.Store(optionService, ql)
		res = getUnifiedQuota(context.Background(), 123, "group")
		So(res, ShouldEqual, 0)

		limiterRules.Store(optionService, "123")
		res = getUnifiedQuota(context.Background(), 123, "group")
		So(res, ShouldEqual, 0)
	})
}

func TestQuotaLoaderV2_getQuotaWithKey(t *testing.T) {
	Convey("test getQuotaWithKey", t, func() {
		patch := gomonkey.ApplyFunc(gmetric.ObserveDefaultLatencySince, func(t time.Time, typ, label string) {})
		defer patch.Reset()

		ql2 := &quotaLoaderV2{}
		quotaCacheV2 = freecache.NewCache(500)
		_ = quotaCacheV2.Set([]byte("key"), noExistValue, 200)
		q, b := ql2.getQuotaWithKey(context.Background(), "key")
		So(q, ShouldEqual, 0)
		So(b, ShouldBeFalse)

		_ = quotaCacheV2.Set([]byte("key"), []byte("12"), 200)
		q, b = ql2.getQuotaWithKey(context.Background(), "key")
		So(q, ShouldEqual, 12)
		So(b, ShouldBeTrue)

		ql2.configure = &mockCfg{}
		q, b = ql2.getQuotaWithKey(context.Background(), "key1")
		So(q, ShouldEqual, 0)
		So(b, ShouldBeFalse)

		q, b = ql2.getQuotaWithKey(context.Background(), "key2")
		So(q, ShouldEqual, 0)
		So(b, ShouldBeFalse)

		q, b = ql2.getQuotaWithKey(context.Background(), "key3")
		So(q, ShouldEqual, 0)
		So(b, ShouldBeFalse)
	})
}

// func TestQuotaLoaderV2_GetQuotaByUid(t *testing.T) {
// 	Convey("test QuotaLoaderV2 GetQuotaByUid", t, func() {
// 		ql2 := &quotaLoaderV2{}
// 		ql2.getQuotaByUid(context.Background())
// 	})
// }

type Configure interface {
	Listen(ctx context.Context, key string, listener observer.EventListener) error
	Get(ctx context.Context, key string) (string, error)
	GetChildren(ctx context.Context, key string) ([]string, []string, error)
	Put(ctx context.Context, key, value string) error
	Del(ctx context.Context, key string) error
}

type mockCfg struct{}

func (m *mockCfg) Listen(ctx context.Context, key string, listener observer.EventListener) error {
	return nil
}

func (m *mockCfg) Get(ctx context.Context, key string) (string, error) {
	if key == "key1" {
		return "", etcd.ErrKVPairNotFound
	}

	if key == "key2" {
		return "", errors.New("mock err")
	}

	return "{12", nil
}

func (m *mockCfg) GetChildren(ctx context.Context, key string) ([]string, []string, error) {
	return nil, nil, nil
}

func (m *mockCfg) Put(ctx context.Context, key, value string) error {
	return nil
}

func (m *mockCfg) Del(ctx context.Context, key string) error {
	return nil
}
