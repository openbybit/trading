package gapp_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gapp"
)

// go test -v -ldflags -s -run ^TestApp$
func TestApp(t *testing.T) {
	svc := &demoService{}
	err := gapp.Run(gapp.WithDefaultEndpoints(), gapp.WithHealth(svc.Health), gapp.WithLifecycles(svc))
	if err != nil {
		t.Error(err)
	}
}

// go test -v -ldflags -s -run ^TestTimeout$
func TestTimeout(t *testing.T) {
	svc := &demoService{testTimeout: true}
	err := gapp.Run(gapp.WithDefaultEndpoints(), gapp.WithHealth(svc.Health), gapp.WithLifecycles(svc))
	if err != nil {
		t.Log(err)
	} else {
		t.Error("should exit timeout")
	}
}

// go test -v -ldflags -s -run ^TestError$
func TestError(t *testing.T) {
	svc := &demoService{testError: true}
	err := gapp.Run(gapp.WithDefaultEndpoints(), gapp.WithHealth(svc.Health), gapp.WithLifecycles(svc), gapp.WithExitTimeout(time.Second*2))
	if err != nil {
		t.Log(err)
	} else {
		t.Error("should exit timeout")
	}
}

type demoService struct {
	testTimeout bool
	testError   bool
}

func (s *demoService) Health() (bool, interface{}) {
	return true, nil
}

func (s *demoService) OnLifecycle(ctx context.Context, event gapp.LifecycleEvent) error {
	switch event {
	case gapp.LifecycleStart:
		fmt.Println("demo start")
	case gapp.LifecycleStop:
		fmt.Println("demo stop")
		if s.testError {
			return fmt.Errorf("test error")
		}
		if s.testTimeout {
			time.Sleep(time.Second * 4)
		}
	}
	return nil
}
