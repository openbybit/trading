package gcompliance

import (
	"fmt"
	"strconv"

	"code.bydev.io/cht/customer/kyc-stub.git/pkg/bybit/common/enums"
	compliance "code.bydev.io/cht/customer/kyc-stub.git/pkg/bybit/compliancewall/strategy/v1"
	jsoniter "github.com/json-iterator/go"

	"code.bydev.io/fbu/gateway/gway.git/gmetric"
)

type WhitelistMQMsg struct {
	RequestID       string   `json:"request_id"`
	WhitelistValues []string `json:"whitelist_values"`
}

func (w *wall) HandleUserWhiteListEvent(data []byte) error {
	if !w.withCache {
		return nil
	}
	msg := &WhitelistMQMsg{}
	err := jsoniter.Unmarshal(data, msg)
	if err != nil {
		return err
	}

	for _, v := range msg.WhitelistValues {
		uid, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			continue
		}
		w.userInfos.Remove(uid)
		gmetric.IncDefaultCounter("compliance", "user_whitelist_event")
	}
	return nil
}

type ComplianceWallKycMQMsg struct {
	RequestID string  `json:"request_id"`
	MemberIDs []int64 `json:"member_ids"`
}

func (w *wall) HandleUserKycEvent(data []byte) error {
	if !w.withCache {
		return nil
	}
	rn := &ComplianceWallKycMQMsg{}
	err := jsoniter.Unmarshal(data, rn)
	if err != nil {
		return err
	}

	for _, id := range rn.MemberIDs {
		w.userInfos.Remove(id)
	}
	return nil
}

func (w *wall) HandleStrategyEvent(data []byte) error {
	if !w.withCache {
		return nil
	}
	cfg := &compliance.ComplianceConfigTopicMessage{}
	err := jsoniter.Unmarshal(data, cfg)
	if err != nil {
		return err
	}
	w.cs.Update(convert(cfg.SceneItems...))
	return nil
}

type event struct {
	BusType enums.ComplianceMQEventType `json:"bus_type"`
	BusData *compliance.SitesConfigOut  `json:"bus_data"`
}

func (w *wall) HandleSiteConfigEvent(data []byte) error {
	if !w.withCache {
		return nil
	}
	e := &event{}
	err := jsoniter.Unmarshal(data, e)
	if err != nil {
		return err
	}

	if e.BusType != enums.ComplianceMQEventType_SITE_CONFIG || e.BusData == nil {
		return nil
	}

	jsonRes := make(map[string]string)
	res := make(map[string]*compliance.SitesConfigItemConfig)
	for _, cfg := range e.BusData.Item {
		if cfg == nil {
			continue
		}
		key := fmt.Sprintf("%s.%s", cfg.GetSite(), cfg.GetProduct())
		c := cfg.GetConfig()
		if c == nil {
			continue
		}
		val, err := jsoniter.Marshal(c)
		if err != nil {
			continue
		}
		jsonRes[key] = string(val)
		res[key] = c
	}

	w.siteMutex.Lock()
	defer w.siteMutex.Unlock()

	for k, v := range jsonRes {
		w.jsonCfg[k] = v
	}

	for k, v := range res {
		w.siteCfg[k] = v
	}
	return nil
}
