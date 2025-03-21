package generic

import (
	"fmt"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cache/lru"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
)

type protoTypeEntry struct {
	sd  *desc.ServiceDescriptor
	md  *desc.MethodDescriptor
	ext *dynamic.ExtensionRegistry
}

type protoTypesCache struct {
	cache *lru.Cache
}

func newProtoTypeCache() *protoTypesCache {
	c, err := lru.NewLRU(4094)
	if err != nil {
		return nil
	}

	p := &protoTypesCache{
		cache: c,
	}
	return p
}

func (p *protoTypesCache) get(key string) (*protoTypeEntry, bool) {
	value, ok := p.cache.Get(key)
	if !ok {
		return nil, false
	}

	rr, ok := value.(*protoTypeEntry)
	return rr, ok
}

func (p *protoTypesCache) set(key string, typ *protoTypeEntry) {
	p.cache.Add(key, typ)
}

func (p *protoTypesCache) makeKey(service, method string) string {
	return fmt.Sprintf("%s/%s", service, method)
}
