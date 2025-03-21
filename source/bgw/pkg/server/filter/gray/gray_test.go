package gray

import (
	"context"
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"

	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	rgray "bgw/pkg/service/gray"
)

func TestNewGray(t *testing.T) {
	Convey("test new gray", t, func() {
		Init()
		g := new()
		n := g.GetName()
		So(n, ShouldEqual, filter.GrayFilterKey)
	})
}

func TestGray_Init(t *testing.T) {
	Convey("test gray init", t, func() {
		patch := gomonkey.ApplyFunc(rgray.NewGrayer, func() rgray.Grayer { return nil })
		defer patch.Reset()

		g := &gray{}
		err := g.Init(context.Background())
		So(err, ShouldBeNil)

		args := []string{"gray", "--tags=user,uta"}
		err = g.Init(context.Background(), args...)
		So(err, ShouldBeNil)

		args = append(args, "--wrongArg=123")
		err = g.Init(context.Background(), args...)
		So(err, ShouldNotBeNil)
	})
}

func TestGray_Do(t *testing.T) {
	Convey("test gray do", t, func() {
		patch := gomonkey.ApplyFunc(rgray.NewGrayer, func() rgray.Grayer { return &mockGrayer{} })
		defer patch.Reset()

		g := &gray{
			grayers: []rgray.Grayer{&mockGrayer{}},
		}
		args := []string{"gray", "--tags=user,uta"}
		_ = g.Init(context.Background(), args...)
		next := func(ctx *types.Ctx) error {
			return nil
		}
		handler := g.Do(next)
		ctx := &types.Ctx{}
		md := metadata.MDFromContext(ctx)
		err := handler(ctx)
		So(err, ShouldBeNil)

		md.UID = 10
		err = handler(ctx)
		So(err, ShouldBeNil)

		patch1 := gomonkey.ApplyFunc((*mockGrayer).GrayStatus, func(*mockGrayer, context.Context) (bool, error) {
			return false, errors.New("mock err")
		})
		defer patch1.Reset()

		md.UID = 10
		err = handler(ctx)
		So(err, ShouldBeNil)
	})
}

type mockGrayer struct{}

func (m *mockGrayer) GrayStatus(ctx context.Context) (gray bool, err error) {
	return true, nil
}

func (m *mockGrayer) Tag() string {
	return "tag"
}
