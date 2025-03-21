package gray

import (
	"context"
	"flag"

	"code.bydev.io/fbu/gateway/gway.git/glog"

	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	rgray "bgw/pkg/service/gray"
)

func Init() {
	filter.Register(filter.GrayFilterKey, new)
}

const (
	// 资金账户的两个灰度策略，目前已经废弃
	FATag     = "IsFA"
	FASpotTAg = "IsUTASpotClose"
)

type gray struct {
	grayers []rgray.Grayer
}

func new() filter.Filter {
	return &gray{
		grayers: make([]rgray.Grayer, 0),
	}
}

func (g *gray) Do(next types.Handler) types.Handler {
	return func(ctx *types.Ctx) (err error) {
		md := metadata.MDFromContext(ctx)
		grayTags := []string{FATag, FASpotTAg}
		for _, grayer := range g.grayers {
			ok, err := grayer.GrayStatus(ctx)
			if err != nil {
				glog.Error(ctx, "get gray status failed", glog.String("err", err.Error()))
				continue
			}
			if ok {
				grayTags = append(grayTags, grayer.Tag())
			}
		}
		md.GrayTags = grayTags

		glog.Debug(ctx, "filter gray",
			glog.Int64("uid", md.UID),
			glog.Any("grayTags", grayTags),
		)

		return next(ctx)
	}
}

func (g *gray) GetName() string {
	return filter.GrayFilterKey
}

func (g *gray) Init(ctx context.Context, args ...string) (err error) {
	// warm up init gray
	// default strategy
	if len(args) == 0 {
		return nil
	}
	var tag string
	p := flag.NewFlagSet("gray", flag.ContinueOnError)
	p.StringVar(&tag, "tags", "", "gary tag")

	if err = p.Parse(args[1:]); err != nil {
		glog.Error(ctx, "response parse error", glog.Any("args", args), glog.String("error", err.Error()))
		return
	}
	// 不启用
	// tags := strings.Split(tag, ",")
	// for _, t := range tags {
	// 	g.grayers = append(g.grayers, rgray.NewGrayer(t, config.GetHTTPServerConfig().ServiceRegistry.ServiceName))
	// }

	return
}
