package ban

import (
	"bgw/pkg/common/berror"
	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service/ban"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gmetric"
	"context"
	"encoding/json"
	"flag"
	"github.com/valyala/fasthttp"
)

func Init() {
	filter.Register(filter.BanFilterKey, newBanFilter)
}

func newBanFilter() filter.Filter {
	return &banFilter{}
}

type banRule struct {
	bts map[banTag]struct{}
}

type banTag struct {
	BizType  string `json:"bizType"`
	TagName  string `json:"tagName"`
	TagValue string `json:"tagValue"`
}

type banFilter struct {
	br banRule
}

// Init '--banTags=[{"bizType":"XXX","tagName":"xxx","tagValue":"xxx"}]}'
func (bf *banFilter) Init(ctx context.Context, args ...string) (err error) {
	rule, err := bf.parseFlag(ctx, args)
	bf.br = rule
	return err
}

func (bf *banFilter) Do(next types.Handler) types.Handler {
	return func(ctx *fasthttp.RequestCtx) error {
		md := metadata.MDFromContext(ctx)

		banSvc, err := ban.GetBanService()
		if err != nil {
			return next(ctx)
		}
		memberStatus, err := banSvc.GetMemberStatus(ctx, md.UID)
		if err != nil {
			gmetric.IncDefaultError("ban", "member_status")
			return next(ctx)
		}
		its := memberStatus.UserState.GetBanItems()
		for _, it := range its {
			_, ok := bf.br.bts[banTag{
				BizType:  it.GetBizType(),
				TagName:  it.GetTagName(),
				TagValue: it.GetTagValue(),
			}]
			if !ok {
				continue
			}
			if it.GetErrorCode() > 0 {
				return berror.NewBizErr(int64(it.GetErrorCode()), it.GetReasonText())
			} else {
				return berror.ErrOpenAPIUserLoginBanned
			}
		}
		return next(ctx)
	}
}

func (bf *banFilter) GetName() string {
	return filter.BanFilterKey
}

func (bf *banFilter) parseFlag(ctx context.Context, args []string) (banRule, error) {
	rule := banRule{
		bts: make(map[banTag]struct{}),
	}

	var btsStr string

	parse := flag.NewFlagSet("ban", flag.ContinueOnError)
	parse.StringVar(&btsStr, "banTags", "[]", "Configuring matching and banning service triplet.")
	if err := parse.Parse(args[1:]); err != nil {
		glog.Error(ctx, "ban parseFlag error", glog.String("error", err.Error()), glog.Any("args", args))
		return rule, err
	}
	var ts []banTag
	err := json.Unmarshal([]byte(btsStr), &ts)
	if err != nil {
		return rule, err
	}

	for _, t := range ts {
		rule.bts[t] = struct{}{}
	}
	return rule, nil
}
