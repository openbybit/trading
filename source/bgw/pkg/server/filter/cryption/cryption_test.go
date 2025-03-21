package cryption

import (
	"context"
	"errors"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"code.bydev.io/public-lib/sec/sec-sign.git/cipher"
	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	gmd "google.golang.org/grpc/metadata"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
)

var errMock = errors.New("mock err")

func TestNewCrypter(t *testing.T) {
	Convey("test newCrypter", t, func() {
		f := newCrypter()
		n := f.GetName()
		So(n, ShouldEqual, filter.CryptionFilterKey)
	})
}

func TestCrypter_Init(t *testing.T) {
	Convey("test init", t, func() {
		c := &crypter{}
		err := c.Init(context.Background())
		So(err, ShouldBeNil)

		err = c.Init(context.Background(), "route", "--request=true", "--response=true")
		So(err, ShouldBeNil)
		So(c.Req, ShouldBeTrue)
		So(c.Resp, ShouldBeTrue)

		err = c.Init(context.Background(), "route", "--unknown=true")
		So(err, ShouldNotBeNil)
	})
}

func TestCrypter_Do(t *testing.T) {
	Convey("test do", t, func() {
		gmetric.Init("test")
		f := &crypter{
			Req:  true,
			Resp: false,
		}
		c, _ := cipher.NewCipher(&cipher.Config{})
		globalCipher = &Cipher{
			Cipher: c,
			grayer: &grayer{},
		}
		globalCipher.set(1, false, []int64{1, 2})

		next := func(ctx *types.Ctx) error {
			return nil
		}
		handler := f.Do(next)
		ctx := &types.Ctx{}

		md := metadata.MDFromContext(ctx)
		md.UID = 333
		err := handler(ctx)
		So(err, ShouldBeNil)

		md.UID = 111
		ctx.Request.Header.Set(cryptionReqHeader, "req_sign")
		patch := gomonkey.ApplyFunc((*cipher.Cipher).VerifySign, func(*cipher.Cipher, context.Context, string, []byte, string) (bool, error) { return false, errMock })
		err = handler(ctx)
		So(err, ShouldNotBeNil)
		patch.Reset()

		patch = gomonkey.ApplyFunc((*cipher.Cipher).VerifySign, func(*cipher.Cipher, context.Context, string, []byte, string) (bool, error) { return false, nil })
		err = handler(ctx)
		So(err, ShouldNotBeNil)
		patch.Reset()

		patch = gomonkey.ApplyFunc((*cipher.Cipher).VerifySign, func(*cipher.Cipher, context.Context, string, []byte, string) (bool, error) { return true, nil })
		defer patch.Reset()
		err = handler(ctx)
		So(err, ShouldBeNil)

		f = &crypter{
			Req:  true,
			Resp: true,
		}

		handler = f.Do(next)
		err = handler(ctx)
		So(err, ShouldBeNil)

		ctx.SetUserValue(constant.CtxInvokeResult, &mockSource{})
		patch1 := gomonkey.ApplyFunc((*mockSource).Metadata, func(*mockSource) gmd.MD {
			m := make(gmd.MD)
			m.Set(constant.BgwAPIResponseCodes, "10024")
			return m
		})
		err = handler(ctx)
		So(err, ShouldBeNil)
		patch1.Reset()

		patch1 = gomonkey.ApplyFunc((*mockSource).GetData, func(*mockSource) ([]byte, error) {
			return nil, errors.New("mock err")
		})
		err = handler(ctx)
		So(err, ShouldBeNil)
		patch1.Reset()

		patch2 := gomonkey.ApplyFunc((*cipher.Cipher).SignRespResult, func(*cipher.Cipher, context.Context, []byte) (string, error) {
			return "resp_sign", errors.New("mock err")
		})
		err = handler(ctx)
		So(err, ShouldNotBeNil)
		patch2.Reset()

		patch2 = gomonkey.ApplyFunc((*cipher.Cipher).SignRespResult, func(*cipher.Cipher, context.Context, []byte) (string, error) { return "resp_sign", nil })
		defer patch2.Reset()
		err = handler(ctx)
		So(err, ShouldBeNil)
	})
}

type mockSource struct{}

func (m *mockSource) GetData() ([]byte, error) {
	return []byte{}, nil
}

func (m *mockSource) Metadata() gmd.MD {
	md := make(gmd.MD)
	md.Set(constant.BgwAPIResponseCodes, "0")
	return md
}
