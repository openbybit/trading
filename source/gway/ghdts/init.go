package ghdts

import (
	"sync"

	"git.bybit.com/tni_go/dtssdk_go_interface/dtssdk"
)

var initOnce sync.Once

func ensureInitSDK() {
	initOnce.Do(func() {
		if err := dtssdk.InitDtsSdk(nil, false); err != nil {
			panic(err)
		}
	})
}

// Destroy 消耗dts sdk,内部会保证幂等调用
func Destroy() {
	dtssdk.DestroyDtsSdk()
}
