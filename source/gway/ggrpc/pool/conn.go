package pool

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// Conn single grpc connection interface
//
//go:generate mockgen -source=conn.go -destination=conn_mock.go -package=pool
type Conn interface {
	// Client return the actual grpc connection type *grpc.ClientConn.
	Client() *grpc.ClientConn

	// Close decrease the reference of grpc connection, instead of close it.
	// if the pool is full, just close it.
	Close() error

	validState() bool
}

// Conn is wrapped grpc.ClientConn. to provide close and value method.
type conn struct {
	cc   *grpc.ClientConn
	pool *pool
	once bool
}

// Client see Conn interface.
func (c *conn) Client() *grpc.ClientConn {
	return c.cc
}

// Close see Conn interface.
func (c *conn) Close() error {
	if c == nil || c.pool == nil {
		return nil
	}
	c.pool.decrRef()
	if c.once {
		return c.reset()
	}
	return nil
}

func (c *conn) reset() error {
	cc := c.cc
	c.cc = nil
	c.pool = nil
	c.once = false
	if cc != nil {
		return cc.Close()
	}
	return nil
}

func (p *pool) wrapConn(cc *grpc.ClientConn, once bool) *conn {
	return &conn{
		cc:   cc,
		pool: p,
		once: once,
	}
}

// Reset will restart the grpc connection backoff
func (c *conn) Reset() {
	if c == nil || c.cc == nil {
		return
	}
	c.cc.ResetConnectBackoff()
}

// ValidState check grpc state
func (c *conn) validState() bool {
	if c == nil || c.cc == nil {
		return false
	}

	if !isValidState(c.cc) {
		return false
	}

	return true
}

// isValidState check grpc state
func isValidState(c *grpc.ClientConn) bool {
	switch c.GetState() {
	case connectivity.TransientFailure, connectivity.Shutdown:
		return false
	}

	return true
}
