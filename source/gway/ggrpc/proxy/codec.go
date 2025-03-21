package proxy

import (
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"
)

// Frame 用于中转流量,不解析具体内容
type Frame struct {
	Payload []byte
}

func NewCodec(codec encoding.Codec) encoding.Codec {
	if codec == nil {
		codec = encoding.GetCodec(proto.Name)
	}

	return &proxyCodec{codec: codec}
}

// proxyCodec 用于代理grpc流量,对内容不做解析,通过Frame结构体,直接流量透传
// 对于服务端:
// -- 使用grpc.ForceServerCodec(proxy.proxyCodec{})注册解码方式
// -- 使用grpc.UnknownServiceHandler(XXXStreamHandler)注册回调
// 对于客户端:
// -- 在Dial时,使用grpc.ForceCodec注册编解码,当调用Invoke方法时会使用proxy.proxyCodec来编解码
type proxyCodec struct {
	codec encoding.Codec
}

func (c *proxyCodec) Marshal(v interface{}) ([]byte, error) {
	out, ok := v.(*Frame)
	if !ok {
		return c.codec.Marshal(v)
	}

	return out.Payload, nil
}

func (c *proxyCodec) Unmarshal(data []byte, v interface{}) error {
	if dst, ok := v.(*Frame); ok {
		dst.Payload = data
		return nil
	}

	return c.codec.Unmarshal(data, v)
}

func (c *proxyCodec) String() string {
	return "proxy"
}

func (c *proxyCodec) Name() string {
	return "proxy"
}
