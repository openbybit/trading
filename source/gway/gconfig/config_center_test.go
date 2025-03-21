package gconfig

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	Register("mock", NewMockByURL)
	c, err := New("mock://?k1=v1")
	if err != nil {
		t.Errorf("create configure fail")
	}
	v1, err := c.Get(context.Background(), "k1")
	if err != nil || v1 != "v1" {
		t.Errorf("invalid vaile,err:%v, vaule: %v", err, v1)
	}
}
