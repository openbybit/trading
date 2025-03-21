package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	gencoding "code.bydev.io/fbu/gateway/gway.git/gcore/encoding"
	"code.bydev.io/fbu/gateway/gway.git/gcore/filesystem"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/gcore/recovery"
	"code.bydev.io/fbu/gateway/gway.git/generic"
	"code.bydev.io/fbu/gateway/gway.git/ggrpc/pool"
	"code.bydev.io/fbu/gateway/gway.git/ghttp"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"code.bydev.io/fbu/gateway/gway.git/gs3"
	"code.bydev.io/fbu/gateway/gway.git/gsechub"
	"code.bydev.io/fbu/gateway/gway.git/gtrace"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/config"
	pgrpc "bgw/pkg/server/core/grpc"
	phttp "bgw/pkg/server/core/http"
	gmetadata "bgw/pkg/server/metadata"
	"bgw/pkg/service"
)

// Invoker is a invoker function
type Invoker func(ctx context.Context, addr string, request Request, result Result) error

// entry file of descriptor
type entry struct {
	Key  string
	Data []byte
}

// newInvoker create a new invoker
func newInvoker(ctx context.Context) *invoker {
	iv := &invoker{
		ctx:      ctx,
		entryCh:  make(chan *entry, 1),
		midwares: getInvokeMidwares(),
	}

	iv.invoke = iv.baseInvoke
	for i := len(iv.midwares) - 1; i >= 0; i-- {
		iv.invoke = iv.midwares[i].Do(iv.invoke)
	}

	return iv
}

type invoker struct {
	ctx context.Context

	// session of aws s3
	ss gs3.Client

	// s3 remote s3 config
	sc *config.RemoteConfig

	// grpcEngine is grpc invoker
	grpcEngine *generic.Engine

	// httpEngine is http invoker
	httpEngine *ghttp.Invoker

	// save file chan
	entryCh chan *entry

	midwares []invokeMidware

	invoke invokeFunc
}

// init s3 session, grpc engine and descriptor
func (iv *invoker) init() error {
	session, err := newS3Session()
	if err != nil {
		return err
	}

	iv.ss = session
	cfg := &config.Global.S3
	if cfg == nil {
		panic("invalid s3 remote config")
	}

	iv.sc = cfg
	iv.grpcEngine = generic.NewEngine()
	iv.httpEngine = ghttp.GetInvoker()
	iv.startDescriptorWriter()

	return nil
}

func newS3Session() (gs3.Client, error) {
	rc := &config.Global.S3
	id := rc.GetOptions("id", "")
	secret := rc.GetOptions("secret", "")
	bucket := rc.GetOptions("bucket", "bgw")
	if id == "" || secret == "" {
		return nil, fmt.Errorf("aws s3's id or secret is nil")
	}
	// decrypt s3 secret
	passwd, err := gsechub.Decrypt(secret)
	if err != nil {
		return nil, fmt.Errorf("s3 secret Decrypt error, %w", err)
	}
	sess, err := gs3.NewClient(id, passwd, rc.Address, gs3.WithBucket(bucket))
	if err != nil {
		return nil, err
	}

	return sess, nil
}

func (iv *invoker) getCacheDir() string {
	return iv.sc.GetOptions("cache", "data/cache")
}

// startDescriptorWriter start write descriptor to local data
func (iv *invoker) startDescriptorWriter() {
	recovery.Go(func() {
		for {
			select {
			case <-iv.ctx.Done():
				glog.Info(iv.ctx, "local file saveLocalDescriptor quit")
				return
			case en := <-iv.entryCh:
				if en == nil || en.Key == "" || len(en.Data) == 0 {
					continue
				}

				// save to local file
				file := filepath.Join(iv.getCacheDir(), strings.ReplaceAll(en.Key, ".", "/"))
				err := filesystem.WriteFile(file, en.Data)
				if err != nil {
					glog.Error(iv.ctx, "write file error", glog.String("file", file), glog.String("error", err.Error()))
					continue
				}

				glog.Debug(iv.ctx, "save local file descriptor success", glog.String("file", file), glog.Int64("len", int64(len(en.Data))), glog.String("md5", gencoding.MD5Hex(en.Data)))
			}
		}
	}, recov)
}

func recov(r interface{}) {
	glog.Error(context.Background(), "save local descriptor error", glog.Any("err", r))
	if e, ok := r.(error); ok {
		msg := fmt.Sprintf("save local descriptor error, err = %s", e.Error())
		galert.Error(context.Background(), msg)
	}
}

// getLocalDescriptor check local cache if not-modified since last version
func (iv *invoker) getLocalDescriptor(version *AppVersion) []byte {
	file := filepath.Join(iv.getCacheDir(), strings.ReplaceAll(version.Key(), ".", "/"))
	fd, err := filesystem.OpenFile(file)
	if err != nil {
		glog.Error(iv.ctx, "open file error", glog.String("file", file))
		return nil
	}

	defer func() {
		if err := fd.Close(); err != nil {
			glog.Error(iv.ctx, "close file error", glog.String("file", file), glog.String("error", err.Error()))
		}
	}()

	data, err := io.ReadAll(fd)
	if err != nil || data == nil {
		return nil
	}

	checksum := gencoding.MD5Hex(data)
	vd := version.GetDescVersionEntry()
	if vd == nil {
		return nil
	}

	if vd.Checksum == checksum {
		return data
	}

	return nil
}

// OnEvent fired on version changed
func (iv *invoker) OnEvent(event observer.Event) (err error) {
	ve, ok := event.(*versionChangeEvent)
	if !ok {
		return nil
	}

	version := ve.GetSource().(*AppVersion)
	namespace := version.Key()
	glog.Info(iv.ctx, "fire invoker event", glog.String("event", namespace))

	key := version.GetS3Key()
	if key == "" {
		glog.Info(iv.ctx, "invalid version key", glog.String("event", namespace))
		return nil
	}

	// check local cached file is valid
	data := iv.getLocalDescriptor(version)
	if data == nil {
		glog.Debug(iv.ctx, "local cache missing, download from remote", glog.String("key", key), glog.String("etcd md5", version.GetDescChecksum()))
		ctx, cancel := context.WithTimeout(iv.ctx, 3*time.Second)
		defer cancel()

		data, err = iv.ss.Download(ctx, key, time.Time{})
		if err != nil {
			// alert
			msg := fmt.Sprintf("download descriptor from s3 error, err = %s, s3-key = %s", err.Error(), key)
			galert.Error(iv.ctx, msg)
			return err
		}
		glog.Debug(iv.ctx, "download protoset data ok", glog.String("key", key), glog.Int64("len", int64(len(data))), glog.String("md5", gencoding.MD5Hex(data)))
	}

	glog.Debug(iv.ctx, "do update protoset data", glog.String("key", key), glog.Int64("len", int64(len(data))), glog.String("md5", gencoding.MD5Hex(data)))
	cached, err := iv.grpcEngine.Update(namespace, data)
	if err != nil {
		msg := fmt.Sprintf("grpc Update engine error, err = %s, s3-key = %s", err.Error(), key)
		galert.Error(iv.ctx, msg)
		return
	}

	// save local cache
	if !cached {
		iv.entryCh <- &entry{
			Key:  namespace,
			Data: data,
		}
	}

	return
}

// GetEventType event of versionChangeEvent
func (iv *invoker) GetEventType() reflect.Type {
	return reflect.TypeOf(versionChangeEvent{})
}

// GetPriority before config manager, less prority value
func (iv *invoker) GetPriority() int {
	return 0
}

// getInvoker get invoker by protocol
func (iv *invoker) baseInvoke(ctx *types.Ctx, route *MethodConfig, md *gmetadata.Metadata) (err error) {
	var (
		invoke  Invoker
		request Request
		result  Result
	)

	switch route.Service().Protocol {
	case constant.HttpProtocol:
		request = phttp.NewRequest(ctx)
		invoke = iv.invokeHTTP
		result = phttp.NewResult()
	default:
		request = pgrpc.NewRPCRequest(
			ctx,
			route.Service().Key(),
			route.Service().GetFullQulifiedName(),
			route.Name,
		)
		invoke = iv.invokeGRPC
		result = pgrpc.NewResult()
	}

	tc, cancel := context.WithTimeout(service.GetContext(ctx), route.GetTimeout())
	defer cancel()

	md.ReqTime = time.Now()
	service.DynamicLog(ctx, "invoke request", glog.Any("request", request), glog.Duration("timeout", route.GetTimeout()))

	defer func() {
		md.ReqCost = time.Since(md.ReqTime)
		service.DynamicLog(ctx, "invoke response", glog.Any("response", result))
		if err != nil {
			fields := []glog.Field{glog.String("error", err.Error()),
				glog.String("addr", md.InvokeAddr),
				glog.String("content-type", string(ctx.Request.Header.ContentType())),
				glog.String("query", cast.UnsafeBytesToString(ctx.URI().QueryString())),
			}
			if len(ctx.Request.Body()) > 1024 {
				fields = append(fields, glog.String("body", cast.UnsafeBytesToString(ctx.Request.Body()[:1024])))
			} else {
				fields = append(fields, glog.String("body", cast.UnsafeBytesToString(ctx.Request.Body())))
			}
			glog.Info(ctx, "invoke error", fields...)
		}
	}()

	if err = invoke(tc, md.InvokeAddr, request, result); err != nil {
		return
	}

	// set invoke result into context
	ctx.SetUserValue(constant.CtxInvokeResult, result)
	return
}

// invokeGRPC invoke grpc service by grpc engine
func (iv *invoker) invokeGRPC(c context.Context, addr string, request Request, result Result) (err error) {
	conn, err := pool.GetConn(c, addr)
	if err != nil {
		return berror.NewUpStreamErr(berror.UpstreamErrInstanceConnFailed, "invokeGRPC error", addr, err.Error())
	}
	defer func() { _ = conn.Close() }()

	md := request.GetMetadata()
	ctx := metadata.NewOutgoingContext(c, md)
	err = iv.grpcEngine.Invoke(ctx, conn.Client(), request, result)

	gmd := result.Metadata()
	if v := gmd.Get(constant.BgwAPIResponseStatusCode); len(v) > 0 && v[0] != "" {
		if statusCode := cast.ToInt(v[0]); statusCode > 0 {
			result.SetStatus(statusCode)
		}
	}

	if err != nil {
		if errors.Is(err, generic.ErrReqUnmarshalFailed) {
			glog.Info(ctx, "request unmarshal to pb message error", glog.String("error", err.Error()))
			return berror.ErrInvalidRequest
		}

		stat, _ := status.FromError(err)
		if stat != nil {
			code := stat.Code()
			if code > codes.Unauthenticated {
				gmd.Set(constant.BgwAPIResponseCodes, cast.Int64toa(int64(stat.Code())))
				gmd.Set(constant.BgwAPIResponseMessages, stat.Message())
				return nil
			}
			switch code {
			case codes.DeadlineExceeded:
				return berror.ErrTimeout
			case codes.Internal, codes.Unavailable, codes.DataLoss, codes.Unimplemented, codes.ResourceExhausted:
				return berror.NewUpStreamErr(berror.UpstreamErrInvokerBreaker, addr, err.Error())
			}
		}
		return berror.NewUpStreamErr(berror.UpstreamErrInvokerFailed, addr, err.Error())
	}

	return
}

// invokeHTTP invoke http request by http client
func (iv *invoker) invokeHTTP(ctx context.Context, addr string, request Request, result Result) (err error) {
	span, _ := gtrace.Begin(
		ctx,
		fmt.Sprintf("http-invoke:%s-%s", request.GetMethod(), request.GetService()),
		opentracing.Tags{"addr": addr},
	)
	defer gtrace.Finish(span)

	request.SetMetadata("uber-trace-id", gtrace.UberTraceIDFromSpan(span))
	request.SetMetadata("traceparent", gtrace.TraceparentFromSpan(span))
	if err = iv.httpEngine.Invoke(ctx, addr, request, result); err == nil {
		return
	}

	switch {
	case errors.Is(err, ghttp.ErrReqBuildFailed):
		glog.Info(ctx, "request build error", glog.String("error", err.Error()))
		err = berror.ErrInvalidRequest
	case errors.Is(err, context.DeadlineExceeded):
		err = berror.ErrTimeout
	default:
		err = berror.NewUpStreamErr(berror.UpstreamErrInvokerBreaker, "invokeHTTP error", addr, err.Error())
	}

	return
}
