package core

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/container"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer/dispatcher"
	"code.bydev.io/fbu/gateway/gway.git/glog"

	"bgw/pkg/config"
	"bgw/pkg/config_center"
	"bgw/pkg/config_center/nacos"
)

// configManager manage app config resource for app
// cache data and synchronize with remoting
// version control, watch & sync
type configManager struct {
	ctx       context.Context
	configure config_center.Configure

	// cached config entry
	// version.Key() -> *configEntry
	cache container.ConcurrentMap

	// listeners
	dispatcher observer.EventDispatcher
}

// configEntry config & version metadata
type configEntry struct {
	// config source
	config *AppConfig
	// version cached version
	version *ResourceEntry
}

// dirty check local cached data is expired
func (ce *configEntry) dirty(ne *configEntry) (dirty bool) {
	// not exist
	if ce == nil {
		return true
	}

	// same checksum
	if ne == nil || ne.version.Checksum == ce.version.Checksum {
		return false
	}

	// version expired
	return ce.version.LastTime.Before(ne.version.LastTime)
}

// configChangeEvent fired event on config change
// source *AppConfig
type configChangeEvent struct {
	action observer.EventType
	*observer.BaseEvent
}

// newConfigureManager create config manager
func newConfigureManager(ctx context.Context) *configManager {
	return &configManager{
		ctx:        ctx,
		cache:      container.NewConcurrentMap(),
		dispatcher: dispatcher.NewDirectEventDispatcher(ctx),
	}
}

// newVersionChangeEvent for version change event
func newConfigChangeEvent(ac *AppConfig, action observer.EventType) *configChangeEvent {
	return &configChangeEvent{
		BaseEvent: observer.NewBaseEvent(ac),
		action:    action,
	}
}

// init config center client
// namespace is for env isolation
// group is specified
func (cm *configManager) init() error {
	nc, err := nacos.NewNacosConfigure(
		cm.ctx,
		nacos.WithGroup(config.GetGroup()), // specified group
		nacos.WithNameSpace(config.GetNamespace()), // namespace isolation
	)

	if err != nil {
		return err
	}

	cm.configure = nc
	return nil
}

// set cache app.module->config
func (cm *configManager) set(key string, entry *configEntry) error {
	old := cm.get(key)

	// check old data is dirty, ignore update if not
	if !old.dirty(entry) {
		glog.Info(cm.ctx, "[configureManager] old version, ignore update",
			glog.String("key", key),
			glog.String("checksum", entry.version.Checksum))
		return nil
	}

	var et observer.EventType
	if old == nil {
		et = observer.EventTypeAdd // not exist
	} else {
		et = observer.EventTypeUpdate
	}

	// dispatch events on listeners
	ce := newConfigChangeEvent(entry.config, et)
	err := cm.dispatcher.Dispatch(ce)
	if err != nil {
		glog.Error(cm.ctx, "[configureManager]dispatch events error",
			glog.String("key", key),
			glog.String("version", entry.version.LastTime.String()))
		return err
	}

	cm.cache.Set(key, entry)

	glog.Info(cm.ctx, "[configureManager]dispatch events done",
		glog.String("key", key),
		glog.String("version", entry.version.LastTime.String()))

	return nil
}

func (cm *configManager) get(key string) *configEntry {
	value, ok := cm.cache.Get(key)
	if ok {
		return value.(*configEntry)
	}

	return nil
}

func (cm *configManager) remove(key string) error {
	entry := cm.get(key)
	if entry == nil || entry.config == nil {
		return nil
	}

	ce := newConfigChangeEvent(entry.config, observer.EventTypeDel)
	cm.cache.Remove(key)
	err := cm.dispatcher.Dispatch(ce)
	if err != nil {
		return err
	}

	return nil
}

func (cm *configManager) Values() []*AppConfig {
	values := make([]*AppConfig, 0)
	for _, v := range cm.cache.Items() {
		if c, ok := v.(*AppConfig); ok {
			values = append(values, c)
		}
	}

	return values
}

// load load config from nacos by app version info
func (cm *configManager) load(version *AppVersion) (*AppConfig, error) {
	key := version.Key() // map key, app.module

	data, err := cm.configure.Get(cm.ctx, version.GetNacosDataID())

	if err != nil || data == "" {
		return nil, fmt.Errorf("load config data err, %s,%v", version.GetNacosDataID(), err)
	}

	app := &AppConfig{}
	if err := app.Unmarshal(bytes.NewBufferString(data), "yaml"); err != nil {
		return nil, err
	}

	entry := &configEntry{
		config:  app,
		version: version.GetConfigVersionEntry(),
	}

	// update cache
	if err = cm.set(key, entry); err != nil {
		return nil, err
	}

	return app, nil
}

func (cm *configManager) addListener(listener ...observer.EventListener) {
	cm.dispatcher.AddEventListeners(listener)
}

// OnEvent fired on version changed
func (cm *configManager) OnEvent(event observer.Event) error {
	ve, ok := event.(*versionChangeEvent)
	if !ok {
		return nil
	}

	version := ve.GetSource().(*AppVersion)
	glog.Info(cm.ctx, "[configureManager] fire version change event", glog.String("event", version.Key()))

	switch ve.action {
	case observer.EventTypeDel:
		err := cm.remove(version.Key())
		if err != nil {
			glog.Error(cm.ctx, "[configureManager] config Remove error", glog.String("error", err.Error()))
			return nil
		}
	default:
		_, err := cm.load(version)
		if err != nil {
			msg := fmt.Sprintf("config manager load error, err = %s, dataid = %s", err, version.GetNacosDataID())
			galert.Error(cm.ctx, msg)
			return err
		}
	}

	return nil
}

// GetEventType fired on version controller
func (cm *configManager) GetEventType() reflect.Type {
	return reflect.TypeOf(versionChangeEvent{})
}

// GetPriority less priority
func (cm *configManager) GetPriority() int {
	return 10
}
