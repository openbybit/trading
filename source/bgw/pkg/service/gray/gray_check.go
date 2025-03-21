package gray

import (
	"context"
	"sync"

	"bgw/pkg/server/metadata"
)

type grayChecker func(ctx context.Context, vals []any) (bool, error)

var (
	l        sync.RWMutex                   // protect checkers
	checkers = make(map[string]grayChecker) // strategy => check func
)

func init() {
	registerChecker(GrayStrategyFullOn, allOnCheck)
	registerChecker(GrayStrategyFullClose, allCloseCheck)
	registerChecker(GrayStrategyUid, uidGrayCheck)
	registerChecker(GrayStrategyTail, tailGrayCheck)
	registerChecker(GrayStrategyService, serviceGrayCheck)
	registerChecker(GrayStrategyPath, pathGrayCheck)
	registerChecker(GrayStrategyIp, ipGrayCheck)
}

func registerChecker(strategy string, f grayChecker) {
	if f == nil {
		return
	}
	l.Lock()
	checkers[strategy] = f
	l.Unlock()
}

func getChecker(strategy string) (grayChecker, bool) {
	l.RLock()
	defer l.RUnlock()

	res, ok := checkers[strategy]
	return res, ok
}

func allOnCheck(ctx context.Context, vals []any) (bool, error) {
	return true, nil
}

func allCloseCheck(ctx context.Context, vals []any) (bool, error) {
	return false, nil
}

func uidGrayCheck(ctx context.Context, vals []any) (bool, error) {
	uid := getUid(ctx)
	for _, v := range vals {
		id := v.(int)
		if uid == int64(id) {
			return true, nil
		}
	}
	return false, nil
}

func tailGrayCheck(ctx context.Context, vals []any) (bool, error) {
	uid := getUid(ctx)
	for _, v := range vals {
		tail := v.(int)
		if tailMatch(uid, int64(tail)) {
			return true, nil
		}
	}
	return false, nil
}

func serviceGrayCheck(ctx context.Context, vals []any) (bool, error) {
	service := getService(ctx)
	for _, se := range vals {
		if service == se.(string) {
			return true, nil
		}
	}
	return false, nil
}

func pathGrayCheck(ctx context.Context, vals []any) (bool, error) {
	path := getPath(ctx)
	for _, p := range vals {
		if path == p.(string) {
			return true, nil
		}
	}
	return false, nil
}

func ipGrayCheck(ctx context.Context, vals []any) (bool, error) {
	ip := getIP(ctx)
	for _, v := range vals {
		if ip == v.(string) {
			return true, nil
		}
	}

	return false, nil
}

func tailMatch(uid, tail int64) bool {
	if tail > uid {
		return false
	}

	if tail == 0 {
		re := uid % 10
		if re == 0 {
			return true
		}
	}

	for tail > 0 {
		t1 := tail % 10
		t2 := uid % 10
		if t1 != t2 {
			return false
		}
		tail = tail / 10
		uid = uid / 10
	}

	return true
}

func getUid(ctx context.Context) int64 {
	md := metadata.MDFromContext(ctx)
	return md.UID
}

func getService(ctx context.Context) string {
	md := metadata.MDFromContext(ctx)
	return md.Route.Registry
}

func getPath(ctx context.Context) string {
	md := metadata.MDFromContext(ctx)
	return md.Path
}

func getIP(ctx context.Context) string {
	md := metadata.MDFromContext(ctx)
	return md.Extension.RemoteIP
}
