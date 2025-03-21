package consistent_hash

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/conhash"
	"code.bydev.io/fbu/gateway/gway.git/glog"

	"bgw/pkg/common/constant"
	"bgw/pkg/registry"
	"bgw/pkg/server/cluster"
	"bgw/pkg/server/metadata"
)

const (
	UID   = "uid"
	IP    = "ip"
	UIDIP = "uid/ip"
)

func init() {
	cluster.Register(constant.SelectorConsistentHash, New())
}

type ConHash struct {
	sync.RWMutex
	Cycles map[string]*conhash.Consistent
}

func New() cluster.Selector {
	return &ConHash{
		Cycles: make(map[string]*conhash.Consistent),
	}
}

func (c *ConHash) Select(ctx context.Context, ins []registry.ServiceInstance) (registry.ServiceInstance, error) {
	count := len(ins)
	if count == 0 {
		return nil, cluster.ErrServiceNotFound
	}
	if count == 1 {
		return ins[0], nil
	}

	// get hash key
	key, err := getHashKey(ctx)
	if err != nil {
		return nil, err
	}

	element := make([]string, 0, len(ins))
	instance := make(map[string]registry.ServiceInstance, len(ins))
	for _, i := range ins {
		element = append(element, i.GetID())
		instance[i.GetID()] = i
	}

	// get hash cycle by upstream service
	service := ins[0].GetServiceName()
	c.RLock()
	hashCycle, ok := c.Cycles[service]
	c.RUnlock()
	if !ok {
		c.Lock()
		hashCycle, ok = c.Cycles[service]
		if !ok {
			hashCycle = conhash.New()
			c.Cycles[service] = hashCycle
		}
		c.Unlock()
	}
	if hashCycle.Diff(element) {
		hashCycle.Set(element)
	}

	id, err := hashCycle.Get(key)
	if err != nil {
		return nil, err
	}

	return instance[id], nil
}

func (c *ConHash) Extract(meta *cluster.ExtractConf) (interface{}, error) {
	if len(meta.SelectKeys) > 0 {
		return meta.SelectKeys[:1], nil
	}
	glog.Debug(context.TODO(), fmt.Sprintf("invalid SelectKeys, ConHash, use uid/ip, %s->%s:%s", meta.Registry, meta.ServiceName, meta.MethodName))
	return []string{UIDIP}, nil
}

func (c *ConHash) Inject(ctx context.Context, metas interface{}) (context.Context, error) {
	if metas == nil {
		return ctx, fmt.Errorf("nothing to inject")
	}
	keys, ok := metas.([]string)
	if !ok || len(keys) == 0 {
		return nil, fmt.Errorf("inject object invalid, ConHash")
	}

	return metadata.ContextWithSelectMetas(ctx, keys), nil
}

func getHashKey(ctx context.Context) (string, error) {
	selectMeta := metadata.MetasFromContext(ctx)
	keys, ok := selectMeta.([]string)
	if !ok || len(keys) != 1 {
		return "", fmt.Errorf("[conHash selector]invalid select metas")
	}

	b := &strings.Builder{}
	meta := metadata.MDFromContext(ctx)
	for _, key := range keys {
		switch key {
		case UID:
			b.WriteString(cast.Int64toa(meta.UID))
		case IP:
			b.WriteString(meta.Extension.RemoteIP)
		case UIDIP:
			if meta.UID > 0 {
				b.WriteString(cast.Int64toa(meta.UID))
			} else {
				b.WriteString(meta.Extension.RemoteIP)
			}
		default:
			return "", fmt.Errorf("conHash unsupport selector key, %s", key)
		}
	}

	return b.String(), nil
}
