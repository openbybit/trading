package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/encoding"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/getcd"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	clientv3 "go.etcd.io/etcd/client/v3"

	"bgw/pkg/common/bhttp"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
	"bgw/pkg/config_center"
	"bgw/pkg/config_center/nacos"
	retcd "bgw/pkg/remoting/etcd"
	"bgw/pkg/server/core"
)

const (
	BGWHttpListenFiles = "bgw_listen_http_files"

	WebConsoleAlertTitle     = "http配置更新"
	WebConsoleAlertErrFormat = "web console err = %s, evenKey = %s"
)

type webConsole struct {
	ctx        context.Context
	etcdClient getcd.Client
	nacosCfg   config_center.Configure
	files      sync.Map
}

func newWebConsole(ctx context.Context) *webConsole {
	return &webConsole{ctx: ctx}
}

func (w *webConsole) init() error {
	rc := config.Global.WebConsole

	files := rc.GetArrayOptions("config_files", nil)
	if len(files) == 0 {
		glog.Info(w.ctx, "webConsole not file to listening in static config")
	}

	ec, err := retcd.NewConfigClient(w.ctx)
	if err != nil {
		return err
	}
	w.etcdClient = ec

	nc, err := nacos.NewNacosConfigure(
		w.ctx,
		nacos.WithGroup(config.GetGroup()), // specified group
		nacos.WithNameSpace(config.GetNamespace()), // namespace isolation
	)
	if err != nil {
		return err
	}
	w.nacosCfg = nc

	for _, file := range files {
		glog.Info(w.ctx, "webConsole listen start", glog.String("file", file))
		if err := w.nacosCfg.Listen(w.ctx, file, w); err != nil {
			galert.Error(context.TODO(), fmt.Sprintf("http config listen error, err = %s, file = %s", err.Error(), file))
			continue
		}
		w.files.Store(file, struct{}{})
	}

	if err := nc.Listen(context.Background(), BGWHttpListenFiles, w); err != nil {
		glog.Error(context.Background(), "listen bgwg listen files failed", glog.String("err", err.Error()))
		return err
	}

	return nil
}

type listenFiles struct {
	ListenFiles []string `yaml:"listen_http_files"`
}

func (w *webConsole) OnEvent(event observer.Event) error {
	e, ok := event.(*observer.DefaultEvent)
	if !ok {
		return nil
	}
	glog.Info(w.ctx, "webConsole receive update event", glog.String("file", e.Key))
	if e.Value == "" {
		return nil
	}

	if e.Key == BGWHttpListenFiles {
		val := &listenFiles{}
		if err := util.YamlUnmarshalString(e.Value, val); err != nil {
			galert.Error(context.TODO(), fmt.Sprintf("http listen config YamlUnmarshal error, err = %s, file = %s", err.Error(), e.Key))
			return err
		}
		for _, file := range val.ListenFiles {
			_, loaded := w.files.LoadOrStore(file, struct{}{})
			if !loaded {
				glog.Info(w.ctx, "webConsole listen start", glog.String("file", file))
				if err := w.nacosCfg.Listen(w.ctx, file, w); err != nil {
					galert.Error(context.TODO(), fmt.Sprintf("http config listen error, err = %s, file = %s", err.Error(), file))
					continue
				}
			}
		}
		return nil
	}

	ac, err := w.staticPreCheck(e.Key, e.Value)
	if err != nil {
		msg := fmt.Sprintf(WebConsoleAlertErrFormat, err.Error(), e.Key)
		galert.Error(w.ctx, msg, galert.WithTitle(WebConsoleAlertTitle))
		return nil
	}

	if err = w.buildVersion(ac); err != nil {
		msg := fmt.Sprintf(WebConsoleAlertErrFormat, err.Error(), e.Key)
		galert.Error(w.ctx, msg, galert.WithTitle(WebConsoleAlertTitle))
		return nil
	}

	return nil
}

// buildVersion upload config to nacos, build AppVersion and set etcd
func (w *webConsole) buildVersion(ac *core.AppConfig) error {
	data, err := util.YamlMarshal(ac)
	if err != nil {
		glog.Error(w.ctx, "yaml.Marshal error", glog.String("key", ac.Key()), glog.String("error", err.Error()))
		return err
	}

	etcdKey := filepath.Join(constant.RootPath, config.GetNamespace(), config.GetGroup(), ac.App, ac.Module, constant.EtcdDeployKey)
	ver, err := w.etcdClient.Get(etcdKey)
	if err != nil {
		if !errors.Is(err, getcd.ErrKVPairNotFound) {
			glog.Error(w.ctx, "get etcd version error", glog.String("key", etcdKey), glog.String("error", err.Error()))
			return err
		}
		glog.Info(w.ctx, "webConsole will build first version", glog.String("key", ac.Key()))
	}

	configMd5 := encoding.MD5Hex(data)
	ver = strings.TrimSpace(ver)
	if ver != "" {
		oldVer := &core.AppVersion{}
		if err = util.JsonUnmarshalString(ver, oldVer); err != nil {
			glog.Error(w.ctx, "get etcd version Unmarshal error", glog.String("key", etcdKey), glog.String("error", err.Error()))
			return err
		}
		if oldVer.Version.Resources[0].Checksum == configMd5 {
			glog.Info(w.ctx, "webConsole skip equal version", glog.String("key", etcdKey), glog.String("md5", configMd5))
			return nil
		}
	}

	lock, err := newLocker(w.etcdClient.GetRawClient(), "webConsole")
	if err != nil {
		glog.Error(w.ctx, "webConsole NewLocker error", glog.String("key", ac.Key()), glog.String("error", err.Error()))
		return err
	}
	unlock, err := lock.tryLock(w.ctx, etcdKey, 1*time.Minute)
	if err != nil {
		glog.Error(w.ctx, "webConsole TryLock error", glog.String("key", ac.Key()), glog.String("error", err.Error()))
		return err
	}
	if unlock == nil {
		glog.Info(w.ctx, "webConsole TryLock fail", glog.String("key", ac.Key()))
		return nil
	}
	defer func() {
		glog.Info(w.ctx, "webConsole unlockFunc invoke", glog.String("key", ac.Key()))
		if err = unlock(); err != nil {
			glog.Error(w.ctx, "webConsole unlockFunc error", glog.String("key", ac.Key()), glog.String("error", err.Error()))
		}
	}()

	now := time.Now()
	configKey := fmt.Sprintf("%s.%s.%s", ac.App, ac.Module, now.Format("20060102150405"))
	glog.Info(w.ctx, "nacos key", glog.String("key", configKey))
	err = w.nacosCfg.Put(w.ctx, configKey, cast.UnsafeBytesToString(data))
	if err != nil {
		glog.Error(w.ctx, "nacosCfg.Put error", glog.String("key", configKey), glog.String("error", err.Error()))
		return err
	}

	appVersion := &core.AppVersion{
		Namespace: config.GetNamespace(),
		Group:     config.GetGroup(),
		App:       ac.App,
		Module:    ac.Module,
		Version: core.AppVersionEntry{
			LastTime: now,
			Resources: [2]core.ResourceEntry{
				{
					ResourceType: core.ResourceConfig,
					LastTime:     now,
					Checksum:     configMd5,
				},
				{
					ResourceType: core.ResourceDesc,
				},
			},
		},
	}

	appVersion.History = []core.AppVersionEntry{appVersion.Version}

	if err = w.setVersion(appVersion); err != nil {
		msg := fmt.Sprintf(WebConsoleAlertErrFormat, err.Error(), configKey)
		galert.Error(w.ctx, msg, galert.WithTitle(WebConsoleAlertTitle))
	} else {
		msg := fmt.Sprintf("web console success, evenKey = %s", configKey)
		galert.Info(w.ctx, msg, galert.WithTitle(WebConsoleAlertTitle))
	}
	return nil
}

// setVersion  set version to etcd
func (w *webConsole) setVersion(version *core.AppVersion) error {
	key := filepath.Join(constant.RootPath, config.GetNamespace(), config.GetGroup(), version.App, version.Module, constant.EtcdDeployKey)
	glog.Info(w.ctx, "etcd key, wait nacos data 10s", glog.String("key", key))
	time.Sleep(10 * time.Second) // wait for nacos data
	data, err := json.Marshal(version)
	if err != nil {
		glog.Error(w.ctx, "json.Marshal app failed", glog.String("error", err.Error()), glog.String("key", key))
		return err
	}
	if err := w.etcdClient.Put(key, cast.UnsafeBytesToString(data)); err != nil {
		return err
	}

	oldkey := filepath.Join(constant.RootPath, config.GetNamespace(), version.App, version.Module, constant.EtcdDeployKey)
	if err := w.etcdClient.Delete(oldkey); err != nil {
		glog.Error(w.ctx, "delete old version error", glog.String("key", oldkey), glog.String("error", err.Error()))
	}

	return nil
}

func (w *webConsole) staticPreCheck(key, content string) (*core.AppConfig, error) {
	ac := &core.AppConfig{}
	if err := ac.Unmarshal(bytes.NewReader([]byte(content)), "yaml"); err != nil {
		glog.Error(w.ctx, "Unmarshal app failed", glog.String("error", err.Error()), glog.String("file", key))
		return nil, err
	}

	if ac.App == "" || ac.Module == "" {
		glog.Error(w.ctx, "config fill error, nil app or module", glog.String("file", key))
		return nil, errors.New("config fill error, nil app or module")
	}
	if !strings.HasSuffix(ac.Module, "-http") {
		glog.Error(w.ctx, "config fill error, module name must suffix with -http", glog.String("file", key))
		return nil, errors.New("config fill error, module name must suffix with -http")
	}

	uniqueRegisterMgr := &unique{
		ctx:            w.ctx,
		uniqueRegister: make(map[string]struct{}, 5),
	}
	for _, service := range ac.Services {
		if service.Registry == "" {
			glog.Error(w.ctx, "config fill error, nil registry", glog.String("file", key))
			return nil, errors.New("config fill error, nil registry")
		}
		if service.Protocol != constant.HttpProtocol {
			service.Protocol = constant.HttpProtocol
		}

		for _, method := range service.Methods {
			if err := w.checkConfigPath(method, uniqueRegisterMgr); err != nil {
				glog.Error(w.ctx, "checkConfigPath error", glog.String("file", key), glog.String("error", err.Error()))
				return nil, err
			}
		}
		if !env.IsProduction() {
			// test env service discovery namespace and group
			if service.Namespace == "" {
				service.Namespace = config.GetNamespace()
			}
			if service.Group == "" {
				service.Group = constant.DEFAULT_GROUP
			}
		}
	}

	// sort for version md5
	for _, service := range ac.Services {
		sort.Slice(service.Methods, func(i, j int) bool {
			if service.Methods[i].Path != "" && service.Methods[j].Path != "" {
				return service.Methods[i].Path < service.Methods[j].Path
			}
			if service.Methods[i].Path != "" {
				return service.Methods[i].Path < service.Methods[j].Paths[0]
			}
			return service.Methods[i].Paths[0] < service.Methods[j].Path
		})
	}
	sort.Slice(ac.Services, func(i, j int) bool {
		return ac.Services[i].Registry < ac.Services[j].Registry
	})

	return ac, nil
}

func (w *webConsole) checkConfigPath(method *core.MethodConfig, uniqueRegisterMgr *unique) error {
	if method.HttpMethod == "" && len(method.HttpMethods) == 0 {
		return errors.New("config fill error, nil http method")
	}
	if method.Path == "" && len(method.Paths) == 0 {
		return errors.New("config fill error, nil path or paths")
	}

	var meths []string
	if method.HttpMethod != "" {
		if method.HttpMethod == bhttp.HTTPMethodAny {
			meths = bhttp.HttpMethodAnyConfig
		} else {
			meths = append(meths, method.HttpMethod)
		}
	}
	for _, httpMethod := range method.HttpMethods {
		if method.HttpMethod == bhttp.HTTPMethodAny {
			return errors.New("method config fill error, HttpMethods not support any method")
		}
		meths = append(meths, httpMethod)
	}
	if method.Path != "" {
		for _, meth := range meths {
			if err := uniqueRegisterMgr.checkMethodPathUnique(meth, method.Path); err != nil {
				return err
			}
		}
	}
	for _, path := range method.Paths {
		for _, meth := range meths {
			if err := uniqueRegisterMgr.checkMethodPathUnique(meth, path); err != nil {
				return err
			}
		}
	}
	return nil
}

type unique struct {
	ctx            context.Context
	uniqueRegister map[string]struct{}
}

func (u *unique) checkMethodPathUnique(meth, path string) error {
	full := meth + path
	if _, ok := u.uniqueRegister[full]; ok {
		glog.Info(u.ctx, "method+path exists", glog.String("full", full))
		return errors.New("method+path exists: " + full)
	}
	u.uniqueRegister[full] = struct{}{}
	return nil
}

func (w *webConsole) GetEventType() reflect.Type {
	return nil
}

func (w *webConsole) GetPriority() int {
	return 0
}

type locker struct {
	client *clientv3.Client
	prefix string
}

func newLocker(client *clientv3.Client, prefix string) (*locker, error) {
	l := &locker{
		client: client,
		prefix: filepath.Join(constant.RootPath, prefix),
	}

	return l, nil
}

func (l *locker) tryLock(ctx context.Context, key string, leaseExpire time.Duration) (unlockFunc unlockFunc, err error) {
	if err = ctx.Err(); err != nil {
		return nil, err
	}

	etcdKey := filepath.Join(l.prefix, key)
	lease := clientv3.NewLease(l.client)
	leaseGrantResp, err := lease.Grant(ctx, 5)
	if err != nil {
		return nil, fmt.Errorf("new raw client Grant error:%w", err)
	}
	leaseId := leaseGrantResp.ID
	unlockFunc = l.unlock(lease, leaseId)

	keepRespChan, err := lease.KeepAlive(ctx, leaseId)
	if err != nil {
		return nil, fmt.Errorf("new raw client KeepAlive error:%w", err)
	}

	go func() {
		timer := time.NewTimer(leaseExpire)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				glog.Info(ctx, "lease expired from leaseExpire", glog.String("key", etcdKey))
				if _, err = lease.Revoke(context.Background(), leaseId); err != nil {
					glog.Error(ctx, "lease expired from leaseExpire Revoke error", glog.String("key", etcdKey), glog.String("error", err.Error()))
				}
				return
			case keepResp := <-keepRespChan:
				if keepResp == nil {
					glog.Info(ctx, "lease expired from keep chan", glog.String("key", etcdKey))
					return
				} else {
					glog.Info(ctx, "lease receive response", glog.Int64("lease_id", int64(keepResp.ID)))
				}
			}
		}
	}()

	kv := clientv3.NewKV(l.client)
	txn := kv.Txn(ctx)
	txn.If(clientv3.Compare(clientv3.CreateRevision(etcdKey), "=", 0)).
		Then(clientv3.OpPut(etcdKey, "lock", clientv3.WithLease(leaseId))).
		Else(clientv3.OpGet(etcdKey))

	txnResp, err := txn.Commit()
	if err != nil {
		return nil, fmt.Errorf("etcd lock commit error:%w", err)
	}

	if !txnResp.Succeeded {
		glog.Info(ctx, "lock fail", glog.String("key", string(txnResp.Responses[0].GetResponseRange().Kvs[0].Key)), glog.String("value", string(txnResp.Responses[0].GetResponseRange().Kvs[0].Value)))
		return nil, nil
	}

	return unlockFunc, nil
}

type unlockFunc func() error

func (l *locker) unlock(lease clientv3.Lease, id clientv3.LeaseID) unlockFunc {
	return func() error {
		if _, err := lease.Revoke(context.Background(), id); err != nil {
			return err
		}
		return nil
	}
}
