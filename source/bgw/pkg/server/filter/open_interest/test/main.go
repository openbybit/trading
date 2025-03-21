package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"code.bydev.io/fbu/gateway/gway.git/gopeninterest"
	"code.bydev.io/fbu/gateway/gway.git/gsymbol/future"
)

func main() {
	temp := "k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9090,k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9091,k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9092"
	k123 := strings.Split(temp, ",")
	temp = "k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9090,k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9091,k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9092"
	kabc := strings.Split(temp, ",")
	temp = "k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9090,k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9091,k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9092,k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9092"
	kusdc := strings.Split(temp, ",")
	if len(k123) == 0 || len(kabc) == 0 || len(kusdc) == 0 {
		log.Printf("kafka config is nil")
		return
	}

	cfg := &gopeninterest.Config{
		K123Brokers:  k123,
		KabcBrokers:  kabc,
		KusdcBrokers: kusdc,

		TopicNameTpl:         "open_interest_exceeded_result.%s",
		EnableLogResult:      false,
		EnableInverseCoin:    true,
		EnableLinearUSDTCoin: true,
		EnableLinearUSDCCoin: true,
	}

	sc, err := getSymbolConfig()
	if err != nil {
		log.Printf("get symbol config failed, err %s", err.Error())
		return
	}

	limiter, err := gopeninterest.New(context.Background(), sc, cfg)
	log.Println("init limiter succeed:", err)

	h := &handler{
		limiter,
		sc,
	}

	err = http.ListenAndServe("127.0.0.1:9090", h)
	if err != nil {
		log.Printf("----------- serve err, %s", err.Error())
	}
}

type handler struct {
	L gopeninterest.Limiter
	S *future.Scmeta
}

type Req struct {
	Uid    int64
	Symbol string
	Side   int32
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	req := &Req{}
	_ = json.Unmarshal(body, req)
	symbol := h.S.SymbolFromName(req.Symbol)
	resp := "not limit"
	if h.L.Limit(req.Uid, int32(symbol), req.Side) {
		resp = "limit"
	}
	_, _ = w.Write([]byte(resp))
}

const (
	ServerName         = "bgw"
	ResultTopicName    = "symbol_config_result"
	ResultAckTopicName = "symbol_config_result_ack"
)

func getSymbolConfig() (*future.Scmeta, error) {
	addr := "k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9090,k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9091,k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:9092"
	addrs := strings.Split(addr, ",")

	cfg := &future.Config{
		Server:           ServerName,
		ResultTopic:      ResultTopicName,
		ResultAckTopic:   ResultAckTopicName,
		Addr:             addrs,
		LogResult:        false,
		AllBrokerSymbols: true,
	}
	sc, err := future.New(context.Background(), cfg)
	return sc, err
}
