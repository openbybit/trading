package nacos

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gconfig"
)

func TestParseAddress(t *testing.T) {
	list := []string{
		"unify-test-1",
		"nacos.test.infra.ww5sawfyut0k.bitsvc.io:8848?namespace=public",
		"nacos://nacos.test.infra.ww5sawfyut0k.bitsvc.io:8848?namespace=public",
	}

	for _, x := range list {
		u, err := parseAddress(x)
		if err != nil {
			t.Error(err)
		} else {
			t.Log(u.String())
		}
	}

	x, _ := url.Parse("nacos:?namespace=unify-test-1")
	t.Logf("scheme: %s, host: %s, query: %s", x.Scheme, x.Host, x.RawQuery)
}

func TestUrl(t *testing.T) {
	addr := "http://nacos:nacos@nacos.test.infra.ww5sawfyut0k.bitsvc.io:8848?namespace=public&cache_dir=/tmp/nacos/cache&log_dir=/tmp/nacos/log&log_level=debug"
	u, err := url.Parse(addr)
	if err != nil {
		t.Error(err)
	} else {
		pass, _ := u.User.Password()
		t.Log(u.Scheme, "host", u.Host, "name", u.Hostname(), "port", u.Port(), u.User.Username(), pass)
		q := u.Query()
		t.Log(q.Get("namespace"))
		t.Log(q.Get("cache_dir"))
		t.Log(q.Get("log_dir"))
		t.Log(q.Get("log_level"))
	}

	if u, err := url.Parse("http://?group=uta"); err != nil {
		t.Error(err)
	} else {
		t.Log(u.Query().Get("group"))
	}

	if u, err := url.Parse("http://nacos:nacos@"); err != nil {
		t.Error(err)
	} else {
		t.Log(u.Host, u.User)
	}
}

func TestLoad(t *testing.T) {
	type TestConfig struct {
		App  string `json:"app" yaml:"app"`
		Name string `json:"name" yaml:"name"`
	}
	addr := "http://nacos:nacos@nacos.test.infra.ww5sawfyut0k.bitsvc.io:8848?namespace=bgw"
	const key = "test_key"

	configure, err := New(addr)
	if err != nil {
		t.Error(err)
	}
	conf := &TestConfig{}
	if err := gconfig.Load(context.Background(), configure, key, &conf, nil, nil); err != nil {
		t.Error(err)
	}

	fmt.Println(conf)

	time.Sleep(time.Second * 20)
}

func TestNacos(t *testing.T) {
	addr := "http://nacos:nacos@nacos.test.infra.ww5sawfyut0k.bitsvc.io:8848?namespace=bgw"
	c, err := New(addr)

	if err != nil {
		t.Error(err)
	}

	const key = "gway_test_key"
	if value, err := c.Get(nil, key); err != nil || value != "" {
		t.Errorf("should empty data, err=%v, data=%v", err, value)
	}

	putData := `{"name": "test"}`
	if err := c.Put(nil, key, putData); err != nil {
		t.Errorf("put fail, err=%v", err)
	}

	time.Sleep(time.Second)

	if value, err := c.Get(nil, key); err != nil || value != putData {
		t.Errorf("get data fail, err=%v, data=%v", err, value)
	}

	if err := c.Listen(nil, key, gconfig.ListenFunc(func(ev *gconfig.Event) {
		if ev.Value != putData {
			t.Errorf("listen data fail, data: %v", ev.Value)
		} else {
			t.Log("listen data:", ev.Type, ev.Key, ev.Value)
		}
	}), gconfig.WithForceGet(true)); err != nil {
		t.Errorf("listen data fail, err: %v", err)
	}

	if err := c.Put(nil, key, putData); err != nil {
		t.Errorf("put fail, err=%v", err)
	}

	time.Sleep(time.Second)

	if err := c.Delete(nil, key); err != nil {
		t.Errorf("del data fail, err=%v", err)
	}

	time.Sleep(time.Second)

	if value, err := c.Get(nil, key); err != nil || value != "" {
		t.Errorf("should empty data, err=%v, data=%v", err, value)
	}
}
