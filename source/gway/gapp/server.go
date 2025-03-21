package gapp

import (
	"fmt"
	"log"
	"net"
)

type Server interface {
	Serve(ln net.Listener) error
}

// Serve 辅助方法,自动执行Listen并支持异步Serve
// grpc, fasthttp均实现Server接口,在启动服务时,通常需要异步执行,
func Serve(s Server, network string, addr string, async bool) error {
	ln, err := net.Listen(network, addr)
	if err != nil {
		return fmt.Errorf("gapp listen fail, network: %s, addr: %s, err: %w", network, addr, err)
	}

	if async {
		go func() {
			if err := s.Serve(ln); err != nil {
				log.Printf("serve fail, err: %v", err)
			}
		}()

		return nil
	} else {
		return s.Serve(ln)
	}
}
