package ws

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/deadlock"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"github.com/rs/xid"
)

const (
	defaultQuitWaitTime = time.Second * 20
)

// globalNodeID 全局bws nodeid
var globalNodeID = xid.New().String()

var errServerStopped = errors.New("server stopped")

// New create a new service
func New() *Server {
	return &Server{}
}

// Server is a gRPC server
// go test -v bgw/pkg/server/ws -covermode=count -coverpkg bgw/pkg/server/ws
type Server struct {
	running int32
	ws      wsServer
	grpc    grpcServer
}

func (s *Server) Health() bool {
	return s.isRunning()
}

// State cache state
func (s *Server) State() interface{} {
	st := Status{Health: s.Health()}
	s.ws.FillStatus(&st)
	gSessionMgr.FillStatus(&st)
	gUserMgr.FillStatus(&st)
	return &st
}

func (s *Server) isRunning() bool {
	return atomic.LoadInt32(&s.running) == 1
}

// Start do start websocket service
func (s *Server) Start() error {
	appConf := getAppConf()
	setupAdmin()

	initMetrics(appConf.Cluster)
	setUserServiceRateLimit(appConf.UserSvcRateLimit)

	deadlockBuf := &logWriter{}
	deadlock.Opts.Disable = appConf.DisableDeadlockCheck
	deadlock.Opts.LogBuf = deadlockBuf
	deadlock.Opts.OnPotentialDeadlock = func() {
		WSErrorInc("service", "deadlock")
		galert.Error(context.Background(), deadlockBuf.String(), galert.WithTitle("[BGWS] has deadlock"))
	}

	initExchange()
	gTickerMgr.Start()
	gMetricsMgr.Start()

	if err := s.grpc.Start(); err != nil {
		return err
	}

	if err := s.ws.Start(); err != nil {
		return err
	}

	atomic.StoreInt32(&s.running, 1)
	glog.Infof(context.Background(), "[BGWS] start ok, cluster=%v, nodeid=%v", appConf.Cluster, globalNodeID)

	return nil
}

func (s *Server) Stop() error {
	if !atomic.CompareAndSwapInt32(&s.running, 1, 0) {
		return nil
	}
	glog.Info(context.Background(), "bgws try Unregister")
	s.ws.Unregister()
	wait := getAppConf().StopWaitTime
	glog.Info(context.Background(), "bgws try stop", glog.Duration("wait", wait))
	time.Sleep(wait)

	glog.Info(context.Background(), "start to stop websocket server")
	s.ws.Stop()
	gSessionMgr.Close()

	glog.Info(context.Background(), "start to stop grpc server")
	s.grpc.Stop()
	gAcceptorMgr.Close()

	glog.Info(context.Background(), "start to stop ticker_mgr")
	gTickerMgr.Stop()
	glog.Info(context.Background(), "start to stop metrics_mgr")
	gMetricsMgr.Stop()
	glog.Info(context.Background(), "start to stop public_mgr")
	gPublicMgr.Stop()

	glog.Info(context.Background(), "stop finish")

	return nil
}
