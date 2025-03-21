package biz_limiter

import (
	"strings"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"

	"bgw/pkg/common/types"
	"bgw/pkg/server/metadata"
)

// extractor data from types.Ctx, General data acquisition
type extractor struct {
	UID    bool   `json:"uid"`
	Path   bool   `json:"path"`
	Method bool   `json:"method"`
	Group  string `json:"group"`
	Symbol bool   `json:"symbol"`
	IP     bool   `json:"ip"`
}

// Values Get data from types.Ctx , According to the RuleExtractor config
func (d *extractor) Values(ctx *types.Ctx, symbol string) []string {
	md := metadata.MDFromContext(ctx)

	var keys []string

	if d.Path {
		keys = append(keys, strings.TrimPrefix(md.Path, "/"))
	}

	if d.Method {
		keys = append(keys, md.Method)
	}

	if d.Group != "" {
		keys = append(keys, d.Group)
	}

	if d.UID {
		keys = append(keys, cast.Int64toa(md.UID))
	}

	if d.Symbol {
		keys = append(keys, symbol)
	}

	if d.IP {
		keys = append(keys, md.GetClientIP())
	}

	return keys
}

func getUnifiedKey(uid int64, group string) string {
	return "unified:" + group + ":" + cast.Int64toa(uid)
}

func getOptionKey(uid int64, group string) string {
	return "option:" + group + ":" + cast.Int64toa(uid)
}
