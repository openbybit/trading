package ws

import (
	"context"
	"sync/atomic"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

const (
	defaultSessionTickTime = time.Second * 30 // 默认session tick间隔
)

var gTickerMgr = tickerMgr{}

type tickerMgr struct {
	userTicker           *time.Ticker // 定时同步用户状态ticker
	sessionTicker        *time.Ticker // 定时触发session校验
	quit                 int32        // 退出标识
	lastSyncUsersTime    int64        // 最后一次同步全量用户时间
	lastSyncAcceptorTime int64        // 最后一次同步某些异常Acceptor时间
	lastSyncRegularTime  int64        // 最后一次常规定时同步时间
	acceptorRoundIndex   int          // 轮询acceptor索引
}

func (tm *tickerMgr) Start() {
	appConf := getAppConf()
	if !appConf.DisableUserTick && tm.userTicker == nil {
		tm.userTicker = time.NewTicker(time.Minute)
		go tm.loopUserTick()
	}

	if !appConf.DisableSessionTick && tm.sessionTicker == nil {
		tm.sessionTicker = time.NewTicker(defaultSessionTickTime)
		go tm.loopSessionsTick()
	}
}

func (tm *tickerMgr) Stop() {
	if atomic.CompareAndSwapInt32(&tm.quit, 0, 1) {
		if tm.userTicker != nil {
			tm.userTicker.Stop()
			tm.userTicker = nil
		}

		if tm.sessionTicker != nil {
			tm.sessionTicker.Stop()
			tm.sessionTicker = nil
		}
	}
}

// loopUserTick 定期同步用户数据到sdk
func (tm *tickerMgr) loopUserTick() {
	glog.Info(context.Background(), "loopUserTick start")
	for range tm.userTicker.C {
		if atomic.LoadInt32(&tm.quit) == 1 {
			glog.Info(context.Background(), "loopUserTick stop")
			return
		}

		tm.updateUsages()
		tm.checkSyncUsers()
		tm.checkSyncAcceptors()
		tm.checkSyncRegular()
	}
}

// updateUsages 更新资源使用率
func (tm *tickerMgr) updateUsages() {
	WSGauge(globalExchange.EventChannelRate(), "exchange", "events_usage")
	// acceptors := GetAcceptorMgr().GetAll()
	// for _, acc := range acceptors {
	// 	wsUsageObserve(acc.SendChannelRate(), "acceptor", acc.AppID())
	// }
}

// checkSyncUsers 检测是否需要同步全量用户,如果同步了用户则不再需要同步acceptor
func (tm *tickerMgr) checkSyncUsers() {
	ts := globalExchange.LastWriteChannelFailTime()
	if ts == 0 || ts < tm.lastSyncUsersTime {
		return
	}

	// 两次最小同步时间间隔
	defaultMinSyncInterval := int64(time.Minute * 10)

	now := nowUnixNano()
	if now-tm.lastSyncUsersTime < defaultMinSyncInterval {
		return
	}

	glog.Info(context.Background(), "force sync users to all acceptors")
	WSCounterInc("ticker", "sync_users")
	// 需要全量同步一次用户数据
	tm.lastSyncUsersTime = now
	tm.lastSyncAcceptorTime = now

	acceptors := GetAcceptorMgr().GetAll()
	users := GetUserMgr().GetAllUsers()
	if len(acceptors) == 0 || len(users) == 0 {
		return
	}

	for _, acc := range acceptors {
		DispatchEvent(NewSyncAllUserEventWithUsers(acc.ID(), users))
	}
}

// checkSyncAcceptors 检测是否需要同步某些出错的acceptor
func (tm *tickerMgr) checkSyncAcceptors() {
	ts := globalExchange.LastWriteAcceptorFailTime()
	if ts == 0 || ts < tm.lastSyncAcceptorTime {
		return
	}

	WSCounterInc("ticker", "sync_acceptors")
	oldTs := tm.lastSyncAcceptorTime
	now := nowUnixNano()
	tm.lastSyncAcceptorTime = now

	acceptors := GetAcceptorMgr().GetAll()
	users := GetUserMgr().GetAllUsers()
	if len(acceptors) == 0 || len(users) == 0 {
		return
	}

	for _, acc := range acceptors {
		lastTs := acc.LastWriteFailTime()
		if lastTs == 0 || lastTs < oldTs {
			continue
		}

		glog.Info(context.Background(), "force sync users to acceptor", glog.String("acceptor_id", acc.ID()), glog.Int("size", len(users)))
		DispatchEvent(NewSyncAllUserEventWithUsers(acc.ID(), users))
	}
}

// checkSyncRegular 判断是否需要常规定时同步
func (tm *tickerMgr) checkSyncRegular() {
	dconf := getDynamicConf()
	if !dconf.SyncUserRegularEnable {
		return
	}

	interval := dconf.SyncUserRegularInterval
	if interval == 0 {
		interval = time.Minute * 15
	}

	now := nowUnixNano()
	if now-tm.lastSyncRegularTime < int64(interval) {
		return
	}

	acceptor := GetAcceptorMgr().GetByIndex(tm.acceptorRoundIndex)
	if !isNil(acceptor) {
		DispatchEvent(NewSyncAllUserEvent(acceptor.ID()))
		tm.acceptorRoundIndex++
		if tm.acceptorRoundIndex >= GetAcceptorMgr().Size() {
			tm.acceptorRoundIndex = 0
		}
	}
}

// loopSessionsTick 定时判断session连接是否正常
func (tm *tickerMgr) loopSessionsTick() {
	glog.Info(context.Background(), "loopSessionsTick start")
	for range tm.sessionTicker.C {
		if atomic.LoadInt32(&tm.quit) == 1 {
			glog.Info(context.Background(), "loopSessionsTick stop")
			return
		}

		sessions := GetSessionMgr().GetSessions()
		for _, sess := range sessions {
			sess.OnTick()
		}
	}
}
