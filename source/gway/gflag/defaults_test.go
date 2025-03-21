package gflag

import (
	"testing"
)

func TestParseAuth(t *testing.T) {
	type authRule struct {
		BizType             int `flag:"bizType"`
		AllowGuest          bool
		RefreshToken        bool
		WeakAuth            bool
		SkipAID             bool
		CopyTrade           bool
		TradeCheck          bool
		Unified             bool
		UnifiedTrading      bool
		AidQuery            []string
		UnifiedTradingCheck string
		UtaProcessBan       bool
		UtaStatus           bool
	}

	rule := authRule{}
	args := []string{"--bizType=1", "--allowGuest=true", "--refreshToken=true"}
	if err := ParseFlags(args, &rule); err != nil {
		t.Errorf("parse flags fail, err:%v", err)
		return
	}

	t.Log(rule)
}
