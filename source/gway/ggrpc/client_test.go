package ggrpc

import (
	"context"
	"log"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func TestDial(t *testing.T) {
	addr := "nacos:///uta_router?namespace=unify-test-1"
	if _, err := Dial(context.Background(), addr); err != nil {
		t.Error(err)
	}
	time.Sleep(time.Second * 10)
}

func TestDialAZRR(t *testing.T) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	addr := "nacos:///masq?namespace=cht-test-1"
	if _, err := Dial(context.Background(), addr, grpc.WithDefaultServiceConfig(`{"LoadBalancingPolicy":"az_round_robin"}`)); err != nil {
		t.Error(err)
	}
	time.Sleep(time.Second * 5)
}
