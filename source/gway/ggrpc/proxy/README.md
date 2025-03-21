# proxy

grpc流量代理,需要自定义Codec,不解析数据仅转发流量

- 服务端
  - grpc.ForceServerCodec用于注册自定义编解码协议
  - grpc.UnknownServiceHandler用于接收所有未实现的接口
- 客户端
  - grpc.ForceCodec方法可以在Dial时指定编解码协议
