package signature

import (
	"context"
	"fmt"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/gsechub"
	"git.bybit.com/gtd/gopkg/solutions/risksign.git"
	"git.bybit.com/gtdmicro/stub/pkg/pb/mid/common/sign"
	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/yaml.v2"

	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
)

var mockErr = fmt.Errorf("mock err")

func TestNew(t *testing.T) {
	Convey("test New", t, func() {
		Init()
		s := new()
		n := s.GetName()
		So(n, ShouldEqual, filter.SignatureFilterKey)
	})
}

func TestSignatureKey_Do(t *testing.T) {
	Convey("test Do", t, func() {

		f := new()
		h := func(c *types.Ctx) error {
			return nil
		}
		handler := f.Do(h)

		patch := gomonkey.ApplyFunc((*signatureKey).getSignature, func(kk *signatureKey, ctx context.Context, appID string) (string, error) {
			if appID == "future" {
				return "signature", nil
			}
			return "", mockErr
		})
		defer patch.Reset()

		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)
		md.Route.Registry = "future"

		err := handler(ctx)
		So(err, ShouldBeNil)

		md.Route.Registry = "cht"
		err = handler(ctx)
		So(err, ShouldNotBeNil)
	})
}

func TestSignatureKey_Init(t *testing.T) {
	Convey("test init", t, func() {
		sigK := &signatureKey{}

		Convey("no args", func() {
			err := sigK.Init(context.Background())
			So(err, ShouldBeNil)
		})

		Convey("have args", func() {
			err := sigK.Init(context.Background(), "123")
			So(err, ShouldBeNil)
		})
	})
}

func TestSignatureKey_getSignature(t *testing.T) {
	Convey("test getSignature", t, func() {
		defaultKeyKeeper = newPrivateKeyKeeper()

		sigK := &signatureKey{}

		sigCgf := &signatureCfg{
			AppName: "bgw",
			AppKeys: []appKey{
				{
					AppID: "asset",
					Key:   "3456",
				},
			},
		}

		c, _ := yaml.Marshal(sigCgf)
		e := &observer.DefaultEvent{
			Value: string(c),
		}

		patch := gomonkey.ApplyFunc(gsechub.Decrypt, func(string) (string, error) { return "123", nil })
		defer patch.Reset()

		l, _ := defaultKeyKeeper.(observer.EventListener)
		_ = l.OnEvent(e)

		Convey("get key err", func() {
			_, err := sigK.getSignature(context.Background(), "nft")
			So(err, ShouldNotBeNil)
		})

		Convey("sign info err", func() {
			_, err := sigK.getSignature(context.Background(), "asset")
			So(err, ShouldNotBeNil)
		})

		Convey("sign info success", func() {
			patch1 := gomonkey.ApplyFunc(risksign.WithSignInfo, func(string2 string, string3 string) (*sign.SignInfo, error) {
				return &sign.SignInfo{}, nil
			})
			defer patch1.Reset()
			_, err := sigK.getSignature(context.Background(), "asset")
			So(err, ShouldBeNil)
		})
	})
}
