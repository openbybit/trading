package diagnosis

import (
	"context"
	"time"

	"code.bydev.io/frameworks/byone/kafka"
	"code.bydev.io/frameworks/byone/zrpc"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"bgw/pkg/common"
	"bgw/pkg/common/constant"
	"bgw/pkg/config_center/etcd"
	"bgw/pkg/discovery"
	"bgw/pkg/registry"
	"bgw/pkg/remoting/redis"
)

type Result struct {
	Es   []error  `json:"-"`
	Errs []string `json:"errs"`
	Cost string   `json:"cost"`
}

func NewResult(es ...error) Result {
	return Result{Es: es}
}

func DiagnoseGrpcUpstream(ctx context.Context, reg string, namespace string, group string) Result {
	return apply(func() Result {
		ins, err := getAllInstance(ctx, reg, namespace, group)
		if err != nil {
			return NewResult(err)
		}
		return NewResult(diagnosisInstances(ctx, reg, ins)...)
	})
}

func DiagnoseGrpcDependency(ctx context.Context, grpcCfg zrpc.RpcClientConf) Result {
	return apply(func() Result {
		if !grpcCfg.HasNacos() {
			err := Dial(ctx, grpcCfg.Target)
			if err != nil {
				return NewResult(err)
			}
			return NewResult()
		}
		ins, err := getAllInstance(ctx, grpcCfg.Nacos.Key, grpcCfg.Nacos.NamespaceId, grpcCfg.Nacos.Group)
		if err != nil {
			return NewResult(err)
		}
		return NewResult(diagnosisInstances(ctx, grpcCfg.Nacos.Key, ins)...)
	})
}

func DiagnoseKafka(_ context.Context, topic string, config kafka.UniversalClientConfig) Result {
	return apply(func() Result {
		client, err := kafka.NewClient(config)
		if err != nil {
			return NewResult(err)
		}
		defer func(client kafka.Client) {
			_ = client.Close()
		}(client)

		partitions, err := client.Partitions(topic)
		if err != nil {
			return NewResult(err)
		}

		consumer, err := client.NewConsumer()
		if err != nil {
			return NewResult(err)
		}
		var ee []error
		for _, partition := range partitions {
			_, err = consumer.GetNewestOffset(topic, partition)
			if err != nil {
				ee = append(ee, err)
			}
		}
		return NewResult(ee...)
	})
}

func DiagnoseEtcd(c context.Context) Result {
	return apply(func() Result {
		// ctx, cf := context.WithTimeout(c, time.Second*1)
		// defer cf()
		ctx := context.Background()
		ed, err := etcd.NewEtcdConfigure(ctx)
		if err != nil {
			return NewResult(err)
		}
		_, err = ed.Get(ctx, "xxx")
		if err != nil && !errors.Is(err, etcd.ErrKVPairNotFound) {
			return NewResult(err)
		}
		return NewResult()
	})
}

func DiagnoseRedis(ctx context.Context) Result {
	return apply(func() Result {
		c, cf := context.WithTimeout(ctx, time.Second*1)
		defer cf()
		r := redis.NewClient()
		if r == nil {
			return NewResult(errors.New("redis init failed"))
		}
		_, err := r.GetCtx(c, "xxx")
		if err != nil {
			return NewResult(errors.WithMessage(err, r.Addr))
		}
		return NewResult()
	})
}

func diagnosisInstances(ctx context.Context, reg string, instances []registry.ServiceInstance) []error {
	if len(instances) == 0 {
		return []error{errors.New(reg + ": instance not found")}
	}
	var errs []error
	for _, instance := range instances {
		addr := instance.GetAddress(constant.GrpcProtocol)
		err := Dial(ctx, addr)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func Dial(ctx context.Context, target string) error {
	ctx, cf := context.WithTimeout(ctx, time.Second*3)
	defer cf()
	c, e := grpc.DialContext(ctx, target, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if e != nil {
		return errors.WithMessage(e, target)
	}
	return c.Close()
}

func getAllInstance(ctx context.Context, reg string, na string, group string) ([]registry.ServiceInstance, error) {
	url, err := common.NewURL(reg,
		common.WithProtocol(constant.NacosProtocol),
		common.WithGroup(group),
		common.WithNamespace(na),
	)
	if err != nil {
		return nil, err
	}
	sr := discovery.NewServiceRegistry(ctx)
	ins := sr.GetInstancesNoCache(url)
	return ins, nil
}

func diagnoseWithTimeout(d Diagnosis) (interface{}, error) {
	c, cf := context.WithTimeout(context.Background(), time.Second*5)
	defer cf()
	return d.Diagnose(c)
}

func apply(f func() Result) Result {
	t := time.Now()
	r := f()
	r.Cost = time.Since(t).String()
	for _, err := range r.Es {
		r.Errs = append(r.Errs, err.Error())
	}
	return r
}
