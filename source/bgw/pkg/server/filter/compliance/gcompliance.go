package compliance

import (
	"context"

	"bgw/pkg/diagnosis"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcompliance"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/frameworks/byone/kafka"
	"code.bydev.io/frameworks/byone/zrpc"

	"bgw/pkg/common/kafkaconsume"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
	"bgw/pkg/service"
)

const (
	topicWhitelist = "cht-compliance-wall-whitelist"
	topicKyc       = "cht-compliance-wall-kyc"
	topicStrategy  = "cht-compliance-wall-strategy"
	topicEvent     = "cht-compliance-wall-event"
)

var (
	gw gcompliance.Wall
)

func initComplianceService() error {
	rpcClient, err := zrpc.NewClient(config.Global.Compliance, zrpc.WithDialOptions(service.DefaultDialOptions...))
	if err != nil {
		glog.Errorf(context.Background(), " compliance rpc dial fail, error=%v", err)
		galert.Error(context.Background(), "compliance rpc dial fail", galert.WithField("error", err))
		return err
	}
	gw, err = gcompliance.NewWall(gcompliance.NewConnCfg(rpcClient.Conn()), true)

	if err != nil {
		galert.Error(context.Background(), "[compliance filter]NewWall err"+err.Error())
		return err
	}

	topics := []string{topicWhitelist, topicKyc, topicStrategy, topicEvent}
	handlers := []kafkaconsume.Handler{handleWhiteMsg, handleUserKycMsg, handleStrategyMSg, handleSiteConfigMsg}
	for i, _ := range topics {
		kafkaconsume.AsyncHandleKafkaMessage(context.Background(), topics[i], config.Global.ComplianceKafkaCli, handlers[i], onErr)
	}

	registerAdmin()
	_ = diagnosis.Register(&diagnose{
		svc:  gw,
		cfg:  config.Global.Compliance,
		kCfg: config.Global.ComplianceKafkaCli,
	})
	return nil
}

func handleWhiteMsg(ctx context.Context, msg *kafka.Message) {
	glog.Info(ctx, "compliance_filter whitelist msg", glog.String("whiteList", string(msg.Value)), glog.Int64("offset", msg.Offset))
	err := gw.HandleUserWhiteListEvent(msg.Value)
	if err != nil {
		glog.Error(ctx, "compliance_filter whitelist msg err", glog.Int64("offset", msg.Offset), glog.String("err", err.Error()))
	}
}

func handleUserKycMsg(ctx context.Context, msg *kafka.Message) {
	glog.Info(ctx, "compliance_filter userKyc msg", glog.String("userKyc", string(msg.Value)), glog.Int64("offset", msg.Offset))
	err := gw.HandleUserKycEvent(msg.Value)
	if err != nil {
		glog.Error(ctx, "compliance_filter userKyc msg err", glog.Int64("offset", msg.Offset), glog.String("err", err.Error()))
	}
}

func handleStrategyMSg(ctx context.Context, msg *kafka.Message) {
	glog.Info(ctx, "compliance_filter strategy msg", glog.String("strategy", string(msg.Value)), glog.Int64("offset", msg.Offset))
	err := gw.HandleStrategyEvent(msg.Value)
	if err != nil {
		glog.Error(ctx, "compliance_filter strategy msg err", glog.Int64("offset", msg.Offset), glog.String("err", err.Error()))
	}
}

func handleSiteConfigMsg(ctx context.Context, msg *kafka.Message) {
	glog.Info(ctx, "compliance_filter site config msg", glog.String("siteCfg", string(msg.Value)), glog.Int64("offset", msg.Offset))
	err := gw.HandleSiteConfigEvent(msg.Value)
	if err != nil {
		glog.Error(ctx, "compliance_filter site config msg err", glog.Int64("offset", msg.Offset), glog.String("err", err.Error()))
	}
}

func onErr(err *kafka.ConsumerError) {
	if err != nil {
		galert.Error(context.Background(), "compliance consumer err "+err.Error())
	}
}

type complianceResult struct {
	Config rawResult `json:"config"`
}

type rawResult struct {
	EndpointExec string   `json:"endpointExec"`
	EndpointArgs struct{} `json:"endpointArgs"`
}

func marshalComplianceResult(result gcompliance.Result) ([]byte, error) {
	resp := complianceResult{
		rawResult{
			EndpointExec: result.GetEndPointExec(),
			EndpointArgs: struct{}{},
		},
	}

	return util.JsonMarshal(&resp)
}

type diagnose struct {
	svc  gcompliance.Wall
	cfg  zrpc.RpcClientConf
	kCfg kafka.UniversalClientConfig
}

func (o *diagnose) Key() string {
	return "compliance_wall"
}

func (o *diagnose) Diagnose(ctx context.Context) (interface{}, error) {
	resp := make(map[string]interface{})
	resp["kafka_whitelist_topic"] = diagnosis.DiagnoseKafka(ctx, topicWhitelist, o.kCfg)
	resp["kafka_kyc_type_topic"] = diagnosis.DiagnoseKafka(ctx, topicKyc, o.kCfg)
	resp["kafka_strategy_topic"] = diagnosis.DiagnoseKafka(ctx, topicStrategy, o.kCfg)
	resp["kafka_site_topic"] = diagnosis.DiagnoseKafka(ctx, topicEvent, o.kCfg)
	resp["grpc"] = diagnosis.DiagnoseGrpcDependency(ctx, o.cfg)
	return resp, nil
}
