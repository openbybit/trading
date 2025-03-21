package roundrobin

import (
	"context"
	"fmt"
	"testing"

	"bgw/pkg/common/types"
	"bgw/pkg/registry"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	. "github.com/smartystreets/goconvey/convey"
)

// d *demo bgw/pkg/registry.ServiceInstance
type demo struct {
	id          string
	serviceName string
	host        string
	port        int
	weight      int64
	metadata    registry.Metadata
}

func (d *demo) GetCluster() string {
	return ""
}

// GetID will return this instance's id. It should be unique.
func (d *demo) GetID() string {
	return d.id
}

// GetServiceName will return the serviceName
func (d *demo) GetServiceName() string {
	return d.serviceName
}

// GetHost will return the hostname
func (d *demo) GetHost() string {
	return d.host
}

// GetPort will return the port.
func (d *demo) GetPort() int {
	return d.port
}

// IsEnable will return the enable status of this instance
func (d *demo) IsEnable() bool {
	return true
}

// IsHealthy will return the value represent the instance whether healthy or not
func (d *demo) IsHealthy() bool {
	return true
}

func (d *demo) GetWeight() int64 {
	return d.weight
}

// GetMetadata will return the metadata
func (d *demo) GetMetadata() registry.Metadata {
	return d.metadata
}

// GetEndPoints
func (d *demo) GetEndPoints() []*registry.Endpoint {
	panic("not implemented") // TODO: Implement
}

// Copy
func (d *demo) Copy(endpoint *registry.Endpoint) registry.ServiceInstance {
	panic("not implemented") // TODO: Implement
}

// GetAddress
func (d *demo) GetAddress(protocol string) string {
	return fmt.Sprintf("%s:%d", d.host, d.port)
}

// metadate wight must  mot  be  zero and GetID()  not  ""
func Test_roundRobinLoadBalance_Select(t *testing.T) {
	type args struct {
		ctx       *types.Ctx
		instances []registry.ServiceInstance
	}
	tests := []struct {
		name string
		lb   *roundRobinLoadBalance
		args args
		want registry.ServiceInstance
	}{
		{
			name: "demos",
			lb:   &roundRobinLoadBalance{},
			args: args{
				ctx: nil,
				instances: []registry.ServiceInstance{
					&demo{
						id:          "demo1",
						serviceName: "demo",
						host:        "127.0.0.1",
						port:        123,
						weight:      1,
						metadata: registry.Metadata{
							"weidht": "1",
						},
					},
					&demo{
						id:          "demo2",
						serviceName: "demo",
						host:        "127.0.0.1",
						port:        124,
						weight:      2,
						metadata: registry.Metadata{
							"weidht": "2",
						},
					},
					&demo{
						id:          "demo3",
						serviceName: "demo",
						host:        "127.0.0.1",
						port:        125,
						weight:      3,
						metadata: registry.Metadata{
							"weidht": "3",
						},
					},
				},
			},
			want: nil,
		},
	}
	count := make(map[string]int)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb := &roundRobinLoadBalance{}
			for i := 0; i < 600; i++ {
				got, _ := lb.Select(tt.args.ctx, tt.args.instances)
				// t.Log("current", got.GetAddress())
				count[got.GetID()] = count[got.GetID()] + 1
			}
		})
	}
	for k, v := range count {
		t.Logf("instance:%s,count:%d", k, v)

	}

}

type mockInstance struct {
	registry.ServiceInstance
	name   string
	weight int
}

func (m *mockInstance) GetID() string {
	return m.name
}

func (m *mockInstance) GetMetadata() registry.Metadata {
	return registry.Metadata{
		"weight": cast.ToString(m.weight),
	}
}

func BenchmarkSelect(b *testing.B) {
	var inss []registry.ServiceInstance

	for i := 0; i < 10; i++ {
		inss = append(inss, &mockInstance{
			name:   cast.ToString(i),
			weight: i,
		})
	}

	m := make(map[string]int)
	selector := New()
	for i := 0; i < b.N; i++ {
		ins, _ := selector.Select(context.TODO(), inss)
		m[ins.GetID()]++
	}

	b.Log(m)
}

func TestRoundRobinLoadBalance_setWeight(t *testing.T) {
	Convey("test RoundRobinLoadBalance setWeight", t, func() {
		wrr := &weightedRoundRobin{}
		wrr.setWeight(10)
	})
}
