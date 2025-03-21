package ws

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gcore/nets"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	envelopev1 "code.bydev.io/fbu/gateway/proto.git/pkg/envelope/v1"
	bn "code.bydev.io/frameworks/byone/core/discov/nacos"
	"code.bydev.io/frameworks/byone/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"bgw/pkg/common/constant"
)

type grpcServer struct {
	envelopev1.UnimplementedEnvelopeServiceServer

	srv                  *zrpc.RpcServer
	registryDefaultGroup *bn.Register
}

func (s *grpcServer) Start() error {
	conf := getStaticConf().RPC
	glog.Infof(context.TODO(),
		"start grpc, type=%v, tcp_port=%v, unix_addr=%v",
		conf.ListenType,
		conf.ListenTcpPort,
		conf.ListenUnixAddr,
	)

	cluster := getAppConf().Cluster
	unixEnable := conf.ListenType != listenTypeTcp
	unixAddr := conf.ListenUnixAddr
	tcpPort := conf.ListenTcpPort

	sconf := zrpc.RpcServerConf{
		ListenOn: toListenAddress(tcpPort),
		Nacos: bn.NacosConf{
			Key:       serviceName,
			Group:     cluster,
			NacosConf: getStaticConf().Nacos,
		},
	}

	glog.Infof(context.Background(), "grpc serve tcp, %v", sconf.ListenOn)

	register := func(srv *grpc.Server) {
		envelopev1.RegisterEnvelopeServiceServer(srv, s)
		if unixEnable {
			go func() {
				glog.Infof(context.Background(), "start to serve unix, %v", unixAddr)
				unixDir := filepath.Dir(unixAddr)
				_ = os.MkdirAll(unixDir, os.ModePerm)
				_ = os.Remove(unixAddr)
				ln, err := net.Listen("unix", unixAddr)
				if err != nil {
					panic(fmt.Errorf("listen unix fail: %v, %v", unixAddr, err))
				}

				if err := srv.Serve(ln); err != nil {
					panic(fmt.Errorf("serve unix fail: %v, %v", unixAddr, err))
				}
			}()
		}
	}

	s.srv = zrpc.MustNewServer(sconf, register, zrpc.WithMetadata(getMetadata(cluster)))

	go s.srv.Start()

	// 主网site需要单独注册一下default_group,全部迁移完后会下线
	defaultGroup := false
	if cluster == "site" {
		defaultGroup = true
	}

	if defaultGroup {
		glog.Info(context.TODO(), "register default group")
		dr, err := buildRegister(tcpPort, cluster, constant.DEFAULT_GROUP)
		if err != nil {
			glog.Error(context.Background(), "register nacos fail, NewNacosServiceDiscovery", glog.String("error", err.Error()))
			return err
		}
		s.registryDefaultGroup = dr
		if err := s.registryDefaultGroup.Register(); err != nil {
			glog.Error(context.Background(), "register default nacos fail", glog.String("error", err.Error()))
			return err
		}
	}

	glog.Infof(context.TODO(), "start grpc server success, register default group: %v", defaultGroup)
	return nil
}

func (s *grpcServer) Stop() {
	if s.registryDefaultGroup != nil {
		glog.Info(context.Background(), "start to stop default registry")
		if err := s.registryDefaultGroup.Stop(); err != nil {
			glog.Error(context.TODO(), "registryDefaultGroup.Stop default group fail", glog.String("error", err.Error()))
		}
	}

	if s.srv != nil {
		s.srv.Stop()
	}
}

// Subscribe impl the grpc service
func (s *grpcServer) Subscribe(server envelopev1.EnvelopeService_SubscribeServer) error {
	msg, err := server.Recv()
	if err != nil {
		glog.Error(context.Background(), "receive dispatcher info error", glog.String("error", err.Error()))
		return err
	}

	if err := s.checkSubscribeConfig(msg); err != nil {
		return err
	}

	appId := msg.Header.AppId
	id := msg.Header.ConnectorId
	shardIndex := msg.Header.UserShardIndex
	shardTotal := msg.Header.UserShardTotal
	focusEvents := msg.Header.FocusEvents
	extensions := msg.Header.Extensions
	addr := ""
	if p, ok := peer.FromContext(server.Context()); ok && p != nil {
		addr = p.Addr.String()
	}
	privateTopics := msg.Topics
	publicTopics := []string{}
	publicConfigs := []PublicTopicConf{}

	for _, t := range msg.Header.TopicConfigs {
		t.Name = strings.TrimSpace(t.Name)
		if len(t.Name) == 0 {
			continue
		}
		if t.Type == envelopev1.PushType_PUSH_TYPE_PRIVATE {
			privateTopics = append(privateTopics, t.Name)
		} else {
			publicTopics = append(publicTopics, t.Name)
			publicConfigs = append(publicConfigs, PublicTopicConf{Topic: t.Name, PushMode: t.Mode})
		}
	}

	privateTopics = distinctString(privateTopics)

	glog.Info(context.Background(),
		"subscribe new acceptor",
		glog.String("id", id),
		glog.String("app_id", appId),
		glog.String("addr", addr),
		glog.Any("private_topics", privateTopics),
		glog.Any("public_topics", publicTopics),
		glog.Int64("user_shard_index", shardIndex),
		glog.Int64("user_shard_total", shardTotal),
		glog.Uint64("focus_events", focusEvents),
		glog.Any("extensions", extensions),
	)

	accOpts := &acceptorOptions{ShardIndex: int(shardIndex), ShardTotal: int(shardTotal), FocusEvents: int(focusEvents), Address: addr, Extensions: extensions, PublicTopics: publicTopics}
	acceptor := newAcceptor(server, id, appId, privateTopics, accOpts)
	if err := gAcceptorMgr.Add(acceptor); err != nil {
		glog.Errorf(context.Background(), "add acceptor fail: id=%v, appId=%v, err=%v", id, appId, err)
		return status.Error(codes.InvalidArgument, err.Error())
	}

	acceptor.Start()

	DispatchEvent(&SyncConfigEvent{acceptorID: acceptor.ID()})
	DispatchEvent(NewSyncAllUserEvent(acceptor.ID()))

	getConfigMgr().AddTopic(topicTypePrivate, privateTopics)
	getConfigMgr().AddTopic(topicTypePublic, publicTopics)
	gPublicMgr.Run(publicConfigs)

	gAcceptorMgr.RefreshAppIDGauge(appId)
	acceptor.Wait()
	gAcceptorMgr.Remove(id)
	gAcceptorMgr.RefreshAppIDGauge(appId)
	glog.Info(context.Background(), "acceptor closed", glog.String("id", id), glog.String("app_id", appId), glog.String("addr", addr), glog.Any("topics", privateTopics), glog.Any("extensions", extensions))
	return nil
}

func (s *grpcServer) checkSubscribeConfig(msg *envelopev1.SubscribeRequest) error {
	h := msg.Header
	if msg.Header == nil {
		return status.Errorf(codes.InvalidArgument, "no header")
	}

	if h.ConnectorId == "" {
		return status.Errorf(codes.InvalidArgument, "empty connector id")
	}

	if h.UserShardTotal > 0 && (h.UserShardIndex < 0 || h.UserShardIndex > h.UserShardTotal) {
		WSCounterInc("server", "invalid_shard_index")
		glog.Info(context.Background(), "invalid shard index config",
			glog.String("app_id", h.AppId),
			glog.Int64("user_shard_index", h.UserShardIndex),
			glog.Int64("user_shard_total", h.UserShardTotal),
		)
	}

	return nil
}

// buildRegister build register
func buildRegister(port int, cluster, group string) (*bn.Register, error) {
	glog.Info(context.TODO(), "grpc Server buildRegister nacos")
	ip := nets.GetLocalIP()
	cfg := getStaticConf().Nacos
	namespace := cfg.NamespaceId

	cc, err := cfg.BuildNamingClient()
	if err != nil {
		glog.Error(context.TODO(), "BuildNamingClient error", glog.String("namespace", namespace), glog.NamedError("err", err))
		return nil, err
	}
	glog.Info(context.TODO(), "Server BuildNamingClient ok", glog.String("namespace", namespace))

	md := getMetadata(cluster)
	n, err := bn.NewRegister(cc, "bgws", fmt.Sprintf("%s:%d", ip, port), group, bn.WithMetadata(md))
	if err != nil {
		glog.Error(context.TODO(), "NewRegister error", glog.String("namespace", namespace), glog.NamedError("err", err))
		return nil, err
	}

	return n, nil
}

func getMetadata(cluster string) map[string]string {
	const (
		createTime     = "create_time"
		lastModifyTime = "last_modify_time"
		registerDate   = "register_date"
		language       = "language"
		az             = "az"
		cloudName      = "cloud_name"
		clusterName    = "cluster"
	)

	now := cast.ToString(time.Now().UnixMilli())
	md := map[string]string{
		createTime:     now,
		lastModifyTime: now,
		registerDate:   time.Now().Format(time.RFC3339),
		language:       "golang",
	}
	if cluster != "" {
		md[clusterName] = cluster
	}
	azz := env.AvailableZoneID()
	if azz != "" {
		md[az] = azz
	}
	cn := env.CloudProvider()
	if cn != "" {
		md[cloudName] = cn
	}
	return md
}
