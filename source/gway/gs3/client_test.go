package gs3

import (
	"context"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
)

func TestNewClient(t *testing.T) {
	convey.Convey("NewClient", t, func() {
		keyID := "xx"
		secret := "xx"
		region := "ap-southeast-1"
		key := "/BGW/unify-test-1/spot/broker-server/20230111071037"

		s3, err := NewClient(keyID, secret, region, WithPath("."), WithBucket("bbgw-gateway-protocol"))
		convey.So(err, convey.ShouldBeNil)
		convey.So(s3, convey.ShouldNotBeNil)
		data, err := s3.Download(context.TODO(), key, time.Time{})
		convey.So(err, convey.ShouldBeNil)
		t.Log(len(data), err)

		ctx, cancel := context.WithTimeout(context.TODO(), 50*time.Millisecond)
		data, err = s3.Download(ctx, key, time.Time{})
		cancel()
		convey.So(err, convey.ShouldNotBeNil)
		t.Log(len(data), err)

		s3, err = NewClient("", secret, region, WithPath("."), WithBucket("bbgw-gateway-protocol"))
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(s3, convey.ShouldBeNil)

		s3, err = NewClient(keyID, secret, region)
		convey.So(err, convey.ShouldBeNil)
		convey.So(s3, convey.ShouldNotBeNil)
	})
}
