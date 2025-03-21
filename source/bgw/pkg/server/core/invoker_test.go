package core

import (
	"context"
	"errors"
	"io"
	"os"
	"reflect"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gcore/filesystem"
	"code.bydev.io/fbu/gateway/gway.git/generic"
	_ "code.bydev.io/fbu/gateway/gway.git/ggrpc"
	"code.bydev.io/fbu/gateway/gway.git/ghttp"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tj/assert"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/config"
	"bgw/pkg/server/core/grpc"
	"bgw/pkg/server/core/http"
	"bgw/pkg/server/metadata"
	"bgw/pkg/test"
)

func TestNewInvoker(t *testing.T) {
	Convey("test newInvoker", t, func() {
		patch := gomonkey.ApplyFunc(gmetric.IncDefaultError, func(typ string, label string) {})
		defer patch.Reset()
		iv := newInvoker(context.Background())
		So(iv, ShouldNotBeNil)
	})
}

func TestInvokeHttp(t *testing.T) {
	iv := invoker{
		sc:         &config.RemoteConfig{},
		httpEngine: ghttp.GetInvoker(),
	}
	rctx, _ := test.NewReqCtx()
	err := iv.invokeHTTP(context.Background(), "aaa", http.NewRequest(rctx),
		http.NewResult())
	assert.Error(t, err)

	assert.Equal(t, reflect.TypeOf(versionChangeEvent{}), iv.GetEventType())
	assert.Equal(t, 0, iv.GetPriority())
}

func TestInvokeGrpc(t *testing.T) {
	iv := invoker{
		sc:         &config.RemoteConfig{},
		grpcEngine: generic.NewEngine(),
	}
	rctx, _ := test.NewReqCtx()
	err := iv.invokeGRPC(context.Background(), "aaa", grpc.NewRPCRequest(rctx, "xxx", "sas", "gpt"),
		http.NewResult())
	assert.Error(t, err)
}

func TestGetLocalDescriptor(t *testing.T) {
	iv := invoker{
		sc: &config.RemoteConfig{},
	}
	b := iv.getLocalDescriptor(&AppVersion{})
	assert.Nil(t, b)

	f, err := os.CreateTemp("", "TestGetLocalDescriptorTmp")
	p := gomonkey.ApplyFuncReturn(filesystem.OpenFile, f, err)
	b = iv.getLocalDescriptor(&AppVersion{
		Version: AppVersionEntry{Resources: [2]ResourceEntry{
			{}, {Checksum: "d41d8cd98f00b204e9800998ecf8427e"},
		}},
	})
	assert.Equal(t, 0, len(b))

	f, err = os.CreateTemp("", "TestGetLocalDescriptorTmp")
	p.ApplyFuncReturn(filesystem.OpenFile, f, err)
	b = iv.getLocalDescriptor(&AppVersion{
		Version: AppVersionEntry{Resources: [2]ResourceEntry{
			{}, {Checksum: "222"},
		}},
	})
	assert.Nil(t, b)
	p.Reset()

	f, err = os.CreateTemp("", "TestGetLocalDescriptorTmp")
	p.ApplyFuncReturn(filesystem.OpenFile, f, err).ApplyFuncReturn(io.ReadAll, nil, errors.New("sss"))
	b = iv.getLocalDescriptor(&AppVersion{
		Version: AppVersionEntry{Resources: [2]ResourceEntry{
			{}, {Checksum: "d41d8cd98f00b204e9800998ecf8427e"},
		}},
	})
	assert.Nil(t, b)

	f, err = os.CreateTemp("", "TestGetLocalDescriptorTmp")
	p.ApplyFuncReturn(filesystem.OpenFile, f, err).ApplyFuncReturn(io.ReadAll, nil, errors.New("sss"))
	f.Close()
	b = iv.getLocalDescriptor(&AppVersion{
		Version: AppVersionEntry{Resources: [2]ResourceEntry{
			{}, {Checksum: "d41d8cd98f00b204e9800998ecf8427e"},
		}},
	})
	assert.Nil(t, b)
	p.Reset()
}

func TestInvoker_invoker(t *testing.T) {
	Convey("test invoker invoke", t, func() {
		iv := &invoker{}
		iv.invoke = iv.baseInvoke
		ctx := &types.Ctx{}
		mf := &MethodConfig{
			service: &ServiceConfig{},
		}
		md := metadata.MDFromContext(ctx)
		mf.Service().Protocol = constant.HttpProtocol
		patch := gomonkey.ApplyFunc((*invoker).invokeHTTP, func(i *invoker, ctx context.Context, addr string, request Request, result Result) (err error) {
			return nil
		})
		err := iv.invoke(ctx, mf, md)
		So(err, ShouldBeNil)
		patch.Reset()

		patch = gomonkey.ApplyFunc((*invoker).invokeHTTP, func(i *invoker, ctx context.Context, addr string, request Request, result Result) (err error) {
			return errors.New("mock err")
		})
		err = iv.invoke(ctx, mf, md)
		So(err, ShouldNotBeNil)
		patch.Reset()

	})
}

func Test_recov(t *testing.T) {
	Convey("test revov", t, func() {
		var r = errors.New("mock err")
		recov(r)
	})
}
