package gsymbol

import "encoding/json"

var globalMgr = newManager()

func Start(conf *Config) error {
	return globalMgr.Start(conf)
}

func Stop() {
	globalMgr.Stop()
}

func GetFutureManager() *FutureManager {
	return globalMgr.GetFutureManager()
}

func GetOptionManager() *OptionManager {
	return globalMgr.GetOptionManager()
}

func GetSpotManager() *SpotManager {
	return globalMgr.GetSpotManager()
}

// future utility functions
func GetFutureConfigList(opts ...Option) []*FutureConfig {
	return globalMgr.GetFutureManager().GetList(opts...)
}

func GetFutureConfigByID(id int, opts ...Option) *FutureConfig {
	return globalMgr.GetFutureManager().GetByID(id, opts...)
}

func GetFutureConfigByName(name string, opts ...Option) *FutureConfig {
	return globalMgr.GetFutureManager().GetByName(name, opts...)
}

func GetFutureConfigByAlias(name string, opts ...Option) *FutureConfig {
	return globalMgr.GetFutureManager().GetByAlias(name, opts...)
}

func GetFutureConfigByCoin(coin int, opts ...Option) []*FutureConfig {
	return globalMgr.GetFutureManager().GetByCoin(coin, opts...)
}

// option utility functions
func GetOptionConfigList() []*OptionConfig {
	return globalMgr.GetOptionManager().GetList()
}

func GetOptionConfigByID(id int) *OptionConfig {
	return globalMgr.GetOptionManager().GetByID(id)
}

func GetOptionConfigByName(name string) *OptionConfig {
	return globalMgr.GetOptionManager().GetByName(name)
}

// spot utility functions
func GetSpotConfigList() []*SpotConfig {
	return globalMgr.GetSpotManager().GetList()
}

func GetSpotConfigByID(id int) *SpotConfig {
	return globalMgr.GetSpotManager().GetByID(id)
}

func GetSpotConfigByName(name string) *SpotConfig {
	return globalMgr.GetSpotManager().GetByName(name)
}

func SetMockFutureData(obj interface{}) {
	var data string
	switch x := obj.(type) {
	case string:
		data = x
	case []*FutureConfig:
		c := futureAllConfig{
			Data:    x,
			Version: "",
		}
		r, _ := json.Marshal(c)
		data = string(r)
	case *FutureConfig:
		c := futureAllConfig{
			Data:    []*FutureConfig{x},
			Version: "",
		}
		r, _ := json.Marshal(c)
		data = string(r)
	default:
		panic("invalid mock future data")
	}
	_ = GetFutureManager().build(data)
}

func SetMockOptionData(obj interface{}) {
	var data string
	switch x := obj.(type) {
	case string:
		data = x
	case []*OptionConfig:
		c := optionAllConfig{
			Data:    x,
			Version: "",
		}
		r, _ := json.Marshal(c)
		data = string(r)
	case *OptionConfig:
		c := optionAllConfig{
			Data:    []*OptionConfig{x},
			Version: "",
		}
		r, _ := json.Marshal(c)
		data = string(r)
	default:
		panic("invalid mock option data")
	}
	_ = GetOptionManager().build(data)
}

func SetMockSpotData(obj interface{}) {
	var data string
	switch x := obj.(type) {
	case string:
		data = x
	case []*SpotConfig:
		c := spotAllConfig{
			Data:    x,
			Version: "",
		}
		r, _ := json.Marshal(c)
		data = string(r)
	case *SpotConfig:
		c := spotAllConfig{
			Data:    []*SpotConfig{x},
			Version: "",
		}
		r, _ := json.Marshal(c)
		data = string(r)
	default:
		panic("invalid mock spot data")
	}
	_ = GetSpotManager().build(data)
}
