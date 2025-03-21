package core

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
)

const (
	historyMaxLength = 5
)

type ResourceType string

const (
	ResourceConfig ResourceType = "config"
	ResourceDesc   ResourceType = "descriptor"
)

type AppVersion struct {
	lock sync.RWMutex

	Namespace string            `json:"namespace"`
	Group     string            `json:"group"`
	App       string            `json:"app"`
	Module    string            `json:"module"`
	Version   AppVersionEntry   `json:"version"`
	History   []AppVersionEntry `json:"history"`
}

type AppVersionEntry struct {
	LastTime  time.Time        `json:"lastTime"`
	Resources [2]ResourceEntry `json:"resources"`
}

type ResourceEntry struct {
	ResourceType ResourceType `json:"resourceType"`
	LastTime     time.Time    `json:"lastTime"`
	Checksum     string       `json:"checksum"`
}

func NewAppVersion() *AppVersion {
	return &AppVersion{
		Namespace: config.GetNamespace(),
		Group:     config.GetGroup(),
		History:   make([]AppVersionEntry, 0, historyMaxLength),
		Version: AppVersionEntry{
			Resources: [2]ResourceEntry{
				{
					ResourceType: ResourceConfig,
				},
				{
					ResourceType: ResourceDesc,
				},
			},
		},
	}
}

func (a *AppVersion) SetCurrentResource(typ ResourceType, checksum string, lastTime time.Time) {
	switch typ {
	case ResourceConfig:
		a.Version.Resources[0].Checksum = checksum
		a.Version.Resources[0].LastTime = lastTime
	case ResourceDesc:
		a.Version.Resources[1].Checksum = checksum
		a.Version.Resources[1].LastTime = lastTime
	}
}

// String app.module.version
func (a *AppVersion) String() string {
	return fmt.Sprintf("%s.%s.%s", a.App, a.Module, a.Compact())
}

// Key app.module, omit version spec
// used as resource key
func (a *AppVersion) Key() string {
	return fmt.Sprintf("%s.%s", a.App, a.Module)
}

func (a *AppVersion) GetConfig() ResourceEntry {
	return a.Version.Resources[0]
}

func (a *AppVersion) GetDesc() ResourceEntry {
	return a.Version.Resources[1]
}

// Path root.app.module
// full etcd path
func (a *AppVersion) Path() string {
	return filepath.Join(constant.RootPath, a.App, a.Module)
}

// GetS3Key for aws.s3
// namespace isolation
func (a *AppVersion) GetS3Key() string {
	desc := a.GetDescVersion()
	if desc == "" {
		return ""
	}
	return filepath.Join(constant.RootPath, a.Namespace, a.App, a.Module, desc)
}

// GetEtcdKey for etcd
func (a *AppVersion) GetEtcdKey() string {
	return filepath.Join(constant.RootPath, a.Namespace, a.App, a.Module, constant.EtcdDeployKey)
}

// GetNacosDataID for nacos dataid
func (a *AppVersion) GetNacosDataID() string {
	return fmt.Sprintf("%s.%s.%s", a.App, a.Module, a.GetConfigVersion())
}

func (a *AppVersion) AddHistory(version *AppVersionEntry) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.History == nil {
		a.History = make([]AppVersionEntry, 0, historyMaxLength)
	}

	if len(a.History) == historyMaxLength {
		a.History = append([]AppVersionEntry{*version}, a.History[:historyMaxLength-1]...)
	} else {
		a.History = append([]AppVersionEntry{*version}, a.History[:]...)
	}
}

// Compact format version as compact datetime
func (a *AppVersion) Compact() string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.Version.LastTime.Format("20060102150405")
}

func (a *AppVersion) GetCurrentVersion() *AppVersionEntry {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return &a.Version
}

func (a *AppVersion) GetConfigVersion() string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.Version.Resources[0].LastTime.Format("20060102150405")
}

func (a *AppVersion) GetConfigChecksum() string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.Version.Resources[0].Checksum
}

func (a *AppVersion) GetDescVersion() string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	var t time.Time
	if a.Version.Resources[1].Checksum == "" || a.Version.Resources[1].LastTime == t {
		return ""
	}
	return a.Version.Resources[1].LastTime.Format("20060102150405")
}

func (a *AppVersion) GetDescChecksum() string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.Version.Resources[1].Checksum
}

func (a *AppVersion) GetConfigVersionEntry() *ResourceEntry {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return &a.Version.Resources[0]
}

func (a *AppVersion) GetDescVersionEntry() *ResourceEntry {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return &a.Version.Resources[1]
}

// func (a *AppVersion) ToURL() (*common.URL, error) {
// 	r := config.GetRemoteConfig(constant.EtcdConfigKey)
// 	return common.NewURL(r.Address,
// 		common.WithProtocol(constant.EtcdProtocol),
// 		common.WithPath(a.Path()),
// 		common.WithGroup(config.GetGroup()),
// 		common.WithVersion(a.Compact()),
// 	)
// }

func (a *AppVersion) Decode(data string) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	return util.JsonUnmarshal([]byte(data), a)
}

func (a *AppVersion) Encode() string {
	a.lock.Lock()
	defer a.lock.Unlock()

	data, err := util.JsonMarshal(a)
	if err != nil {
		return ""
	}

	return string(data)
}
