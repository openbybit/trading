package diagnosis

import (
	"context"
	"testing"

	"code.bydev.io/frameworks/byone/kafka"
	"code.bydev.io/frameworks/byone/zrpc"
	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
)

func TestConfigPreflight(t *testing.T) {
	Convey("test ConfigPreflight", t, func() {
		patch := gomonkey.ApplyFunc(DiagnoseEtcd, func(context.Context) Result { return Result{Errs: []string{"mock err"}} })
		defer patch.Reset()

		patch1 := gomonkey.ApplyFunc(DiagnoseRedis, func(context.Context) Result { return Result{Errs: []string{"mock err"}} })
		defer patch1.Reset()

		patch2 := gomonkey.ApplyFunc(DiagnoseGrpcDependency, func(context.Context, zrpc.RpcClientConf) Result {
			return Result{Errs: []string{"mock err"}}
		})
		defer patch2.Reset()

		patch3 := gomonkey.ApplyFunc(DiagnoseKafka, func(context.Context, string, kafka.UniversalClientConfig) Result {
			return Result{Errs: []string{"mock err"}}
		})
		defer patch3.Reset()

		configPreflight()
	})
}
