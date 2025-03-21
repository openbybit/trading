package generic

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	protoSetMode     = iota // 0, default mode, protoset of proto
	protoFilesMode          // 1, for user specified upload proto file
	protoReflectMode        // 2, for dev & test only, grpc reflect server
)

// controller descriptors manager
type controller struct {
	descMode   int
	descriptor Descriptor
	// marshal a protobuf message into JSON
	codec Codec
	// cache proto types
	cache *protoTypesCache

	protoFiles  multiString // source
	importPaths multiString // source import path
}

// newController new desc controller
func newController() *controller {
	desc := &controller{
		descMode: protoSetMode,
		cache:    newProtoTypeCache(),
	}

	return desc
}

func (ct *controller) setProtoFiles(importPath string, protoFile string) {
	ct.importPaths.Set(importPath)
	ct.protoFiles.Set(protoFile)
}

func (ct *controller) setMode(mode int) {
	ct.descMode = mode
}

// init init descriptors by mode
func (ct *controller) init(data []byte) (err error) {
	var desc Descriptor

	switch ct.descMode {
	case protoSetMode:
		desc, err = DescriptorFromProtoSetReader(bytes.NewBuffer(data))
		if err != nil {
			return err
		}
		ct.descriptor = desc
	case protoFilesMode:
		ct.descriptor, err = DescriptorFromProtoFiles(
			ct.importPaths,
			ct.protoFiles...,
		)
		if err != nil {
			return err
		}
	default:
		return errInvalidMode
	}

	return ct.initCodec()
}

func (ct *controller) initCodec() error {
	jc, err := newJsonpbCodec(ct.descriptor)
	if err != nil {
		return err
	}

	ct.codec = jc
	return nil
}

// extractProtoType extract protoTypeEntry from descriptor by service & method
// check cache first
func (ct *controller) extractProtoType(service, method string) (*protoTypeEntry, error) {
	key := ct.cache.makeKey(service, method)
	cached, ok := ct.cache.get(key)
	if ok { // hit cache
		return cached, nil
	}

	dsc, err := ct.descriptor.FindSymbol(service)
	if err != nil {
		if isNotFoundError(err) {
			return nil, errors.New("not find service in pb descriptor")
		}

		return nil, errors.New("query service failed in pb descriptor")
	}

	sd, ok := dsc.(*desc.ServiceDescriptor)
	if !ok {
		return nil, errors.New("not expose service")
	}

	md := sd.FindMethodByName(method)
	if md == nil {
		return nil, fmt.Errorf("service %q does not include a method named %q", service, method)
	}

	var ext = new(dynamic.ExtensionRegistry)
	alreadyFetched := map[string]bool{}
	if err = fetchAllExtensions(ct.descriptor, ext, md.GetInputType(), alreadyFetched); err != nil {
		return nil, fmt.Errorf("error resolving server extensions for message %s: %w", md.GetInputType().GetFullyQualifiedName(), err)
	}

	if err = fetchAllExtensions(ct.descriptor, ext, md.GetOutputType(), alreadyFetched); err != nil {
		return nil, fmt.Errorf("error resolving server extensions for message %s: %w", md.GetOutputType().GetFullyQualifiedName(), err)
	}

	// set types to cache
	typeEntry := &protoTypeEntry{
		md:  md,
		sd:  sd,
		ext: ext,
	}
	ct.cache.set(key, typeEntry)
	return typeEntry, nil
}

// invokeUnary uses the given gRPC channel to invokeUnary the given method.
func (ct *controller) invokeUnary(ctx context.Context, ch grpcdynamic.Channel, request Request, result Result) (err error) {
	protoType, err := ct.extractProtoType(request.GetService(), request.GetMethod())
	if err != nil || protoType.md == nil {
		return fmt.Errorf("extract proto type failed: %w", err)
	}

	factory := dynamic.NewMessageFactoryWithExtensionRegistry(protoType.ext)
	msg := factory.NewMessage(protoType.md.GetInputType())
	stub := grpcdynamic.NewStubWithMessageFactory(ch, factory)

	md := protoType.md
	err = ct.codec.Unmarshal(request.PayLoad(), msg)
	if err != nil && err != io.EOF {
		return fmt.Errorf("%w, request body unmarshal error, field name: %s, err:%s", ErrReqUnmarshalFailed, md.GetFullyQualifiedName(), err.Error())
	}

	var headers metadata.MD
	defer func() {
		result.SetMetadata(headers)
	}()
	resp, err := stub.InvokeRpc(ctx, md, msg, grpc.Header(&headers))
	if err != nil {
		return err
	}

	result.SetMessage(resp)
	return
}

func checksum(data []byte) string {
	h := md5.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
