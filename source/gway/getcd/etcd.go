package getcd

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	perrors "github.com/pkg/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
)

type Client interface {
	GetCtx() context.Context
	Close()
	GetRawClient() *clientv3.Client
	GetEndPoints() []string
	GetChildren(k string, isDir ...bool) ([]string, []string, error)
	Done() <-chan struct{}
	Valid() bool
	Create(k string, v string) error
	BatchCreate(kList []string, vList []string) error
	Update(k, v string) error
	Put(k, v string, opts ...clientv3.OpOption) error
	UpdateWithRev(k, v string, rev int64, opts ...clientv3.OpOption) error
	Delete(k string) error
	RegisterTemp(k, v string) error
	GetChildrenKVList(k string) ([]string, []string, error)
	GetValAndRev(k string) (string, int64, error)
	Get(k string) (string, error)
	Watch(k string) (clientv3.WatchChan, error)
	WatchWithPrefix(prefix string) (clientv3.WatchChan, error)
	WatchWithOption(k string, opts ...clientv3.OpOption) (clientv3.WatchChan, error)
}

var (
	ErrETCDEndpoints = perrors.New("etcd remote config error, invalid endpoints")
	// ErrNilETCDV3Client raw client nil
	ErrNilETCDV3Client = perrors.New("etcd raw client is nil") // full describe the ERR
	// ErrKVPairNotFound not found key
	ErrKVPairNotFound = perrors.New("k/v pair not found")
	// ErrKVListSizeIllegal k/v list empty or not equal size
	ErrKVListSizeIllegal = perrors.New("k/v List is empty or kList's size is not equal to the size of vList")
	// ErrCompareFail txn compare fail
	ErrCompareFail = perrors.New("txn compare fail")
	// ErrRevision revision when error
	ErrRevision int64 = -1
)

var (
	clientPool *etcdClientPool
	// If the WithEndpoints function is not used to set endpoints,
	// GWAY_GETCD_ENDPOINTS in the environment variable is used
	envName = "GWAY_GETCD_ENDPOINTS"
)

type etcdClientPool struct {
	sync.RWMutex
	clients map[string]Client
}

func init() {
	clientPool = &etcdClientPool{
		clients: make(map[string]Client),
	}
}

// NewClient create new etcd client
func NewClient(ctx context.Context, opts ...Option) (Client, error) {

	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	if len(options.Endpoints) == 0 {
		endpoints := strings.TrimSpace(os.Getenv(envName))
		if len(endpoints) > 0 {
			options.Endpoints = strings.Split(endpoints, ",")
		}
		if len(options.Endpoints) == 0 {
			return nil, ErrETCDEndpoints
		}
	}

	return newClient(ctx, options.Name, options.Endpoints, options.Timeout, options.Heartbeat, options.Username, options.Password)
}

// client represents etcd client Configuration
type client struct {
	lock     sync.RWMutex
	quitOnce sync.Once

	// these properties are only set once when they are started.
	name      string
	endpoints []string
	timeout   time.Duration
	heartbeat int

	ctx       context.Context    // if etcd server connection lose, the ctx.Done will be sent msg
	cancel    context.CancelFunc // cancel the ctx, all watcher will stopped
	rawClient *clientv3.Client

	exit chan struct{}
	Wait sync.WaitGroup
}

// newClient create a client instance with name, endpoints etc.
func newClient(ctx context.Context, name string, endpoints []string, timeout time.Duration, heartbeat int, username, password string) (Client, error) {
	key := strings.Join(endpoints, "-")

	clientPool.Lock()
	defer clientPool.Unlock()

	cli, ok := clientPool.clients[key]
	if ok {
		return cli, nil
	}

	if ctx == nil {
		ctx = context.Background()
	}
	cancelCtx, cancel := context.WithCancel(ctx)

	rawClient, err := clientv3.New(clientv3.Config{
		Context:     cancelCtx,
		Endpoints:   endpoints,
		DialTimeout: timeout,
		Username:    username,
		Password:    password,
		DialOptions: []grpc.DialOption{
			grpc.WithBlock(),
			grpc.WithConnectParams(
				grpc.ConnectParams{
					Backoff: backoff.Config{
						BaseDelay:  time.Second,
						Multiplier: 1.6,
						Jitter:     0.2,
						MaxDelay:   30 * time.Second,
					},
					MinConnectTimeout: 2 * time.Second,
				},
			),
		},
	})
	if err != nil {
		cancel()
		return nil, perrors.WithMessage(err, "new raw client block connect to server")
	}

	c := &client{
		name:      name,
		timeout:   timeout,
		endpoints: endpoints,
		heartbeat: heartbeat,

		ctx:       ctx,
		cancel:    cancel,
		rawClient: rawClient,

		exit: make(chan struct{}),
	}

	c.Wait.Add(1)
	go c.keepSessionLoop()
	clientPool.clients[key] = c

	return c, nil
}

// NOTICE: need to get the lock before calling this method
func (c *client) clean() {
	// close raw client
	_ = c.rawClient.Close()

	// cancel ctx for raw client
	c.cancel()

	// clean raw client
	c.rawClient = nil
}

func (c *client) stop() bool {
	select {
	case <-c.exit:
		return false
	default:
		ret := false
		c.quitOnce.Do(func() {
			ret = true
			close(c.exit)
		})
		return ret
	}
}

// GetCtx return client context
func (c *client) GetCtx() context.Context {
	return c.ctx
}

// Close close client
func (c *client) Close() {
	if c == nil {
		return
	}

	// stop the client
	if ret := c.stop(); !ret {
		return
	}

	// wait client keep session stop
	c.Wait.Wait()

	c.lock.Lock()
	defer c.lock.Unlock()
	if c.rawClient != nil {
		c.clean()
	}
	log.Println("etcd client exit now", c.name, c.endpoints)
}

func (c *client) keepSessionLoop() {
	defer func() {
		c.Wait.Done()
		log.Println("etcd client keep goroutine game over.", c.name, c.endpoints)
	}()

	for {
		select {
		case <-c.Done():
			// Client be stopped, will clean the client hold resources
			return
		case <-c.ctx.Done():
			log.Println("etcd will quit")
			c.lock.Lock()
			// when etcd server stopped, cancel ctx, stop all watchers
			c.clean()
			// when connection lose, stop client, trigger reconnect to etcd
			c.stop()
			c.lock.Unlock()
			return
		}
	}
}

// GetRawClient return etcd raw client
func (c *client) GetRawClient() *clientv3.Client {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.rawClient
}

// GetEndPoints return etcd endpoints
func (c *client) GetEndPoints() []string {
	return c.endpoints
}

// if k not exist will put k/v in etcd, otherwise return ErrCompareFail
func (c *client) create(k string, v string, opts ...clientv3.OpOption) error {
	rawClient := c.GetRawClient()
	if rawClient == nil {
		return ErrNilETCDV3Client
	}

	resp, err := rawClient.Txn(c.ctx).
		If(clientv3.Compare(clientv3.CreateRevision(k), "=", 0)).
		Then(clientv3.OpPut(k, v, opts...)).
		Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return ErrCompareFail
	}

	return nil
}

// if k in bulk insertion not exist all, then put all k/v in etcd, otherwise return error
func (c *client) batchCreate(kList []string, vList []string, opts ...clientv3.OpOption) error {
	rawClient := c.GetRawClient()
	if rawClient == nil {
		return ErrNilETCDV3Client
	}

	kLen := len(kList)
	vLen := len(vList)
	if kLen == 0 || vLen == 0 || kLen != vLen {
		return ErrKVListSizeIllegal
	}

	var cs []clientv3.Cmp
	var ops []clientv3.Op

	for i, k := range kList {
		v := vList[i]
		cs = append(cs, clientv3.Compare(clientv3.CreateRevision(k), "=", 0))
		ops = append(ops, clientv3.OpPut(k, v, opts...))
	}

	resp, err := rawClient.Txn(c.ctx).
		If(cs...).
		Then(ops...).
		Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return ErrCompareFail
	}

	return nil
}

// put k/v in etcd, if fail return error
func (c *client) put(k string, v string, opts ...clientv3.OpOption) error {
	rawClient := c.GetRawClient()
	if rawClient == nil {
		return ErrNilETCDV3Client
	}

	_, err := rawClient.Put(c.ctx, k, v, opts...)
	return err
}

// put k/v in etcd when ModRevision equal with rev, if not return ErrCompareFail or other err
func (c *client) updateWithRev(k string, v string, rev int64, opts ...clientv3.OpOption) error {
	rawClient := c.GetRawClient()
	if rawClient == nil {
		return ErrNilETCDV3Client
	}

	resp, err := rawClient.Txn(c.ctx).
		If(clientv3.Compare(clientv3.ModRevision(k), "=", rev)).
		Then(clientv3.OpPut(k, v, opts...)).
		Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return ErrCompareFail
	}

	return nil
}

func (c *client) delete(k string) error {
	rawClient := c.GetRawClient()
	if rawClient == nil {
		return ErrNilETCDV3Client
	}

	_, err := rawClient.Delete(c.ctx, k)
	return err
}

// getValAndRev get value and revision
func (c *client) getValAndRev(k string) (string, int64, error) {
	rawClient := c.GetRawClient()
	if rawClient == nil {
		return "", ErrRevision, ErrNilETCDV3Client
	}

	resp, err := rawClient.Get(c.ctx, k)
	if err != nil {
		return "", ErrRevision, err
	}

	if len(resp.Kvs) == 0 {
		return "", ErrRevision, ErrKVPairNotFound
	}

	return string(resp.Kvs[0].Value), resp.Header.Revision, nil
}

// GetChildren return node children
func (c *client) GetChildren(k string, isDir ...bool) ([]string, []string, error) {
	rawClient := c.GetRawClient()
	if rawClient == nil {
		return nil, nil, ErrNilETCDV3Client
	}

	if len(isDir) > 0 && isDir[0] && !strings.HasSuffix(k, "/") {
		k += "/"
	}

	resp, err := rawClient.Get(c.ctx, k, clientv3.WithPrefix())
	if err != nil {
		return nil, nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, nil, ErrKVPairNotFound
	}

	kList := make([]string, 0, len(resp.Kvs))
	vList := make([]string, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		kList = append(kList, string(kv.Key))
		vList = append(vList, string(kv.Value))
	}
	return kList, vList, nil
}

// watchWithOption watch
func (c *client) watchWithOption(k string, opts ...clientv3.OpOption) (clientv3.WatchChan, error) {
	rawClient := c.GetRawClient()
	if rawClient == nil {
		return nil, ErrNilETCDV3Client
	}

	return rawClient.Watch(c.ctx, k, opts...), nil
}

func (c *client) keepAliveKV(k string, v string) error {
	rawClient := c.GetRawClient()
	if rawClient == nil {
		return ErrNilETCDV3Client
	}

	// make lease time longer, since 1 second is too short
	lease, err := rawClient.Grant(c.ctx, int64(30*time.Second.Seconds()))
	if err != nil {
		return perrors.WithMessage(err, "grant lease")
	}

	keepAlive, err := rawClient.KeepAlive(c.ctx, lease.ID)
	if err != nil || keepAlive == nil {
		_, err = rawClient.Revoke(c.ctx, lease.ID)
		if err != nil {
			return perrors.WithMessage(err, "keep alive lease")
		}
		return perrors.New("keep alive lease")
	}

	_, err = rawClient.Put(c.ctx, k, v, clientv3.WithLease(lease.ID))
	return perrors.WithMessage(err, "put k/v with lease")
}

// Done return exit chan
func (c *client) Done() <-chan struct{} {
	return c.exit
}

// Valid check client
func (c *client) Valid() bool {
	select {
	case <-c.exit:
		return false
	default:
	}

	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.rawClient != nil
}

// Create key value ...
func (c *client) Create(k string, v string) error {
	err := c.create(k, v)
	return perrors.WithMessagef(err, "put k/v (key: %s value %s)", k, v)
}

// BatchCreate bulk insertion
func (c *client) BatchCreate(kList []string, vList []string) error {
	err := c.batchCreate(kList, vList)
	return perrors.WithMessagef(err, "batch put k/v error ")
}

// Update key value ...
func (c *client) Update(k, v string) error {
	err := c.put(k, v)
	return perrors.WithMessagef(err, "Update k/v (key: %s value %s)", k, v)
}

// Put key value ...
func (c *client) Put(k, v string, opts ...clientv3.OpOption) error {
	err := c.put(k, v, opts...)
	return perrors.WithMessagef(err, "Put k/v (key: %s value %s)", k, v)
}

// UpdateWithRev update key value ...
func (c *client) UpdateWithRev(k, v string, rev int64, opts ...clientv3.OpOption) error {
	err := c.updateWithRev(k, v, rev, opts...)
	return perrors.WithMessagef(err, "Update k/v (key: %s value %s)", k, v)
}

// Delete key
func (c *client) Delete(k string) error {
	err := c.delete(k)
	return perrors.WithMessagef(err, "delete k/v (key %s)", k)
}

// RegisterTemp registers a temporary node
func (c *client) RegisterTemp(k, v string) error {
	err := c.keepAliveKV(k, v)
	return perrors.WithMessagef(err, "keepalive kv (key %s)", k)
}

// GetChildrenKVList gets children kv list by @k
func (c *client) GetChildrenKVList(k string) ([]string, []string, error) {
	kList, vList, err := c.GetChildren(k)
	return kList, vList, perrors.WithMessagef(err, "get key children (key %s)", k)
}

// GetValAndRev gets value and revision by @k
func (c *client) GetValAndRev(k string) (string, int64, error) {
	v, rev, err := c.getValAndRev(k)
	return v, rev, perrors.WithMessagef(err, "get key value (key %s)", k)
}

// Get gets value by @k
func (c *client) Get(k string) (string, error) {
	v, _, err := c.getValAndRev(k)
	return v, perrors.WithMessagef(err, "get key value (key %s)", k)
}

// Watch watches on spec key
func (c *client) Watch(k string) (clientv3.WatchChan, error) {
	wc, err := c.watchWithOption(k)
	return wc, perrors.WithMessagef(err, "watch (key %s)", k)
}

// WatchWithPrefix watches on spec prefix
func (c *client) WatchWithPrefix(prefix string) (clientv3.WatchChan, error) {
	wc, err := c.watchWithOption(prefix, clientv3.WithPrefix())
	return wc, perrors.WithMessagef(err, "watch prefix (key %s)", prefix)
}

// WatchWithOption watches on spc key with OpOption
func (c *client) WatchWithOption(k string, opts ...clientv3.OpOption) (clientv3.WatchChan, error) {
	wc, err := c.watchWithOption(k, opts...)
	return wc, perrors.WithMessagef(err, "watch (key %s)", k)
}
