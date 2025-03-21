package nacos

import (
	"context"
	"testing"

	"github.com/tj/assert"

	"bgw/pkg/config"
)

// func TestInitNacosConfigCenter(t *testing.T) {
// 	a := assert.New(t)
// 	ctx := context.TODO()
// 	cc, err := NewNacosConfigure(ctx)
// 	a.Nil(err)
// 	content, err := cc.Get(context.TODO(), "trade_demo")
// 	a.NoError(err)

// 	app := &config.AppConfig{}
// 	err = app.Unmarshal(bytes.NewBufferString(content), "json")
// 	a.NoError(err)

// 	for _, srv := range app.Services {
// 		for _, meth := range srv.Methods {
// 			a.NotNil(meth.Service())
// 		}
// 	}

// 	ch := make(chan observer.Event)
// 	l := config_center.NewDefaultListener(context.TODO(), ch)
// 	err = cc.Listen(context.Background(), "trade_demo", l)
// 	a.Nil(err)

// 	select {
// 	case <-ch:
// 		t.Log("got event")
// 	case <-time.After(30 * time.Second):
// 		t.Fatal("timeout")
// 	}

// 	app2 := &config.AppConfig{}
// 	current := l.Get("trade_demo")

// 	err = app2.Unmarshal(bytes.NewBufferString(current.Content), "json")
// 	a.NoError(err)
// }

func TestGet(t *testing.T) {
	a := assert.New(t)
	ctx := context.TODO()
	cc, err := NewNacosConfigure(ctx, WithGroup(config.GetGroup()))
	a.Nil(err)
	data, err := cc.Get(context.TODO(), "OPTION.trading.20210902061814")
	a.Nil(err)
	a.NotEmpty(data)
	t.Log(data)
}
