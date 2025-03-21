package gcompliance

import (
	"context"

	"google.golang.org/grpc"
)

type Discovery = func(ctx context.Context, registry, namespace, group string) (addrs []string)

// RemoteCfg is necessary config of compliance server.
type RemoteCfg interface {
	Addr() string
	Registry() string
	Namespace() string
	Group() string
	Discovery() Discovery
	Conn() grpc.ClientConnInterface
}

type addrCfg struct {
	addr string
}

// NewAddrCfg directly use addr as remote config.
func NewAddrCfg(addr string) RemoteCfg {
	return &addrCfg{addr: addr}
}

func (a *addrCfg) Addr() string { return a.addr }

func (a *addrCfg) Registry() string { return "" }

func (a *addrCfg) Namespace() string { return "" }

func (a *addrCfg) Group() string { return "" }

func (a *addrCfg) Discovery() Discovery { return nil }

func (a *addrCfg) Conn() grpc.ClientConnInterface { return nil }

type registryCfg struct {
	registry  string
	namespace string
	group     string
	discovery Discovery
}

// NewRegistryCfg use service discovery as remote config.
func NewRegistryCfg(registry, namespace, group string, discovery Discovery) RemoteCfg {
	return &registryCfg{
		registry:  registry,
		namespace: namespace,
		group:     group,
		discovery: discovery,
	}
}

func (r *registryCfg) Registry() string { return r.registry }

func (r *registryCfg) Namespace() string { return r.namespace }

func (r *registryCfg) Group() string { return r.group }

func (r *registryCfg) Addr() string { return "" }

func (r *registryCfg) Discovery() Discovery { return r.discovery }

func (r *registryCfg) Conn() grpc.ClientConnInterface { return nil }

type connCfg struct {
	conn grpc.ClientConnInterface
}

func NewConnCfg(conn grpc.ClientConnInterface) RemoteCfg {
	return &connCfg{conn: conn}
}

func (c *connCfg) Addr() string { return "" }

func (c *connCfg) Registry() string { return "" }

func (c *connCfg) Namespace() string { return "" }

func (c *connCfg) Group() string { return "" }

func (c *connCfg) Discovery() Discovery { return nil }

func (c *connCfg) Conn() grpc.ClientConnInterface { return c.conn }
