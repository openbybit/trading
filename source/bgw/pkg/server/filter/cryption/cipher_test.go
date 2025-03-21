package cryption

import (
	"context"
	"testing"

	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/public-lib/sec/sec-sign.git/cipher"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/config_center"
	"bgw/pkg/config_center/nacos"
)

func TestNewCipher(t *testing.T) {
	Convey("test new cipher", t, func() {
		ctrl := gomock.NewController(t)
		mockNacos := config_center.NewMockConfigure(ctrl)
		mockNacos.EXPECT().Listen(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		patch := gomonkey.ApplyFunc(nacos.NewNacosConfigure, func(ctx context.Context, opts ...nacos.Options) (config_center.Configure, error) {
			return mockNacos, nil
		})
		defer patch.Reset()

		_, err := newCipher(context.Background())
		So(err, ShouldBeNil)

		c := getCipher(context.Background())
		So(c, ShouldNotBeNil)

		c = getCipher(context.Background())
		So(c, ShouldNotBeNil)
	})
}

func TestCipher_OnEvent(t *testing.T) {
	Convey("test cipher on event", t, func() {
		c, _ := cipher.NewCipher(&cipher.Config{})
		Ci := &Cipher{
			Cipher: c,
			grayer: &grayer{},
		}

		err := Ci.OnEvent(nil)
		So(err, ShouldBeNil)

		e := &observer.DefaultEvent{}
		err = Ci.OnEvent(e)
		So(err, ShouldBeNil)

		e.Value = "123"
		e.Key = cipherCfgFile
		err = Ci.OnEvent(e)
		So(err, ShouldBeNil)

		e.Key = grayCfgFile
		err = Ci.OnEvent(e)
		So(err, ShouldBeNil)
	})
}

func TestCipher_UpdateCiperCfg(t *testing.T) {
	Convey("test update ciperCfg", t, func() {
		c, _ := cipher.NewCipher(&cipher.Config{})
		Ci := &Cipher{
			Cipher: c,
			grayer: &grayer{},
		}

		v := `{
    "Disable": false,
    "AllowNilSign": false,
    "SignExpiredBefore": "30m",
    "SignExpiredAfter": "30m",
    "SignKey": [
        {
            "Latest": true,
            "Version": "v1",
            "XORKey": "345",
            "RSAPrivateKey": "345"
		}
    ]    
}`
		err := Ci.updateCipherCfg(v)
		So(err, ShouldBeNil)

		v = `
status: 1 # 灰度配置总开关，0表示不生效， 1表示灰度生效
full_on: true # 增加全量灰度的字段，如果为true，则表示所有用户都在灰度中
tails: [11, 22, 33, 44]`

		err = Ci.updateGrayCfg(v)
		So(err, ShouldBeNil)
	})
}

func TestGrayer(t *testing.T) {
	Convey("test grayer", t, func() {
		g := &grayer{}
		g.set(0, false, []int64{1, 2})
		s := g.check(123)
		So(s, ShouldBeFalse)

		g.set(1, true, []int64{1, 2})
		s = g.check(123)
		So(s, ShouldBeTrue)

		g.set(1, false, []int64{1, 2})
		s = g.check(123)
		So(s, ShouldBeFalse)

		s = g.check(122)
		So(s, ShouldBeTrue)
	})
}

func TestTailMatch(t *testing.T) {
	Convey("test tailMatch", t, func() {
		res := tailMatch(5, 6)
		So(res, ShouldBeFalse)

		res = tailMatch(10, 0)
		So(res, ShouldBeTrue)
	})
}
