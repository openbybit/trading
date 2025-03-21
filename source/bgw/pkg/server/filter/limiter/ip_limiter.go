package limiter

import (
	"context"
	"flag"
	"strings"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/types"
	"bgw/pkg/server/filter"
	"bgw/pkg/server/metadata"
)

type ipLimiter struct {
	allowedIP string
}

func newIPLimiter() filter.Filter {
	return &ipLimiter{}
}

// GetName returns the name of the filter
func (*ipLimiter) GetName() string {
	return filter.IPRateLimitFilterKey
}

// Do the filter do
func (i *ipLimiter) Do(next types.Handler) types.Handler {
	return func(c *types.Ctx) error {
		if i.allowedIP != "" {
			md := metadata.MDFromContext(c)
			if !strings.Contains(i.allowedIP, md.Extension.RemoteIP) {
				return berror.ErrInvalidIP
			}
		}

		return next(c)
	}
}

// Init the filter init
func (i *ipLimiter) Init(_ context.Context, args ...string) (err error) {
	if len(args) == 0 {
		return
	}

	parse := flag.NewFlagSet("ip_limiter", flag.ContinueOnError)
	parse.StringVar(&i.allowedIP, "allowIPs", "", "allow ips")

	return parse.Parse(args[1:])
}
