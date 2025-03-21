package tradingroute

import (
	"context"
	"errors"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/fbu/gateway/gway.git/gnacos"
	"code.bydev.io/frameworks/byone/core/threading"
	"code.bydev.io/frameworks/byone/zrpc"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
	"github.com/tj/assert"

	"bgw/pkg/common/berror"
	"bgw/pkg/diagnosis"
	"bgw/pkg/remoting/nacos"

	routev1 "code.bydev.io/fbu/future/bufgen.git/pkg/bybit/uta/route/v1"
	"code.bydev.io/fbu/gateway/gway.git/gconfig"
	"code.bydev.io/frameworks/byone/core/syncx"
	"github.com/coocood/freecache"
	"google.golang.org/grpc"

	"bgw/pkg/config"
)

func TestFreshCache(t *testing.T) {
	gmetric.Init("TestAsyncFreshLoop")

	r := newRouting(routerTypeCommon).(*routing)
	out := &routeCache{}
	r.freshCache(&GetRouteRequest{
		UserId: 100,
	}, 0, out)
	_, ok := r.freshMap.Load(int64(100))
	assert.Equal(t, false, ok)
	assert.Equal(t, false, out.IsAioUser)
	assert.Equal(t, "", out.DataId)
	assert.Equal(t, int64(0), out.ExpireTime)
	out = &routeCache{}
	r.freshCache(&GetRouteRequest{
		UserId: 100,
	}, time.Now().Unix()+1000000, out)
	_, ok = r.freshMap.Load(int64(100))
	assert.Equal(t, false, ok)
	assert.Equal(t, false, out.IsAioUser)
	assert.Equal(t, "", out.DataId)
	assert.Equal(t, int64(0), out.ExpireTime)

	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(r), "invokeRouting", func(ctx context.Context, in *GetRouteRequest, cacheKey []byte) (*routev1.GetRouteResponse, int64, error) {
		return &routev1.GetRouteResponse{
			DataId:   "123",
			IsOldUta: false,
		}, 0, nil
	})
	out = &routeCache{}
	r.freshCache(&GetRouteRequest{
		UserId: 100,
	}, time.Now().Unix()-1000, out)
	_, ok = r.freshMap.Load(int64(100))
	assert.Equal(t, false, ok)
	assert.Equal(t, true, out.IsAioUser)
	assert.Equal(t, "123", out.DataId)
	assert.Equal(t, int64(0), out.ExpireTime)

	out = &routeCache{}
	p.ApplyPrivateMethod(reflect.TypeOf(r), "invokeRouting", func(ctx context.Context, in *GetRouteRequest, cacheKey []byte) (*routev1.GetRouteResponse, int64, error) {
		return nil, 0, errors.New("123")
	})
	r.freshCache(&GetRouteRequest{
		UserId: 100,
	}, time.Now().Unix()-30, out)
	_, ok = r.freshMap.Load(int64(100))
	assert.Equal(t, true, ok)
	assert.Equal(t, false, out.IsAioUser)
	assert.Equal(t, "", out.DataId)
	assert.Equal(t, int64(0), out.ExpireTime)

	p.Reset()
}

// func TestAsyncFreshLoop(t *testing.T) {
//	gmetric.Init("TestAsyncFreshLoop")
//	r := newRouting(routerTypeCommon).(*routing)
//	r.freshCh <- &freshMessage{UserId: 100}
//	r.freshMap.Store(int64(100), "123")
//
//	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(r), "invokeRouting", func(ctx context.Context, in *GetRouteRequest, cacheKey []byte) (*routev1.GetRouteResponse, int64, error) {
//		return nil, 0, nil
//	})
//	go func() {
//		time.Sleep(time.Second * 3)
//		close(r.freshCh)
//	}()
//	r.asyncFreshLoop()
//	_, ok := r.freshMap.Load(int64(100))
//	assert.Equal(t, false, ok)
//	p.Reset()
// }

func TestParse(t *testing.T) {
	gmetric.Init("TestTradingRouteParse")

	r := newRouting(routerTypeCommon).(*routing)
	info, err := r.parse("", "1212")
	assert.Equal(t, "1212", info.Addr)
	assert.NoError(t, err)

	info, err = r.parse("", "{1212")
	assert.Nil(t, info)
	assert.EqualError(t, err, "unmarshal route fail, error: invalid character '1' looking for beginning of object key string, key: , value: {1212")
}

func TestInitConfigClientErr(t *testing.T) {
	r := newRouting(routerTypeCommon).(*routing)

	p := gomonkey.ApplyFuncReturn(nacos.GetNacosConfig, nil, errors.New("asas"))
	r.initConfigClient()
	assert.Nil(t, r.configClient)

	p.ApplyFuncReturn(nacos.GetNacosConfig, nil, nil)
	p.ApplyFuncReturn(gnacos.NewConfigClient, nil, errors.New("asas"))
	r.initConfigClient()
	assert.Nil(t, r.configClient)
	p.Reset()
}

func TestInitRouteClientErr(t *testing.T) {
	r := newRouting(routerTypeCommon).(*routing)

	p := gomonkey.ApplyFuncReturn(zrpc.NewClient, nil, errors.New("asas"))
	r.initRouteClient()
	assert.Nil(t, r.routeClient)
	p.Reset()
}

func TestGetEndpointErr(t *testing.T) {
	r := newRoutingMock()
	r.configClient = nil
	_, err := r.GetEndpoint(context.Background(), nil)
	assert.Equal(t, berror.ErrDefault, err)
	r = newRoutingMock()
	_, err = r.GetEndpoint(context.Background(), &GetRouteRequest{UserId: 0})
	assert.Equal(t, berror.ErrParams, err)

	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(r), "getDataIdFromRouting", func(ctx context.Context, in *GetRouteRequest) (Endpoint, error) {
		return Endpoint{}, errors.New("ddd")
	})
	_, err = r.GetEndpoint(context.Background(), &GetRouteRequest{UserId: 10})
	assert.EqualError(t, err, "GetRoute fail, err: ddd")

	p.ApplyPrivateMethod(reflect.TypeOf(r), "getDataIdFromRouting", func(ctx context.Context, in *GetRouteRequest) (Endpoint, error) {
		return Endpoint{}, nil
	})
	p.ApplyPrivateMethod(reflect.TypeOf(r), "getInstance", func(ctx context.Context, dataId string) (string, error) {
		return "", errors.New("ddd")
	})
	_, err = r.GetEndpoint(context.Background(), &GetRouteRequest{UserId: 10})
	assert.EqualError(t, err, "getInstance fail: dataId=, err=ddd")

	p.Reset()
}

func TestGetEndpoint(t *testing.T) {

	convey.Convey("test GetEndpoint", t, func() {
		r := newRoutingMock()
		r.namespace = config.GetNamespace()
		convey.Convey("test getEndpoint", func() {
			endpoint, err := r.GetEndpoint(context.Background(), &GetRouteRequest{UserId: 123})
			convey.So(err, convey.ShouldBeNil)
			convey.So(endpoint.Address, convey.ShouldEqual, "172.17.15.246:28002")
		})

		convey.Convey("getEndpoint from cache", func() {
			endpoint, err := r.GetEndpoint(context.Background(), &GetRouteRequest{UserId: 123})
			convey.So(err, convey.ShouldBeNil)
			convey.So(endpoint.Address, convey.ShouldEqual, "172.17.15.246:28002")
		})
	})

	convey.Convey("concurrent call GetEndpoint", t, func() {
		convey.Convey("test getEndpoint", func() {
			r := newRoutingMock()
			r.namespace = config.GetNamespace()
			times := 100
			pool := threading.NewTaskRunner(runtime.NumCPU())

			var counter int32
			var waitGroup sync.WaitGroup
			var errs []error
			var endpoints []Endpoint // 假设Endpoint是一个自定义的类型
			mutex := &sync.Mutex{}   // 用于保护共享变量
			for i := 0; i < times; i++ {
				waitGroup.Add(1)
				r.ClearRoutings()
				r.ClearInstances()
				pool.Schedule(func() {
					endpoint, err := r.GetEndpoint(context.Background(), &GetRouteRequest{UserId: 123})
					mutex.Lock() // 加锁
					if err == nil && endpoint.Address == "172.17.15.246:28002" {
						atomic.AddInt32(&counter, 1)
					}

					errs = append(errs, err)
					endpoints = append(endpoints, endpoint)
					mutex.Unlock() // 解锁
					waitGroup.Done()
				})
			}
			waitGroup.Wait()
			convey.Convey("test getEndpoint", func() {
				for _, err := range errs {
					convey.So(err, convey.ShouldBeNil)
				}
				for _, endpoint := range endpoints {
					convey.So(endpoint.Address, convey.ShouldEqual, "172.17.15.246:28002")
				}
				convey.So(counter, convey.ShouldEqual, times)
			})
		})

	})
}

func TestNamespace(t *testing.T) {

	r := newRouting(routerTypeCommon)
	assert.Equal(t, config.GetNamespace(), r.Namespace())
}

func TestIsRoutingService(t *testing.T) {
	assert.Equal(t, true, IsRoutingService("routing://sss"))
	assert.Equal(t, false, IsRoutingService("routinsag://sss"))
}

func TestClearRoutingByUser(t *testing.T) {

	r := newRouting(routerTypeCommon).(*routing)
	r.routingCache.Clear()
	r.routingCache.Set([]byte(toKey(123, "123")), []byte("123"), 3000)
	r.routingCache.Set([]byte(toKey(123, "")), []byte("123"), 3000)
	r.ClearRoutingByUser(0, "123")
	assert.Equal(t, int64(2), r.routingCache.EntryCount())

	r.ClearRoutingByUser(123, "123")
	assert.Equal(t, int64(1), r.routingCache.EntryCount())

	r.ClearRoutingByUser(123, "1212")
	assert.Equal(t, int64(1), r.routingCache.EntryCount())

	r.ClearRoutingByUser(123, "")
	assert.Equal(t, int64(0), r.routingCache.EntryCount())

}

func TestOnConfigChanged(t *testing.T) {

	r := newRouting(routerTypeCommon).(*routing)
	r.instances.Store("1213", &routeInfo{})
	r.onConfigChanged(&gconfig.Event{Type: gconfig.EventTypeDelete, Key: "1213"})

	_, ok := r.instances.Load("1213")
	assert.Equal(t, false, ok)

	r.onConfigChanged(&gconfig.Event{Type: gconfig.EventTypeUpdate, Key: "1213", Value: ""})
	_, ok = r.instances.Load("1213")
	assert.Equal(t, false, ok)

	r.onConfigChanged(&gconfig.Event{Type: gconfig.EventTypeUpdate, Key: "1213", Value: `{"addr":"123"}`})
	v, ok := r.instances.Load("1213")
	assert.Equal(t, true, ok)
	assert.Equal(t, "123", v.(*routeInfo).Addr)
}

func TestIsAioUser(t *testing.T) {
	gmetric.Init("TestIsAioUser")

	r := newRouting(routerTypeCommon)
	is, err := r.IsAioUser(context.Background(), 0)
	assert.NoError(t, err)
	assert.Equal(t, false, is)

	p := gomonkey.ApplyPrivateMethod(reflect.TypeOf(r), "getDataIdFromRouting", func(ctx context.Context, in *GetRouteRequest) (Endpoint, error) {
		return Endpoint{}, errors.New("ddd")
	})

	is, err = r.IsAioUser(context.Background(), 100)
	assert.EqualError(t, err, "GetRoute fail, err: ddd")

	p.ApplyPrivateMethod(reflect.TypeOf(r), "getDataIdFromRouting", func(ctx context.Context, in *GetRouteRequest) (Endpoint, error) {
		return Endpoint{IsAioUser: true}, nil
	})

	is, err = r.IsAioUser(context.Background(), 100)
	assert.NoError(t, err)
	assert.Equal(t, true, is)

	rr := r.(*routing)
	rr.configClient = nil
	is, err = r.IsAioUser(context.Background(), 100)
	assert.Equal(t, berror.ErrDefault, err)
	assert.Equal(t, false, is)
	p.Reset()
}

func TestDemoRouter(t *testing.T) {
	t.Run("test init, group", func(t *testing.T) {
		rr := GetDemoRouting()
		r := rr.(*routing)
		if r.routerGroup != "uta_router_da" {
			t.Failed()
		}
		if r.engineGroup != "uta_da" {
			t.Failed()
		}

		rr = newRouting(routerTypeCommon)
		r = rr.(*routing)
		if r.routerGroup != "uta_router" {
			t.Failed()
		}
		if r.engineGroup != "uta" {
			t.Failed()
		}
	})
}

func TestDiagnosis(t *testing.T) {
	convey.Convey("router Diagnosis", t, func() {

		result := diagnosis.NewResult(errors.New("xxx"))

		p := gomonkey.ApplyFuncReturn(diagnosis.DiagnoseGrpcDependency, result)
		p.ApplyFuncReturn(diagnosis.Dial, result.Es[0])
		defer p.Reset()

		dig := trDiagnose{key: "xxxxx", svc: &routing{}}
		dig.svc.instances.Store("aaa", &routeInfo{
			Addr: "asas",
		})

		convey.So(dig.Key(), convey.ShouldEqual, "xxxxx")
		r, err := dig.Diagnose(context.Background())
		resp := r.(map[string]interface{})
		convey.So(resp, convey.ShouldNotBeNil)
		convey.So(resp["grpc"], convey.ShouldEqual, result)
		convey.So(resp["grpc_uta_engine"], convey.ShouldEqual, result.Es)
		convey.So(err, convey.ShouldBeNil)
	})
}

func newRoutingMock() *routing {
	r := &routing{
		routingCache: freecache.NewCache(defaultCacheSize * 1024 * 1024),
		freshCh:      make(chan *freshMessage, freshMaxSize),
		singleFlight: syncx.NewSingleFlight(),
		routeClient: &MockRoutingAPIClient{
			getRouteFunc: func(ctx context.Context, in *GetRouteRequest, opts ...grpc.CallOption) (*routev1.GetRouteResponse, error) {
				return &routev1.GetRouteResponse{DataId: "uta_engine.g0i0.rpc", CacheTtl: -1, IsOldUta: true}, nil
			},
		},
		configClient: &MockConfigClient{
			getFunc: func(ctx context.Context, key string, opts ...gconfig.Option) (string, error) {
				return "{\"addr\":\"172.17.15.246:28002\"}", nil
			},
			listenFunc: func(ctx context.Context, key string, listener gconfig.Listener, opts ...gconfig.Option) error {
				return nil
			},
			deleteFunc: func(ctx context.Context, key string, opts ...gconfig.Option) error {
				return nil
			},
			putFunc: func(ctx context.Context, key string, value string, opts ...gconfig.Option) error {
				return nil
			},
		},
	}
	return r
}

type MockConfigClient struct {
	getFunc    func(ctx context.Context, key string, opts ...gconfig.Option) (string, error)
	listenFunc func(ctx context.Context, key string, listener gconfig.Listener, opts ...gconfig.Option) error
	putFunc    func(ctx context.Context, key string, value string, opts ...gconfig.Option) error
	deleteFunc func(ctx context.Context, key string, opts ...gconfig.Option) error
}

func (mock MockConfigClient) Get(ctx context.Context, key string, opts ...gconfig.Option) (string, error) {
	return mock.getFunc(ctx, key, opts...)
}

func (mock MockConfigClient) Listen(ctx context.Context, key string, listener gconfig.Listener, opts ...gconfig.Option) error {
	return mock.listenFunc(ctx, key, listener, opts...)
}

func (mock MockConfigClient) Put(ctx context.Context, key string, value string, opts ...gconfig.Option) error {
	return mock.putFunc(ctx, key, value, opts...)
}

func (mock MockConfigClient) Delete(ctx context.Context, key string, opts ...gconfig.Option) error {
	return mock.deleteFunc(ctx, key, opts...)
}

type MockRoutingAPIClient struct {
	getRouteFunc func(ctx context.Context, in *GetRouteRequest, opts ...grpc.CallOption) (*routev1.GetRouteResponse, error)
}

func (mock *MockRoutingAPIClient) GetRoute(ctx context.Context, in *GetRouteRequest, opts ...grpc.CallOption) (*routev1.GetRouteResponse, error) {
	return mock.getRouteFunc(ctx, in, opts...)
}
