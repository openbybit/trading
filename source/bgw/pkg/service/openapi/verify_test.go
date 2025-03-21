package openapi

import (
	"context"
	"errors"
	"testing"

	"git.bybit.com/svc/stub/pkg/pb/api/user"
	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
)

func TestOpenapiService_VerifyAPIKey(t *testing.T) {
	Convey("test VerifyAPIKey", t, func() {
		os := &openapiService{}
		gomonkey.ApplyFunc((*openapiService).GetAPIKey, func(service *openapiService, ctx context.Context, key string, string3 string) (*MemberLogin, error) {
			if key == "apikey222" {
				return &MemberLogin{
					ExtInfo: &user.MemberLoginExt{},
				}, nil
			}

			if key == "apikey333" {
				return nil, errors.New("mock err")
			}

			return &MemberLogin{
				ExtInfo: &user.MemberLoginExt{ExpiredTimeE0: 1234},
			}, nil
		})
		_, err := os.VerifyAPIKey(context.Background(), "apikey333", "from")
		So(err, ShouldNotBeNil)

		_, err = os.VerifyAPIKey(context.Background(), "apikey222", "from")
		So(err, ShouldBeNil)

		_, err = os.VerifyAPIKey(context.Background(), "apikey111", "from")
		So(err, ShouldNotBeNil)
	})
}
