package core

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/container"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer/dispatcher"
	"code.bydev.io/fbu/gateway/gway.git/gcore/recovery"
	"code.bydev.io/fbu/gateway/gway.git/glog"

	"bgw/pkg/common/constant"
	"bgw/pkg/config"
	"bgw/pkg/config_center"
	"bgw/pkg/config_center/etcd"
)

// versionController watch etcd version data
type versionController struct {
	ctx       context.Context
	configure config_center.Configure

	// env separated
	namespace string
	group     string

	// cached versions entry
	// version.String() -> *AppVersion
	versions container.ConcurrentMap

	// event dispatcher
	// fired on versions changed
	dispatcher observer.EventDispatcher
}

// versionChangeEvent source: *AppVersion
type versionChangeEvent struct {
	action observer.EventType
	*observer.BaseEvent
}

// newVersionChangeEvent for version change event
func newVersionChangeEvent(version *AppVersion, action observer.EventType) *versionChangeEvent {
	return &versionChangeEvent{
		BaseEvent: observer.NewBaseEvent(version),
		action:    action,
	}
}

// newVersionController watch etcd version data
// sync current version on timer(10m)
func newVersionController(ctx context.Context) *versionController {
	return &versionController{
		ctx:        ctx,
		namespace:  config.GetNamespace(),
		group:      config.GetGroup(),
		versions:   container.NewConcurrentMap(),
		dispatcher: dispatcher.NewDirectEventDispatcher(ctx),
	}
}

// init init client and list current version data
func (vc *versionController) init() error {
	ec, err := etcd.NewEtcdConfigure(vc.ctx)
	if err != nil {
		return err
	}

	vc.configure = ec

	// just setup path, no use of value right now
	if err := ec.Put(vc.ctx, vc.root(), time.Now().String()); err != nil {
		return err
	}

	// sync version data on timer
	recovery.Go(vc.loopEvent, recov)

	return nil
}

// root path of real root (with namespace,group)
func (vc *versionController) root() string {
	return filepath.Join(constant.RootPath, vc.namespace, vc.group)
}

// listen on configure
func (vc *versionController) listen() error {
	if err := vc.load(vc.root()); err != nil {
		glog.Info(vc.ctx, "version controller listen load error", glog.String("err", err.Error()))
		return err
	}
	return vc.configure.Listen(vc.ctx, vc.root(), vc)
}

// load current version data
// run on init process
func (vc *versionController) load(prefix string) error {
	kl, vl, err := vc.configure.GetChildren(vc.ctx, prefix)
	if err != nil {
		msg := fmt.Sprintf("version controller load error, err = %s, prefix = %s", err.Error(), prefix)
		galert.Error(vc.ctx, msg)
		return err
	}

	versions := make([]*AppVersion, 0, len(kl))
	for i, k := range kl {
		version := vc.decodeMetadata(k, vl[i])
		if version == nil {
			continue
		}

		if !vc.isLocalGroup(version) {
			continue
		}

		versions = append(versions, version)
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version.Resources[0].LastTime.Before(versions[j].Version.Resources[0].LastTime)
	})

	for _, version := range versions {
		// fixme 这里可能会出现漏掉的情况？
		if err = vc.set(version); err != nil {
			glog.Error(vc.ctx, "[versionController]on event result failed", glog.String("version", version.String()), glog.String("error", err.Error()))
		}
	}

	return nil
}

// decodeMetadata parse metedata
func (vc *versionController) decodeMetadata(path, content string) *AppVersion {
	if base := filepath.Base(path); base != constant.EtcdDeployKey {
		return nil
	}

	version := &AppVersion{}
	if err := version.Decode(content); err != nil {
		glog.Error(vc.ctx, "[versionController]decode version metadata error", glog.String("error", err.Error()), glog.String("content", content))
		return nil
	}

	return version
}

// addListener add listener for version change event
func (vc *versionController) addListener(listener ...observer.EventListener) {
	vc.dispatcher.AddEventListeners(listener)
}

// set will set app version if any update(LastTime is newer)
func (vc *versionController) set(version *AppVersion) error {
	key := version.String()
	old, ok := vc.versions.Get(key)

	// no change ignore
	if ok && !version.Version.LastTime.After(old.(*AppVersion).Version.LastTime) {
		glog.Info(vc.ctx, "version controller ignore", glog.String("version", version.String()))
		return nil
	}

	// fire event of version change
	event := newVersionChangeEvent(version, observer.EventTypeUpdate)
	// dispatch error
	if err := vc.dispatcher.Dispatch(event); err != nil {
		msg := fmt.Sprintf("version controller update version dispatch error, err = %s, version = %s", err.Error(), key)
		galert.Error(vc.ctx, msg)
		return err
	}

	vc.versions.Set(key, version)
	glog.Info(vc.ctx, "[versionController]update version", glog.String("version", version.String()))

	return nil
}

// get a version data if exists otherwise nil
// nolint
func (vc *versionController) get(key string) *AppVersion {
	val, ok := vc.versions.Get(key)
	if !ok {
		return nil
	}

	return val.(*AppVersion)
}

// remove do remove version and fire event
func (vc *versionController) remove(version *AppVersion) error {
	vc.versions.Remove(version.String())
	// fire event of version delete
	event := newVersionChangeEvent(version, observer.EventTypeDel)
	if err := vc.dispatcher.Dispatch(event); err != nil {
		glog.Error(vc.ctx, "[versionController]remove version error", glog.String("version", version.String()), glog.String("err", err.Error()))
		return err
	}

	return nil
}

// Keys return all version keys
func (vc *versionController) Keys() []string {
	return vc.versions.Keys()
}

// Values all versions info
func (vc *versionController) Values() []*AppVersion {
	vals := make([]*AppVersion, 0, vc.versions.Count())

	for _, v := range vc.versions.Items() {
		vals = append(vals, v.(*AppVersion))
	}

	return vals
}

// OnEvent fired on version changed
func (vc *versionController) OnEvent(event observer.Event) error {
	re, ok := event.(*observer.DefaultEvent)
	if !ok {
		return nil
	}

	version := vc.decodeMetadata(re.Key, re.Value)
	if version == nil {
		return nil
	}

	if !vc.isLocalGroup(version) {
		return nil
	}
	glog.Info(vc.ctx, "[versionController] on event", glog.String("key", re.Key), glog.String("version", version.String()))

	var (
		err error
	)
	switch re.Action {
	case observer.EventTypeAdd:
		err = vc.set(version)
	case observer.EventTypeUpdate:
		err = vc.set(version)
	case observer.EventTypeDel:
		err = vc.remove(version)
	}

	if err != nil {
		glog.Info(vc.ctx, "[versionController]on event result failed", glog.String("error", err.Error()), glog.Int64("action", int64(re.Action)))
		return err
	}

	return nil
}

// GetEventType remoting etcd watch event
func (vc *versionController) GetEventType() reflect.Type {
	return reflect.TypeOf(observer.DefaultEvent{})
}

// GetPriority get priority, implement event.Priority
func (vc *versionController) GetPriority() int {
	return -1
}

// loopEvent loop on timer, sync current version
func (vc *versionController) loopEvent() {
	timer := time.NewTicker(10 * time.Minute)

	for {
		select {
		case <-vc.ctx.Done():
			glog.Warn(vc.ctx, "context done", glog.String("err", vc.ctx.Err().Error()))
			return
		case <-timer.C:
			if err := vc.load(vc.root()); err != nil {
				glog.Error(vc.ctx, "[versionController]load version error", glog.String("error", err.Error()))
			} else {
				glog.Info(vc.ctx, "[versionController]sync version success")
			}
		}
	}
}

func (vc *versionController) isLocalGroup(version *AppVersion) bool {
	// ignore other group version event
	if version.Group == "" {
		version.Group = constant.BGW_GROUP
	}
	if version.Namespace != vc.namespace {
		glog.Info(vc.ctx, "[versionController] check event, ignore version namespace", glog.String("version", version.String()), glog.String("namespace", version.Namespace))
		return false
	}
	if version.Group != vc.group {
		glog.Info(vc.ctx, "[versionController] check event, ignore version group", glog.String("version", version.String()), glog.String("group", version.Group))
		return false
	}
	glog.Info(vc.ctx, "[versionController] check event, apply version", glog.String("version", version.String()), glog.String("namespace", version.Namespace), glog.String("group", version.Group))
	return true
}
