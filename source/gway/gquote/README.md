# gquote

IndexPrice query service

## 实现机制

- 服务启动时,调用grpc接口获取全量数据
- 运行过程中,消费rocketmq/hdts获取增量数据
- 期权需要定期清理过期的symbol

## 相关文档

- [行情指数服务](https://uponly.larksuite.com/wiki/wikushDigxqUx0D5bQRf4ThuYhI)
