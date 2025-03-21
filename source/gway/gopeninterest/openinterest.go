package gopeninterest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"code.bydev.io/fbu/future/sdk.git/pkg/future"
	"code.bydev.io/frameworks/sarama"
	"go.uber.org/atomic"

	gfuture "code.bydev.io/fbu/gateway/gway.git/gsymbol/future"
)

type Limiter interface {
	Limit(uid int64, symbol int32, side int32) bool
	CheckUserOpenInterestExceeded(uid int64, symbol int32) (buyOI, sellOI bool)
}

type limiter struct {
	ctx           context.Context
	cfg           *Config
	kInverse      sarama.Client
	kLinearUsdt   sarama.Client
	kLinearUsdc   sarama.Client
	scmeta        *gfuture.Scmeta
	ready         chan struct{}
	unreadyTopics *atomic.Int32
	once          sync.Once
	status        *atomic.Bool

	configs     map[future.Symbol]*oiExceededResultDTO
	configsLock sync.RWMutex
}

type oiExceededResultDTO struct {
	Symbol                   future.Symbol           `json:"symbol"`
	ExceededResultVer        int64                   `json:"exceeded_result_ver,omitempty"`
	BuyExceededResultMap     map[future.UserID]int64 `json:"buy_exceeded_result_map,omitempty"`
	SellExceededResultMap    map[future.UserID]int64 `json:"sell_exceeded_result_map,omitempty"`
	ExtraBuyExceedResultMap  map[future.UserID]int64 `json:"extra_buy_exceeded_result_map,omitempty"`
	ExtraSellExceedResultMap map[future.UserID]int64 `json:"extra_sell_exceeded_result_map,omitempty"`
}

func New(ctx context.Context, sc *gfuture.Scmeta, cfg *Config) (Limiter, error) {
	k123Cli, err := sarama.NewClient(cfg.K123Brokers, newConfig())
	if err != nil {
		return nil, err
	}

	kabcCli, err := sarama.NewClient(cfg.KabcBrokers, newConfig())
	if err != nil {
		return nil, err
	}

	kusdcCli, err := sarama.NewClient(cfg.KusdcBrokers, newConfig())
	if err != nil {
		return nil, err
	}

	l := &limiter{
		ctx:           ctx,
		kInverse:      k123Cli,
		kLinearUsdt:   kabcCli,
		kLinearUsdc:   kusdcCli,
		cfg:           cfg,
		scmeta:        sc,
		status:        atomic.NewBool(false),
		ready:         make(chan struct{}),
		unreadyTopics: atomic.NewInt32(0),
	}
	l.configs = make(map[future.Symbol]*oiExceededResultDTO)
	l.init()

	select {
	case <-l.ready:
	case <-ctx.Done():
		return nil, errors.New("ctx has been down")
	case <-time.After(5 * time.Second):
		if l.unreadyTopics.Load() > 0 {
			return nil, errors.New("wait open limit init timeout")
		}
	}

	return l, nil
}

func (l *limiter) init() {
	allCoins := l.scmeta.GetOnlineTradingCoins()
	log.Printf("[gway]oi init all coins, %v", allCoins)
	if l.cfg.EnableLinearUSDCCoin {
		allCoins[future.Coin(16)] = "USDC"
	}
	var isInverse bool
	var err error
	var needBind bool
	for coin := range allCoins {
		needBind = false
		if isInverse, err = l.scmeta.IsInverse(coin); err != nil {
			// 认为USDT和USDC是正向，其他是反向
			isInverse = coin != 5 && coin != 16 // USDT, USDC
		}
		if isInverse && l.cfg.EnableInverseCoin {
			needBind = true
		} else if l.cfg.EnableLinearUSDCCoin && coin == 16 {
			// USDC
			needBind = true
		} else if l.cfg.EnableLinearUSDTCoin && coin == 5 {
			needBind = true
		}
		if needBind {
			// topic计数
			l.unreadyTopics.Add(1)
			go l.foreverBind(l.ctx, coin)
		}
	}
}

func (l *limiter) foreverBind(ctx context.Context, coin future.Coin) {
	topicName := fmt.Sprintf(l.cfg.TopicNameTpl, l.scmeta.CoinName(coin))
	var isInverse bool
	var err error
	if isInverse, err = l.scmeta.IsInverse(coin); err != nil {
		// 认为USDT和USDC是正向，其他是反向
		isInverse = coin != 5 && coin != 16 // USDT, USDC
	}
	var kfkCli sarama.Client
	if isInverse {
		kfkCli = l.kInverse
	} else if coin == 16 {
		// USDC
		kfkCli = l.kLinearUsdc
	} else {
		kfkCli = l.kLinearUsdt
	}

	offset, err := kfkCli.GetOffset(topicName, 0, sarama.OffsetNewest)
	if err != nil {
		log.Printf("get offset err, %s", err.Error())
		return
	}
	log.Println(fmt.Sprintf("oi topicName:%s;offset:%d", topicName, offset))
	if offset > 0 {
		offset -= 1
	} else {
		// 减去无数据的topic
		l.unreadyTopics.Sub(1)
	}

	cumer, err := sarama.NewConsumerFromClient(kfkCli)
	if err != nil {
		log.Printf("get consumer err, %s", err.Error())
		return
	}

	for {
		offset, err = l.consume(ctx, cumer, topicName, offset)
		if err != nil {
			log.Println(err.Error())
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (l *limiter) consume(ctx context.Context, consumer sarama.Consumer, topic string, offset int64) (int64, error) {
	pc, err := consumer.ConsumePartition(topic, 0, offset)
	if err != nil {
		return offset, err
	}

	for {
		select {
		case msg, ok := <-pc.Messages():
			if !ok {
				return offset, nil
			}
			err = l.handle(ctx, msg)
			if err != nil {
				return offset, err
			}
			offset = msg.Offset + 1
			if !l.status.Load() {
				ut := l.unreadyTopics.Sub(1)
				if ut <= 0 {
					l.status.Store(true)
					l.once.Do(func() {
						close(l.ready)
					})
				}
			}
		case <-ctx.Done():
			return offset, ctx.Err()
		}
	}
}

func (l *limiter) handle(ctx context.Context, msg *sarama.ConsumerMessage) error {
	r := make(map[future.Symbol]*oiExceededResultDTO)
	if err := json.Unmarshal(msg.Value, &r); err != nil {
		return errors.New(err.Error() + fmt.Sprintf("open limit, handleSync,value:%s", msg.Value))
	}

	if l.cfg.EnableLogResult {
		log.Printf("open limit, receiving new version, offset:%d, topic:%s, timestamp:%d",
			msg.Offset, msg.Topic, msg.Timestamp.Unix())
		bs, err := json.Marshal(r)
		if err != nil {
			log.Printf("open limit unmarshal error, %s, offset:%d, topic:%s, timestamp:%d",
				err.Error(), msg.Offset, msg.Topic, msg.Timestamp.Unix())
		} else {
			// this log nearly 1~4 MB
			log.Printf("open limit result, offset:%d, topic:%s, timestamp:%d, data %s",
				msg.Offset, msg.Topic, msg.Timestamp.Unix(), string(bs))
		}
	}

	l.configsLock.Lock()
	defer l.configsLock.Unlock()
	for _, dto := range r {
		l.configs[dto.Symbol] = mergeExtraExceedResult(dto)
	}
	return nil
}

func mergeExtraExceedResult(dto *oiExceededResultDTO) *oiExceededResultDTO {
	if dto == nil {
		return nil
	}

	buyExceededResultMap := dto.BuyExceededResultMap
	if buyExceededResultMap == nil {
		buyExceededResultMap = make(map[future.UserID]int64)
	}

	sellExceededResultMap := dto.SellExceededResultMap
	if sellExceededResultMap == nil {
		sellExceededResultMap = make(map[future.UserID]int64)
	}

	extraBuyExceedResultMap := dto.ExtraBuyExceedResultMap
	extraSellExceedResultMap := dto.ExtraSellExceedResultMap

	// 合并 extraBuyExceedResultMap 到 buyExceededResultMap
	for key, value := range extraBuyExceedResultMap {
		buyExceededResultMap[key] = value
	}

	// 合并 extraSellExceedResultMap 到 sellExceededResultMap
	for key, value := range extraSellExceedResultMap {
		sellExceededResultMap[key] = value
	}
	return dto
}
