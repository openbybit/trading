package compliance

import (
	"bytes"

	"code.bydev.io/fbu/gateway/gway.git/glog"

	"bgw/pkg/common/bhttp"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/server/metadata"
	sc "bgw/pkg/service/symbolconfig"
)

func (c *complianceWall) getScene(ctx *types.Ctx) string {
	md := metadata.MDFromContext(ctx)
	// original scene
	scene := c.SceneCode
	glog.Debug(ctx, "multi scenes", glog.String("app", md.Route.GetAppName(ctx)), glog.Any("cfg", c.multiScenes))
	// match aio scene
	if c.multiScenes != nil {
		s, ok := c.multiScenes[md.Route.GetAppName(ctx)]
		if ok {
			scene = s
		}
	}

	// match slt scene
	if c.SLT != nil && SLTMatch(ctx, c.SLT.Category, c.SLT.SymbolField, md) {
		scene = c.SLT.Scene
	}

	return scene
}

const spotLeveragedType = 2

func SLTMatch(ctx *types.Ctx, category, symbolFiled string, md *metadata.Metadata) bool {
	if category != "" && category != md.Route.GetAppName(ctx) {
		return false
	}

	var symbol string
	if !ctx.IsGet() && !bytes.HasPrefix(ctx.Request.Header.ContentType(), bhttp.ContentTypePostForm) {
		symbol = util.JsonGetString(ctx.PostBody(), symbolFiled)
	} else if ctx.IsGet() {
		symbol = string(ctx.QueryArgs().Peek(symbolFiled))
	} else {
		symbol = string(ctx.PostArgs().Peek(symbolFiled))
	}

	glog.Debug(ctx, "spot leveraged match", glog.String("symbol", symbol))
	sm := sc.GetSpotManager()
	if sm == nil {
		glog.Debug(ctx, "gsymbol get config nil")
		return false
	}

	cfg := sm.GetByName(symbol)
	glog.Debug(ctx, "spot leveraged match", glog.Any("symbol cfg", cfg))
	if cfg == nil {
		return false
	}

	return cfg.SymbolType == spotLeveragedType
}
