package symbols_roundrobin

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/bhttp"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/registry"
	"bgw/pkg/server/cluster"
	gmetadata "bgw/pkg/server/metadata"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	"google.golang.org/grpc/metadata"
)

const (
	Symbol = "symbol"
)

func init() {
	cluster.Register(constant.SymbolsRoundRobin, New())
}

type symbols struct {
}

func New() cluster.Selector {
	return &symbols{}
}

func (s *symbols) Select(ctx context.Context, ins []registry.ServiceInstance) (registry.ServiceInstance, error) {
	if len(ins) == 0 {
		return nil, cluster.ErrServiceNotFound
	}
	symbol, ok := gmetadata.MetasFromContext(ctx).(string)
	if !ok {
		return nil, fmt.Errorf("get symbol from context error, not string")
	}

	r := cluster.GetSelector(ctx, constant.SelectorRoundRobin)
	if symbol == "" {
		glog.Debug(ctx, "request symbol invalid, will the default instance")
		return r.Select(ctx, ins)
	}

	var instances []registry.ServiceInstance
	key := "," + symbol + ","
	for _, instance := range ins {
		im := instance.GetMetadata()
		if strings.Contains(im.GetSymbolsName(), key) {
			instances = append(instances, instance)
		}
	}

	if len(instances) == 0 {
		glog.Debug(ctx, "hit the default instance", glog.String("symbol", symbol))
		return r.Select(ctx, ins)
	}
	glog.Debug(ctx, "all instance", glog.Any("ins", instances))

	return r.Select(ctx, instances)
}

func (s *symbols) Inject(ctx context.Context, _ interface{}) (context.Context, error) {
	symbol := getSymbolFromRequest(ctx)
	if symbol == "" {
		return nil, berror.ErrInvalidSymbol
	}
	return gmetadata.ContextWithSelectMetas(ctx, symbol), nil
}

func getSymbolFromRequest(ctx context.Context) (symbol string) {
	c, ok := ctx.(*types.Ctx)
	if !ok {
		// get symbol from grpc
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return
		}
		if s := md.Get(Symbol); len(s) > 0 {
			return s[0]
		}
		return
	}

	defer func() {
		if symbol != "" {
			symbol = strings.ToLower(strings.TrimSpace(symbol))
		}
	}()

	if !c.IsGet() && !bytes.HasPrefix(c.Request.Header.ContentType(), bhttp.ContentTypePostForm) {
		symbol = util.JsonGetString(c.PostBody(), Symbol)
		return
	}

	if c.IsGet() {
		return string(c.QueryArgs().Peek(Symbol))
	} else {
		return string(c.PostArgs().Peek(Symbol))
	}
}
