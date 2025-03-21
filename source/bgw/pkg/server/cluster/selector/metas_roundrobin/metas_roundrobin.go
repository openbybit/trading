package metas_roundrobin

import (
	"context"
	"fmt"
	"strings"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/registry"
	"bgw/pkg/server/cluster"
	gmetadata "bgw/pkg/server/metadata"

	"code.bydev.io/fbu/gateway/gway.git/glog"
	"google.golang.org/grpc/metadata"
)

func init() {
	cluster.Register(constant.MetasRoundRobin, New())
}

type metas struct{}

func New() cluster.Selector {
	return &metas{}
}

func (m *metas) Select(ctx context.Context, ins []registry.ServiceInstance) (registry.ServiceInstance, error) {
	kvs := m.getKeys(ctx)
	if len(kvs) == 0 {
		return nil, berror.ErrParams
	}
	var (
		instances = make([]registry.ServiceInstance, 0, len(ins))
		hit       = make(map[string]struct{})
	)
	kv := make([]string, 0, len(kvs))
	for key, value := range kvs {
		kv = append(kv, key, value)
		glog.Debug(ctx, "select metas", glog.String(key, value))
		if value == "" {
			continue
		}
		value = "," + value + ","
		for _, in := range ins {
			meta := "," + in.GetMetadata().GetDynamicName(key) + ","
			if strings.Contains(meta, value) {
				addr := in.GetID()
				if _, ok := hit[addr]; ok {
					continue
				}
				instances = append(instances, in)
				hit[addr] = struct{}{}
			}
		}
	}
	if len(instances) == 0 {
		kv = append(kv, "kvs is invalid", "service not found", gmetadata.MDFromContext(ctx).Route.Registry)
		return nil, berror.NewUpStreamErr(berror.UpstreamErrInstanceNotFound, kv...)
	}
	r := cluster.GetSelector(ctx, constant.SelectorRoundRobin)
	return r.Select(ctx, instances)
}

func (m *metas) Extract(meta *cluster.ExtractConf) (interface{}, error) {
	if len(meta.SelectKeys) > 0 {
		return meta.SelectKeys, nil
	}
	return nil, fmt.Errorf("invalid SelectKeys, MetasRoundRobin, %s->%s:%s", meta.Registry, meta.ServiceName, meta.MethodName)
}

func (m *metas) Inject(ctx context.Context, metas interface{}) (context.Context, error) {
	if metas == nil {
		return ctx, fmt.Errorf("nothing to inject")
	}
	keys, ok := metas.([]string)
	if !ok || len(keys) == 0 {
		return nil, fmt.Errorf("inject object invalid")
	}
	// parse
	values := m.getValues(ctx, keys)

	origin := make(map[string]string, len(values))
	invalidCount := 0
	for i, value := range values {
		origin[keys[i]] = value
		if value == "" {
			invalidCount++
		}
	}
	if invalidCount == len(keys) {
		return nil, berror.ErrParams
	}

	return gmetadata.ContextWithSelectMetas(ctx, origin), nil
}

func (m *metas) getValues(ctx context.Context, keys []string) (values []string) {
	if len(keys) == 0 {
		return
	}

	if c, ok := ctx.(*types.Ctx); ok {
		for _, key := range keys {
			// peek query string
			v := string(c.QueryArgs().Peek(key))
			if v == "" {
				// peek header
				values = append(values, string(c.Request.Header.Peek(key)))
				continue
			} else {
				values = append(values, v)
			}
		}
		return
	}

	// get value from grpc metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return
	}
	for _, key := range keys {
		if vs := md.Get(key); len(vs) > 0 {
			values = append(values, vs[0])
		} else {
			values = append(values, "")
		}
	}

	return
}

func (m *metas) getKeys(ctx context.Context) map[string]string {
	v := gmetadata.MetasFromContext(ctx)
	if v == nil {
		return nil
	}
	kvs, ok := v.(map[string]string)
	if !ok {
		return nil
	}
	return kvs
}
