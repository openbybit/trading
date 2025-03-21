package gray

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	data = `
- tag: IsFA
  type: 3
  tail_list: [5,2]
- tag: IsFA
  type: 1
  tail_list: [5,2]
- tag: IsUTASpotClose
  type: 1
  tail_list: [5,2]
- tag: uta
  type: 3
  tail_list: [5,2]
- tag: tagForUnitTest
  type: 2
`
)

func TestNewGrayer(t *testing.T) {
	Convey("test NewGrayer", t, func() {
		g := NewGrayer("test_grayer", "test")
		So(g.Tag(), ShouldEqual, "test_grayer")

		gimp := g.(*grayer)
		gimp.OnCfgChange(nil, 123, 0)
		gimp.OnCfgChange(nil, 123, 1)

		res, err := g.GrayStatus(context.Background())
		So(res, ShouldBeFalse)
		So(err, ShouldBeNil)

		d := time.Now().Add(time.Hour).UnixNano()
		gimp.OnCfgChange(nil, d, 2)
		res, err = g.GrayStatus(context.Background())
		So(res, ShouldBeFalse)
		So(err, ShouldBeNil)
	})
}
