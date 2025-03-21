package future

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	modelsv1 "code.bydev.io/fbu/future/bufgen.git/pkg/bybit/future/models/v1"
	"code.bydev.io/fbu/future/sdk.git/pkg/future"
	"code.bydev.io/frameworks/sarama"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

type baseSymbol struct {
	Service        string
	ResultTopic    string
	ResultAckTopic string
	EnableLog      bool
	addr           []string // kafka addr
	kfkCli         sarama.Client
	consumer       sarama.Consumer

	ls   []Listener
	lsCh chan Listener

	sync.RWMutex
	symbols map[future.Symbol]*modelsv1.SymbolConfig
	latest  *modelsv1.SymbolConfigResult
	enabled bool
	events  chan map[future.Symbol]*modelsv1.SymbolConfig
}

type Listener interface {
	OnEvent(map[future.Symbol]*modelsv1.SymbolConfig) error
}

func newBaseSymbol(ctx context.Context, cfg *Config) (*baseSymbol, error) {
	kfkCli, err := sarama.NewClient(cfg.Addr, newConfig())
	if err != nil {
		return nil, err
	}

	cumer, err := sarama.NewConsumerFromClient(kfkCli)
	if err != nil {
		_ = kfkCli.Close()
		return nil, err
	}

	b := &baseSymbol{
		Service:        cfg.Server,
		ResultTopic:    cfg.ResultTopic,
		ResultAckTopic: cfg.ResultAckTopic,
		EnableLog:      cfg.LogResult,
		addr:           cfg.Addr,
		kfkCli:         kfkCli,
		consumer:       cumer,
		events:         make(chan map[future.Symbol]*modelsv1.SymbolConfig, 1),
		lsCh:           make(chan Listener, 1),
	}

	err = b.Consume(ctx)
	if err != nil {
		return nil, err
	}
	go b.Listen(ctx)

	return b, nil
}

func (b *baseSymbol) Register(l Listener) {
	b.lsCh <- l
}

func (b *baseSymbol) Listen(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case l := <-b.lsCh:
			b.RLock()
			cur := b.symbols
			b.RUnlock()
			if cur != nil {
				_ = l.OnEvent(cur)
			}
			b.ls = append(b.ls, l)
		case event := <-b.events:
			b.Lock()
			b.symbols = event
			b.Unlock()
			for _, l := range b.ls {
				_ = l.OnEvent(event)
			}
		}
	}
}

func (b *baseSymbol) Consume(ctx context.Context) error {
	offset, err := b.kfkCli.GetOffset(b.ResultTopic, 0, sarama.OffsetNewest)
	if err != nil {
		return err
	}
	if offset == 0 {
		return errors.New("empty symbol Config queue")
	}

	go func() {
		for {
			offset, err = b.consume(ctx, offset)
			if err != nil {
				log.Printf("[gway]symbol Config consume err ,%s", err.Error())
			}
			log.Println("[gway]symbol Config stop, wait for restart")
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
			}
		}
	}()
	return nil
}

func (b *baseSymbol) consume(ctx context.Context, offset int64) (int64, error) {
	pc, err := b.consumer.ConsumePartition(b.ResultTopic, 0, offset-1)
	if err != nil {
		return offset, err
	}

	for {
		select {
		case msg, ok := <-pc.Messages():
			if !ok {
				return offset, nil
			}
			err = b.handle(ctx, msg)
			if err != nil {
				return offset, err
			}
			offset = msg.Offset + 1
		case <-ctx.Done():
			return offset, ctx.Err()
		}
	}
}

func (b *baseSymbol) handle(ctx context.Context, msg *sarama.ConsumerMessage) error {
	r := new(modelsv1.SymbolConfigResult)
	if err := proto.Unmarshal(msg.Value, r); err != nil {
		return err
	}

	l := b.latest
	// 非首次的时候，如果没有enable到自己，则跳过这个数据。
	if l != nil {
		enable := false
		for _, name := range r.EnableService {
			if b.Service == name {
				enable = true
				break
			}
		}
		if !enable {
			return nil
		}
	}

	r.Offset = msg.Offset
	// 非首次的时候，如果对应的Version的包已经更新过（m.enabled），则跳过。
	if l != nil && l.Version >= r.Version && b.enabled {
		if b.EnableLog {
			log.Printf("scsrc, ignore old version, version %d, known_version %d, offset %d, known_offset %d,"+
				" timestamp %d", r.Version, l.Version, r.Offset, l.Offset, msg.Timestamp.Unix())
		}
		return nil
	}

	if l == nil {
		l = &modelsv1.SymbolConfigResult{}
	}

	log.Printf("scsrc, receiving new version, version %d, known_version %d, offset %d, known_offset %d,"+
		" timestamp %d", r.Version, l.Version, r.Offset, l.Offset, msg.Timestamp.Unix())

	if b.EnableLog {
		bs, err := prototext.Marshal(r)
		if err != nil {
			log.Printf("symbol cfg unmarshal error, %s", err.Error())
		} else {
			// this log nearly 1~4 MB
			log.Printf("symbol cfg result, data %s", string(bs))
		}
	}

	b.enabled = false
	for _, name := range r.EnableService {
		if b.Service == name {
			b.enabled = true
			break
		}
	}

	log.Printf("enable service, version " + strconv.FormatInt(r.Version, 10) +
		", service_name:" + b.Service +
		", m.enabled " + strconv.FormatBool(b.enabled) +
		", enable_services:" + strings.Join(r.EnableService, "|"))

	var incoming map[int32]*modelsv1.SymbolConfig
	if b.enabled {
		log.Println("scsrc, using cur_symbol_config, version " + strconv.FormatInt(r.Version, 10))
		incoming = r.CurSymbolConfig
	} else {
		log.Println("scsrc, using prev_stable_symbol_config, version " + strconv.FormatInt(r.Version, 10))
		incoming = r.PrevStableSymbolConfig
	}

	next := make(map[future.Symbol]*modelsv1.SymbolConfig, len(incoming))

	for _, cfg := range incoming {
		sym := future.Symbol(cfg.Symbol)
		b.RLock()
		cur := b.symbols[sym] // needContinue
		b.RUnlock()
		if cur != nil && cur.Version >= cfg.Version {
			continue
		}

		log.Println("scsrc, incoming symbol " + strconv.Itoa(int(sym)) + ", version " + strconv.Itoa(int(cfg.Version)))

		if len(cfg.RiskLimits) == 0 {
			return fmt.Errorf("empty risk limits, %v, %v", cfg.SymbolName, prototext.Format(cfg))
		}

		next[sym] = cfg
	}

	b.RLock()
	for symbol, cfg := range b.symbols {
		if _, ok := next[symbol]; !ok {
			next[symbol] = cfg
		}
	}
	b.RUnlock()

	isBootstrap := b.latest == nil
	b.latest = r
	b.events <- next

	b.sendAck(ctx, b.enabled, isBootstrap)
	return nil
}

func (b *baseSymbol) sendAck(ctx context.Context, selfEnabled, isBootstrap bool) {
	if !selfEnabled {
		return
	}
	if isBootstrap {
		// 启动时,判断是否ack过 (needAck)
		offset, err := b.kfkCli.GetOffset(b.ResultAckTopic, 0, sarama.OffsetNewest)
		if err != nil {
			log.Println("GetOffset fail", err)
			return
		}
		if offset > 0 {
			var (
				msg *sarama.ConsumerMessage
			)

			pc, err := b.consumer.ConsumePartition(b.ResultAckTopic, 0, offset-1)
			if err != nil {
				log.Printf("get result_ack_topic partition consumer err, %s", err.Error())
				return
			}

			select {
			case m, ok := <-pc.Messages():
				if !ok {
					return
				}
				msg = m
			case <-ctx.Done():
				return
			}

			ack := new(modelsv1.SymbolConfigResultAck)
			err = proto.Unmarshal(msg.Value, ack)
			if err != nil {
				log.Println("Unmarshal fail", err)
				return
			}

			if ack.Version > b.latest.Version {
				log.Printf("ack.Version greater than latest, ack %v, latest %v", ack, b.latest)
				return
			}

			if ack.Version == b.latest.Version {
				for _, sn := range ack.AckService {
					if sn == b.Service {
						// skip ack
						return
					}
				}
			}
		} // end if ack offset > 0
	} // end if Bootstrap

	ack := &modelsv1.SymbolConfigResultAck{
		AckService: b.latest.EnableService,
		Version:    b.latest.Version,
	}
	bs, err := proto.Marshal(ack)
	if err != nil {
		log.Println(err.Error())
		return
	}

	msg := &sarama.ProducerMessage{
		Topic:     b.ResultAckTopic,
		Value:     sarama.ByteEncoder(bs),
		Partition: 0,
	}

	producer, err := sarama.NewSyncProducerFromClient(b.kfkCli)
	if err != nil {
		return
	}

	_, _, err = producer.SendMessage(msg)
	if err != nil {
		log.Println("ack producer send message error", err)
	}
}
