package cluster

import (
	"context"
	"fmt"

	"bgw/pkg/common"
	"bgw/pkg/common/constant"
	"bgw/pkg/registry"
	"bgw/pkg/server/metadata"

	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/glog"
)

var selectors = make(map[string]Selector)

var (
	ErrServiceNotFound = fmt.Errorf("service not found")
)

// Selector loadbalance selector
type Selector interface {
	Select(context.Context, []registry.ServiceInstance) (registry.ServiceInstance, error)
}

// Initializer parse arguments
type Initializer interface {
	Init(ctx context.Context) error
}

type SelectorFunc func(context.Context, []registry.ServiceInstance) (registry.ServiceInstance, error)

// Select selector
func (sf SelectorFunc) Select(ctx context.Context, ins []registry.ServiceInstance) (registry.ServiceInstance, error) {
	return sf(ctx, SwimLaneSelector(ctx, ins))
}

type ExtractConf struct {
	Registry        string
	Namespace       string
	Group           string
	ServiceName     string
	MethodName      string
	SelectKeys      []string
	LoadBalanceMeta string
}

type Extract interface {
	Extract(*ExtractConf) (interface{}, error)
}

type Inject interface {
	Inject(ctx context.Context, metas interface{}) (context.Context, error)
}

type Setter interface {
	SetDiscovery(func(*common.URL) []registry.ServiceInstance)
}

// Register register
func Register(name string, selector Selector) {
	selectors[name] = selector
}

// GetSelector get selector
func GetSelector(ctx context.Context, name string) (lb Selector) {
	lb, ok := selectors[name]
	if !ok {
		return selectors[constant.SelectorRandom]
	}

	if s, ok := lb.(Initializer); ok {
		_ = s.Init(ctx)
	}

	return lb
}

// SwimLaneSelector swimlane selector
func SwimLaneSelector(ctx context.Context, ins []registry.ServiceInstance) []registry.ServiceInstance {
	if env.IsProduction() {
		return ins
	}
	if len(ins) == 0 {
		return nil
	}
	md := metadata.MDFromContext(ctx)

	validIns := make([]registry.ServiceInstance, 0, len(ins))
	envIns := make([]registry.ServiceInstance, 0)
	baseIns := make([]registry.ServiceInstance, 0)
	for _, in := range ins {
		swim := in.GetMetadata().GetSwimLane(md.Intermediate.LaneEnv)
		switch {
		case swim.IsEnvLane():
			envIns = append(envIns, in)
		case swim.IsBaseLane():
			baseIns = append(baseIns, in)
		default:
			validIns = append(validIns, in)
		}
	}

	if md.Intermediate.LaneEnv != "" {
		if len(envIns) > 0 {
			glog.Debug(ctx, "swim lane env hit", glog.Int64("len", int64(len(envIns))), glog.String("env", md.Intermediate.LaneEnv))
			return envIns
		}
	}
	if len(baseIns) > 0 {
		glog.Debug(ctx, "swim lane base hit", glog.Int64("len", int64(len(baseIns))), glog.String("env", md.Intermediate.LaneEnv))
		return baseIns
	}
	if len(validIns) > 0 {
		glog.Debug(ctx, "swim lane validIns hit", glog.Int64("len", int64(len(validIns))), glog.String("env", md.Intermediate.LaneEnv))
		return validIns
	}
	if md.Intermediate.LaneEnv == "" {
		if len(envIns) > 0 {
			glog.Debug(ctx, "swim lane env hit", glog.Int64("len", int64(len(envIns))), glog.String("env", md.Intermediate.LaneEnv))
			return envIns
		}
	}

	glog.Debug(ctx, "swim lane default hit", glog.Int64("len", int64(len(ins))), glog.String("env", md.Intermediate.LaneEnv))
	return ins
}

func LocalAware(ins []registry.ServiceInstance) []registry.ServiceInstance {
	return azAware(cloudAware(ins))
}

// azAware local az aware
func azAware(ins []registry.ServiceInstance) []registry.ServiceInstance {
	if len(ins) == 0 {
		return nil
	}

	azIns := make([]registry.ServiceInstance, 0)
	for _, in := range ins {
		az := in.GetMetadata().GetAZName()
		if az != "" && az == env.AvailableZoneID() {
			azIns = append(azIns, in)
		}
	}
	if len(azIns) > 0 {
		return azIns
	}
	return ins
}

var (
	support   = env.IsSupportCloud()
	cloudName = env.CloudProvider()
)

// cloudAware local cloud aware
func cloudAware(ins []registry.ServiceInstance) []registry.ServiceInstance {
	if len(ins) == 0 {
		return nil
	}
	if !support {
		return ins
	}
	if cloudName == "" {
		return ins
	}

	res := make([]registry.ServiceInstance, 0)
	degradeRes := make([]registry.ServiceInstance, 0)
	degradeMap, _ := GetDegradeCloudMap()
	for _, in := range ins {
		cn := getCluster(in.GetCluster())
		if cn == cloudName {
			res = append(res, in)
		}

		if _, ok := degradeMap[cn]; ok {
			degradeRes = append(degradeRes, in)
		}
	}

	if len(res) > 0 {
		return res
	}

	if len(degradeRes) > 0 {
		return degradeRes
	}

	return ins
}

const (
	AwsClusterName     = "aws"
	DefaultClusterName = "DEFAULT"
)

func getCluster(c string) string {
	if c == DefaultClusterName {
		return AwsClusterName
	}
	return c
}
