package etcd

import (
	"context"
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gconfig"
)

func TestEtcd(t *testing.T) {
	addr := "k8s-istiosys-unifytes-b5f00eb0c2-7e5143857d7256e2.elb.ap-southeast-1.amazonaws.com:2379"
	c, err := New(addr)

	if err != nil {
		t.Error(err)
	}

	t.Log(c.Get(context.Background(), "abc"))

	const key = "gway_test_key"
	if value, err := c.Get(context.Background(), key); err != nil || value != "" {
		t.Errorf("should empty data, err=%v, data=%v", err, value)
	}

	putData := `{"name": "test"}`
	if err := c.Put(context.Background(), key, putData); err != nil {
		t.Errorf("put fail, err=%v", err)
	}

	time.Sleep(time.Second)

	if value, err := c.Get(context.Background(), key); err != nil || value != putData {
		t.Errorf("get data fail, err=%v, data=%v", err, value)
	} else {
		t.Logf("get data ok, data=%v", value)
	}

	if err := c.Listen(context.Background(), key, gconfig.ListenFunc(func(ev *gconfig.Event) {
		if ev.Value != putData && ev.Value != "" {
			t.Errorf("listen data fail, data: %v", ev.Value)
		} else {
			t.Log("listen data:", ev.Type.String(), ev.Key, ev.Value)
		}
	})); err != nil {
		t.Errorf("listen data fail, err: %v", err)
	}

	time.Sleep(time.Second)
	t.Logf("wait listen data")

	if err := c.Put(context.Background(), key, putData); err != nil {
		t.Errorf("put fail, err=%v", err)
	}
	t.Logf("put data, need triger listen")

	time.Sleep(time.Second)

	t.Logf("delete data")
	if err := c.Delete(context.Background(), key); err != nil {
		t.Errorf("del data fail, err=%v", err)
	}

	time.Sleep(time.Second)

	if value, err := c.Get(context.Background(), key); err != nil || value != "" {
		t.Errorf("should empty data, err=%v, data=%v", err, value)
	}
}
