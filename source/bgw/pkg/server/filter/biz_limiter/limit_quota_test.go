package biz_limiter

import (
	"context"
	"errors"
	"log"
	"reflect"
	"testing"
	"time"

	"bgw/pkg/diagnosis"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/gkafka"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/frameworks/byone/zrpc"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/coocood/freecache"
	jsoniter "github.com/json-iterator/go"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tj/assert"

	"bgw/pkg/common/util"
	"bgw/pkg/server/filter"
)

func TestCommon(t *testing.T) {

	lqOnErr(&gkafka.ConsumerError{Err: errors.New("111")})
	q := newQuotaLoaderV2("123")
	assert.Equal(t, reflect.TypeOf(observer.DefaultEvent{}), q.GetEventType())

	s := q.getUidKey(100)
	assert.Equal(t, "/BGW_data/quota_v2/123:100", s)
}

func TestSaveQuota(t *testing.T) {

	qm, err := NewQuotaManager(context.Background())
	assert.NoError(t, err)
	qq := qm.(*quotaManager)
	qq.saveQuota(context.Background(), "123", 100)
	assert.Equal(t, int64(1), qq.quotaCache.EntryCount())
}

func TestNewQuotaManager(t *testing.T) {
	gmetric.Init("TestNewQuotaManager")

	Convey("test new quota manager", t, func() {
		qm, err := NewQuotaManager(context.Background())
		So(err, ShouldBeNil)
		So(qm, ShouldNotBeNil)

		p := gomonkey.ApplyFuncReturn(zrpc.NewClient, nil, errors.New("asas"))
		defer p.Reset()

		quotaMgr = nil
		doNewQuotaManager(context.Background())
		So(quotaMgr, ShouldBeNil)
		quotaMgr = qm.(*quotaManager)

		quotaMgr = nil
		_, _ = NewQuotaManager(context.Background())
		quotaMgr = qm.(*quotaManager)
	})

}

func TestLimiterMemo_Limit(t *testing.T) {
	ff := newLimiterMemo()

	assert.Equal(t, filter.BizRateLimitFilterMEMO, ff.GetName())

	f := ff.(*limiterMemo)
	assert.Equal(t, 0, len(f.limiters))
}

func TestHandleQuotaMessage(t *testing.T) {
	qm := &quotaManager{
		quotaCache: freecache.NewCache(100),
	}

	qm.HandleQuotaMessage(context.Background(), &gkafka.Message{})
	assert.Equal(t, int64(0), qm.quotaCache.EntryCount())

	rs := RateLimitSignal{
		APP:     "",
		UserID:  0,
		Changes: nil,
	}
	rss, _ := util.JsonMarshal(rs)
	qm.HandleQuotaMessage(context.Background(), &gkafka.Message{
		Value: rss,
	})
	assert.Equal(t, int64(0), qm.quotaCache.EntryCount())

	rs = RateLimitSignal{
		APP:     "",
		UserID:  1000,
		Changes: nil,
	}
	rss, _ = util.JsonMarshal(rs)
	qm.HandleQuotaMessage(context.Background(), &gkafka.Message{
		Value: rss,
	})
	assert.Equal(t, int64(0), qm.quotaCache.EntryCount())

	rs = RateLimitSignal{
		APP:    "--",
		UserID: 100,
		Changes: map[string][]rateLimit{
			"A": {{Symbol: BTCUSD}, {Symbol: v3Symbol}},
		},
	}
	rss, _ = util.JsonMarshal(rs)
	qm.HandleQuotaMessage(context.Background(), &gkafka.Message{
		Value: rss,
	})
	assert.Equal(t, int64(2), qm.quotaCache.EntryCount())
}

func TestQuotaManager_GetQuota(t *testing.T) {
	Convey("test get quota", t, func() {
		qm := &quotaManager{
			ctx:        context.Background(),
			quotaCache: freecache.NewCache(mixerCacheSize * 100 * 100),
		}

		_, err := qm.GetQuota(context.Background(), -1, "", "", "")
		So(err, ShouldNotBeNil)
		patch := gomonkey.ApplyFunc(gmetric.ObserveDefaultLatencySince, func(t time.Time, typ, label string) {})
		defer patch.Reset()
		key := qm.getKey(666, "futures", "order", "usdt")
		qm.saveQuota(context.Background(), key, 888)
		_, err = qm.GetQuota(context.Background(), 666, "futures", "order", "usdt")
		So(err, ShouldBeNil)

		msg := &gkafka.Message{}
		qm.HandleQuotaMessage(context.Background(), msg)

		rate := RateLimitSignal{
			APP:     "futures",
			UserID:  0,
			Changes: make(map[string][]rateLimit),
		}
		d, _ := jsoniter.Marshal(rate)
		msg.Value = d
		qm.HandleQuotaMessage(context.Background(), msg)

		rate.UserID = 1
		rate.Changes["test"] = []rateLimit{{}, {Symbol: "usdt"}}
		d, _ = jsoniter.Marshal(rate)
		msg.Value = d
		qm.HandleQuotaMessage(context.Background(), msg)

		lqOnErr(&gkafka.ConsumerError{})
	})
}

func TestMixerDiagnosis(t *testing.T) {
	Convey("MixerDiagnosis", t, func() {

		result := diagnosis.NewResult(errors.New("xxx"))

		p := gomonkey.ApplyFuncReturn(diagnosis.DiagnoseKafka, result)
		defer p.Reset()
		p.ApplyFuncReturn(diagnosis.DiagnoseGrpcDependency, result)
		p.ApplyFuncReturn(diagnosis.DiagnoseRedis, result)
		p.ApplyFuncReturn(diagnosis.DiagnoseEtcd, result)

		dig := quotaDiagnose{}
		So(dig.Key(), ShouldEqual, mixerOpenapi)
		r, err := dig.Diagnose(context.Background())
		resp := r.(map[string]interface{})
		So(resp, ShouldNotBeNil)
		So(resp["kafka"], ShouldEqual, result)
		So(resp["grpc"], ShouldEqual, result)
		So(resp["redis"], ShouldEqual, result)
		So(resp["etcd"], ShouldEqual, result)
		So(err, ShouldBeNil)
	})
}

func TestQuotaManager_GetQuota2(t *testing.T) {
	Convey("test GetQuota", t, func() {
		q := &quotaManager{}
		q.kafkaOnce.Do(func() { log.Println("do nothing") })
		q.quotaCache = freecache.NewCache(500)

		patch := gomonkey.ApplyFunc((*quotaManager).queryQuota,
			func(q *quotaManager, ctx context.Context, uid int64, group, symbol string) (int64, bool) {
				if uid == 1 {
					return 123, false
				}
				return 123, true
			})
		defer patch.Reset()

		rate, err := q.GetQuota(context.Background(), 1, "app", "group", "symbol")
		So(rate, ShouldEqual, 123)
		So(err, ShouldBeNil)

		rate, err = q.GetQuota(context.Background(), 2, "app", "group", "symbol")
		So(rate, ShouldEqual, 123)
		So(err, ShouldBeNil)
	})
}
