package bsp

import (
	"context"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gbsp"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	jsoniter "github.com/json-iterator/go"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/types"
	"bgw/pkg/config"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
)

func Init() {
	filter.Register(filter.BspFilterKey, newBsp)
}

var (
	once    sync.Once
	checker gbsp.Checker
)

const (
	BspHeaderAuth     = "X-Bybit-Auth"
	BspHeaderAuthTime = "X-Bybit-Auth-Timestamp"
	BspHeaderAuthApp  = "X-Bybit-Auth-APP"
)

func newBsp() filter.Filter {
	return &bsp{}
}

type bsp struct{}

// GetName returns the name of the filter
func (b *bsp) GetName() string {
	return filter.BspFilterKey
}

func (b *bsp) Do(next types.Handler) types.Handler {
	return func(ctx *types.Ctx) error {
		if checker != nil {
			auth := ctx.Request.Header.Peek(BspHeaderAuth)
			time := ctx.Request.Header.Peek(BspHeaderAuthTime)
			app := ctx.Request.Header.Peek(BspHeaderAuthApp)
			glog.Debug(ctx, "bsp info", glog.String("auth", string(auth)), glog.String("time", string(time)),
				glog.String("app", string(app)))
			ui, err := checker.Check(ctx, auth, time, app)
			if err != nil {
				glog.Debug(ctx, "bsp check failed", glog.String("err", err.Error()))
				return berror.ErrAuthVerifyFailed
			}

			data, err := jsoniter.Marshal(ui)
			if err != nil {
				return berror.NewInterErr("bsp userinfo marshal failed, " + err.Error())
			}

			md := metadata.MDFromContext(ctx)
			md.BspInfo = string(data)
			md.UID = ui.AdminID
		}

		return next(ctx)
	}
}

func (b *bsp) Init(ctx context.Context, args ...string) error {
	var err error
	once.Do(func() {
		checker, err = gbsp.NewChecker(ctx, getBspPublicKey())
		if err != nil {
			galert.Error(ctx, "init bsp check failed", galert.WithField("err", err.Error()))
		}
	})
	return err
}

func getBspPublicKey() string {
	publicKey := (&config.Global.Bsp).GetOptions("public_key", "")
	glog.Info(context.Background(), "bsp", glog.String("key", publicKey))
	return publicKey
}
