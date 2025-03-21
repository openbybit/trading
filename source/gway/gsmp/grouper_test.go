package gsmp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/willf/bloom"
	"go.uber.org/atomic"
	"google.golang.org/grpc"

	"code.bydev.io/fbu/gateway/gway.git/ggrpc/pool"
	"code.bydev.io/fbu/gateway/gway.git/gsmp/imp"
)

var (
	mockDiscovery = func(ctx context.Context, registry, namespace, group string) (addrs []string) {
		return []string{"addr"}
	}

	mockErr = errors.New("mock err")
)

func TestNew(t *testing.T) {
	Convey("test new grouper", t, func() {
		cfg := &Config{}
		g, err := New(cfg)
		So(g, ShouldBeNil)
		So(err, ShouldNotBeNil)

		cfg = &Config{
			Registry:  "imp",
			Group:     "g",
			Namespace: "n",
			Discovery: mockDiscovery,
		}
		g, err = New(cfg)
		So(g, ShouldNotBeNil)
		So(err, ShouldBeNil)
	})
}

func TestGrouper_Init(t *testing.T) {
	Convey("test grouper init", t, func() {
		g := &grouper{
			synced: atomic.NewInt32(0),
		}

		mockGetFulldata := func(g *grouper, ctx context.Context, uid int64) (map[int64]int32, map[string][]int64, error) {
			return map[int64]int32{123: 123}, nil, nil
		}
		patch := gomonkey.ApplyFunc((*grouper).GetImpGroup, mockGetFulldata)
		g.init(context.Background())
		So(g.synced.Load(), ShouldEqual, 1)
		patch.Reset()
	})
}

func TestGrouper_Init2(t *testing.T) {
	Convey("test grouper init fail", t, func() {
		g := &grouper{
			synced:   atomic.NewInt32(0),
			interval: 1,
		}
		mockGetFulldataErr := func(g *grouper, ctx context.Context, uid int64) (map[int64]int32, map[string][]int64, error) {
			return nil, nil, mockErr
		}
		patch := gomonkey.ApplyFunc((*grouper).GetImpGroup, mockGetFulldataErr)
		g.init(context.Background())
		So(g.synced.Load(), ShouldEqual, 2)
		patch.Reset()
	})
}

func TestGroup_GetGroup(t *testing.T) {
	Convey("test get group", t, func() {
		g := &grouper{
			synced: atomic.NewInt32(0),
			groups: map[int64]int32{123: 321},
		}

		res, err := g.GetGroup(context.Background(), 123)
		So(res, ShouldEqual, 321)
		So(err, ShouldBeNil)

		mockGetUidGroup := func(g *grouper, ctx context.Context, uid int64) (int32, error) {
			return 111, nil
		}

		patch := gomonkey.ApplyFunc((*grouper).GetUidGroup, mockGetUidGroup)
		defer patch.Reset()

		res, err = g.GetGroup(context.Background(), 234)
		So(res, ShouldEqual, 111)
		So(err, ShouldBeNil)
	})
}

// GOARCH=amd64 go test -gcflags='-N -l'
func TestGroup_GetImpGroup(t *testing.T) {
	Convey("test GetImpGroup", t, func() {
		Convey("test get conn err", func() {
			g := &grouper{}
			g.discovery = func(ctx context.Context, registry, namespace, group string) (addrs []string) {
				return []string{}
			}

			m, _, err := g.GetImpGroup(context.Background(), 123)
			So(err, ShouldNotBeNil)
			So(m, ShouldBeNil)
		})

		Convey("test imp res err", func() {
			ctrl := gomock.NewController(t)
			mockConn := pool.NewMockConn(ctrl)
			mockConn.EXPECT().Client().Return(nil)
			mockConn.EXPECT().Close().Return(nil)

			mockGetConn := func() (pool.Conn, error) {
				return mockConn, nil
			}

			patch := gomonkey.ApplyFunc((*grouper).GetImpConn, mockGetConn)
			defer patch.Reset()

			mockClient := imp.NewMockImpSmpServiceClient(ctrl)
			mockClient.EXPECT().SmpGroupQuery(context.Background(), gomock.Any()).Return(nil, mockErr)
			mockNewClient := func(grpc.ClientConnInterface) imp.ImpSmpServiceClient {
				return mockClient
			}
			patch1 := gomonkey.ApplyFunc(imp.NewImpSmpServiceClient, mockNewClient)
			defer patch1.Reset()

			g := &grouper{
				discovery: mockDiscovery,
			}

			m, _, err := g.GetImpGroup(context.Background(), 123)
			So(err, ShouldEqual, mockErr)
			So(m, ShouldBeNil)
		})

		Convey("test imp resp code != 0", func() {
			ctrl := gomock.NewController(t)
			mockConn := pool.NewMockConn(ctrl)
			mockConn.EXPECT().Client().Return(nil).AnyTimes()
			mockConn.EXPECT().Close().Return(nil).AnyTimes()

			mockGetConn := func() (pool.Conn, error) {
				return mockConn, nil
			}

			patch := gomonkey.ApplyFunc((*grouper).GetImpConn, mockGetConn)
			defer patch.Reset()

			mockClient := imp.NewMockImpSmpServiceClient(ctrl)
			resp := &imp.SmpGroupQueryResp{
				RetCode: "1",
			}
			mockClient.EXPECT().SmpGroupQuery(context.Background(), gomock.Any()).Return(resp, nil)
			mockNewClient := func(grpc.ClientConnInterface) imp.ImpSmpServiceClient {
				return mockClient
			}
			patch1 := gomonkey.ApplyFunc(imp.NewImpSmpServiceClient, mockNewClient)
			defer patch1.Reset()

			g := &grouper{
				discovery: mockDiscovery,
			}

			m, _, err := g.GetImpGroup(context.Background(), 123)
			So(err, ShouldNotBeNil)
			So(m, ShouldBeNil)
		})

		Convey("test imp resp success", func() {
			ctrl := gomock.NewController(t)
			mockConn := pool.NewMockConn(ctrl)
			mockConn.EXPECT().Client().Return(nil).AnyTimes()
			mockConn.EXPECT().Close().Return(nil).AnyTimes()

			mockGetConn := func() (pool.Conn, error) {
				return mockConn, nil
			}

			patch := gomonkey.ApplyFunc((*grouper).GetImpConn, mockGetConn)
			defer patch.Reset()

			mockClient := imp.NewMockImpSmpServiceClient(ctrl)
			resp := &imp.SmpGroupQueryResp{
				RetCode: "0",
			}
			mockClient.EXPECT().SmpGroupQuery(context.Background(), gomock.Any()).Return(resp, nil)
			mockNewClient := func(grpc.ClientConnInterface) imp.ImpSmpServiceClient {
				return mockClient
			}
			patch1 := gomonkey.ApplyFunc(imp.NewImpSmpServiceClient, mockNewClient)
			defer patch1.Reset()

			g := &grouper{
				discovery: mockDiscovery,
			}

			_, _, err := g.GetImpGroup(context.Background(), 123)
			So(err, ShouldBeNil)
		})
	})
}

func TestGroup_GetUidGroup(t *testing.T) {
	Convey("test getUidGroup", t, func() {
		g := &grouper{
			groups: map[int64]int32{},
			synced: atomic.NewInt32(0),
			uids:   map[string][]int64{},
			bl:     bloom.New(10000, 5),
		}
		mockGetImpGroup := func(g *grouper, ctx context.Context, uid int64) (map[int64]int32, map[string][]int64, error) {
			return nil, nil, mockErr
		}
		patch := gomonkey.ApplyFunc((*grouper).GetImpGroup, mockGetImpGroup)
		res, err := g.GetUidGroup(context.Background(), 123)
		So(res, ShouldEqual, 0)
		So(err, ShouldEqual, mockErr)
		patch.Reset()

		mockGetImpGroup = func(g *grouper, ctx context.Context, uid int64) (map[int64]int32, map[string][]int64, error) {
			return map[int64]int32{2268: 124}, nil, nil
		}
		patch1 := gomonkey.ApplyFunc((*grouper).GetImpGroup, mockGetImpGroup)
		res, err = g.GetUidGroup(context.Background(), 2268)
		So(res, ShouldEqual, 124)
		So(err, ShouldEqual, nil)

		res, err = g.GetUidGroup(context.Background(), 234)
		So(res, ShouldEqual, 0)
		So(err, ShouldBeNil)

		// bloom filter
		res, err = g.GetUidGroup(context.Background(), 234)
		So(res, ShouldEqual, 0)
		So(err, ShouldBeNil)
		patch1.Reset()
	})
}

func TestGroup_GetUidGroup2(t *testing.T) {
	Convey("test getUidGroup", t, func() {
		g := &grouper{
			groups: map[int64]int32{},
			synced: atomic.NewInt32(0),
			uids:   map[string][]int64{},
			bl:     bloom.New(10000, 5),
		}

		mockGetImpGroup := func(g *grouper, ctx context.Context, uid int64) (map[int64]int32, map[string][]int64, error) {
			return map[int64]int32{2268: 124}, nil, nil
		}
		patch1 := gomonkey.ApplyFunc((*grouper).GetImpGroup, mockGetImpGroup)
		res, err := g.GetUidGroup(context.Background(), 2268)
		So(res, ShouldEqual, 124)
		So(err, ShouldEqual, nil)

		res, err = g.GetUidGroup(context.Background(), 234)
		So(res, ShouldEqual, 0)
		So(err, ShouldBeNil)

		// bloom filter
		res, err = g.GetUidGroup(context.Background(), 234)
		So(res, ShouldEqual, 0)
		So(err, ShouldBeNil)
		patch1.Reset()
	})
}

func TestGroup_getImpConn(t *testing.T) {
	Convey("test GetImpConn", t, func() {
		ctrl := gomock.NewController(t)
		mockPools := pool.NewMockPools(ctrl)
		mockPools.EXPECT().GetConn(context.Background(), gomock.Any()).Return(nil, mockErr)

		g := &grouper{
			connPool: mockPools,
			index:    atomic.NewInt32(0),
		}
		g.discovery = func(ctx context.Context, registry, namespace, group string) (addrs []string) {
			return []string{}
		}

		conn, err := g.GetImpConn(context.Background())
		So(conn, ShouldBeNil)
		So(err, ShouldNotBeNil)

		g.discovery = mockDiscovery
		_, err = g.GetImpConn(context.Background())
		So(err, ShouldEqual, mockErr)

		mockPools = pool.NewMockPools(ctrl)
		mockPools.EXPECT().GetConn(context.Background(), gomock.Any()).Return(nil, nil)
		g = &grouper{
			connPool:  mockPools,
			index:     atomic.NewInt32(0),
			discovery: mockDiscovery,
		}
		_, err = g.GetImpConn(context.Background())
		So(err, ShouldEqual, nil)
	})
}

func TestGroup_getServiceRoundRobin(t *testing.T) {
	Convey("test getServiceRoundRobin", t, func() {
		g := &grouper{
			discovery: mockDiscovery,
			index:     atomic.NewInt32(0),
		}

		_, err := g.getServiceRoundRobin(context.Background())
		So(err, ShouldBeNil)

		g.discovery = func(ctx context.Context, registry, namespace, group string) (addrs []string) {
			return []string{}
		}
		_, err = g.getServiceRoundRobin(context.Background())
		So(err, ShouldNotBeNil)
	})
}

func TestGroup_handle(t *testing.T) {
	Convey("test handle", t, func() {
		msg := &imp.SmpGroupConfigItem{
			InstSmpGroup: []*imp.SmpGroupItem{
				{
					SmpGroup: 456,
					Uids:     []int64{123, 456},
				},
			},
		}
		value, _ := json.Marshal(msg)

		g := &grouper{
			groups: map[int64]int32{},
			uids:   map[string][]int64{},
		}
		err := g.HandleMsg(value)
		So(err, ShouldBeNil)

		group, err := g.GetGroup(context.Background(), 123)
		So(group, ShouldEqual, 456)
		So(err, ShouldBeNil)
	})
}

var bl *bloom.BloomFilter

// 128M
func BenchmarkBloom(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bl = bloom.New(1<<30, 7)
	}
}

// func TestGetGroup(t *testing.T) {
// 	t.Run("GetGroup", func(t *testing.T) {
// 		g := &grouper{
// 			discovery: mockDiscovery,
// 			synced:    atomic.NewInt32(1),
// 		}
// 		g.GetGroup(context.Background(), 1)
// 		g.newImpClient(nil)
// 	})

// 	t.Run("GetImpGroup error", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)
// 		mockConn := pool.NewMockConn(ctrl)
// 		mockConn.EXPECT().Client().Return(nil)
// 		mockConn.EXPECT().Close().Return(nil)
// 		mockPools := pool.NewMockPools(ctrl)
// 		mockPools.EXPECT().GetConn(context.Background(), gomock.Any()).Return(mockConn, nil)

// 		mockClient := imp.NewMockImpSmpServiceClient(ctrl)
// 		mockClient.EXPECT().SmpGroupQuery(context.Background(), gomock.Any()).Return(nil, mockErr)

// 		newImpClientFn := func(cc grpc.ClientConnInterface) imp.ImpSmpServiceClient {
// 			return mockClient
// 		}

// 		g := &grouper{
// 			discovery: mockDiscovery,
// 			synced:    atomic.NewInt32(1),
// 			index:     atomic.NewInt32(0),
// 			connPool:  mockPools,
// 			impFn:     newImpClientFn,
// 		}

// 		g.GetImpGroup(context.Background(), 1)
// 	})

// 	t.Run("GetImpGroup RetCode", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)
// 		mockConn := pool.NewMockConn(ctrl)
// 		mockConn.EXPECT().Client().Return(nil)
// 		mockConn.EXPECT().Close().Return(nil)
// 		mockPools := pool.NewMockPools(ctrl)
// 		mockPools.EXPECT().GetConn(context.Background(), gomock.Any()).Return(mockConn, nil)

// 		mockClient := imp.NewMockImpSmpServiceClient(ctrl)
// 		mockClient.EXPECT().SmpGroupQuery(context.Background(), gomock.Any()).Return(&imp.SmpGroupQueryResp{RetCode: "10001"}, nil)

// 		newImpClientFn := func(cc grpc.ClientConnInterface) imp.ImpSmpServiceClient {
// 			return mockClient
// 		}

// 		g := &grouper{
// 			discovery: mockDiscovery,
// 			synced:    atomic.NewInt32(1),
// 			index:     atomic.NewInt32(0),
// 			connPool:  mockPools,
// 			impFn:     newImpClientFn,
// 		}

// 		g.GetImpGroup(context.Background(), 1)
// 	})

// 	t.Run("GetImpGroup Success", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)
// 		mockConn := pool.NewMockConn(ctrl)
// 		mockConn.EXPECT().Client().Return(nil)
// 		mockConn.EXPECT().Close().Return(nil)
// 		mockPools := pool.NewMockPools(ctrl)
// 		mockPools.EXPECT().GetConn(context.Background(), gomock.Any()).Return(mockConn, nil)

// 		res := &imp.SmpGroupQueryResp{RetCode: "0", SmpGroups: nil}
// 		mockClient := imp.NewMockImpSmpServiceClient(ctrl)
// 		mockClient.EXPECT().SmpGroupQuery(context.Background(), gomock.Any()).Return(res, nil)

// 		newImpClientFn := func(cc grpc.ClientConnInterface) imp.ImpSmpServiceClient {
// 			return mockClient
// 		}

// 		g := &grouper{
// 			discovery: mockDiscovery,
// 			synced:    atomic.NewInt32(1),
// 			index:     atomic.NewInt32(0),
// 			connPool:  mockPools,
// 			impFn:     newImpClientFn,
// 		}

// 		g.GetImpGroup(context.Background(), 1)
// 	})
// }

func TestHandleMsg(t *testing.T) {
	t.Run("unmarshal fail", func(t *testing.T) {
		g := &grouper{}
		err := g.HandleMsg([]byte("aaa"))
		if err == nil {
			t.Error("should be err")
		}
	})

	t.Run("delete uids", func(t *testing.T) {
		msg := &imp.SmpGroupConfigItem{InstId: "a"}
		value, _ := json.Marshal(msg)
		g := &grouper{
			uids: map[string][]int64{"a": {1}},
		}
		g.HandleMsg(value)
	})

	t.Run("normal", func(t *testing.T) {
		msg := &imp.SmpGroupConfigItem{InstId: "a", InstSmpGroup: []*imp.SmpGroupItem{{SmpGroup: 1, Uids: []int64{1, 2}}, {SmpGroup: 2, Uids: []int64{3}}}}
		value, _ := json.Marshal(msg)
		g := &grouper{
			groups: map[int64]int32{1: 4},
			uids:   map[string][]int64{"a": {1, 4}},
		}
		g.HandleMsg(value)
	})
}
