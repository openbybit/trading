# 业务网关

- [业务网关](#业务网关)
  - [在线文档](#在线文档)
  - [系统架构](#系统架构)
  - [流量处理过程概览](#流量处理过程概览)
  - [版本记录](#版本记录)
  - [核心模块结构](#核心模块结构)
  - [泛化调用](#泛化调用)
  - [接口错误码](#接口错误码)
  - [protobuf annotation](#protobuf-annotation)
  - [proto自动化部署过程](#proto自动化部署过程)
  - [access log](#access-log)
    - [demo](#demo)
    - [access log字段说明](#access-log字段说明)
  - [开发及测试服务器](#开发及测试服务器)
  - [s3 related](#s3-related)
  - [GeoIP](#geoip)
    - [install geoipupdate in amazon linux](#install-geoipupdate-in-amazon-linux)
    - [install geoipupdate in mac](#install-geoipupdate-in-mac)
    - [日志查询相关](#日志查询相关)
    - [监控面板metrics](#监控面板metrics)
    - [链路跟踪trace](#链路跟踪trace)

## 在线文档
> 我们在confluence记录了一些设计文档
[业务网关相关文档](https://confluence.yijin.io/pages/viewpage.action?pageId=31889656)

## 系统架构
![系统架构](doc/arc.png)
![系统架构](doc/system.png)

## 流量处理过程概览
![流量处理过程概览](doc/filter.png)

## 版本记录
[版本记录](https://confluence.bybit.com/pages/viewpage.action?pageId=31889691)

## 核心模块结构
```bash
├── README.md
├── app.toml
├── build.sh
├── cmd
│   ├── root.go
│   ├── server.go
│   └── testdata
│       ├── data
│       ├── geo_ip.go
│       ├── grpc.go
│       ├── hello.go
│       ├── hellozoneraft.go
│       ├── hellozonerr.go
│       ├── masq.go
│       └── user.go
├── config
│   ├── conf.d
│   │   └── hello-http.yaml
│   └── main.yaml
├── data
│   ├── cache
│   │   ├── nacos
│   │   └── s3
│   ├── log
│   │   ├── access
│   │   ├── access.log
│   │   └── nacos
│   └── proto
│       ├── bgw
│       └── hello
├── doc
│   ├── arc.png
│   ├── class.wsd
│   ├── pipeline
│   │   ├── lifecycle.png
│   ├── proto.png
│   └── system.png
├── go.mod
├── go.sum
├── internal
│   ├── app
│   │   └── app_bgw.go
│   └── pkg
│       └── services
├── main.go
├── package-lock.json
├── package.json
├── pkg
│   ├── cluster
│   │   ├── selector
│   │   └── selector.go
│   ├── common
│   │   ├── app.go
│   │   ├── berror
│   │   ├── cache
│   │   ├── config.go
│   │   ├── constant
│   │   ├── container
│   │   ├── extension
│   │   ├── http.go
│   │   ├── lru
│   │   ├── metadata.go
│   │   ├── node.go
│   │   ├── observer
│   │   ├── recover.go
│   │   ├── route_group.go
│   │   ├── types
│   │   ├── url.go
│   │   ├── url_test.go
│   │   └── util
│   ├── config
│   │   ├── filters_test.go
│   │   ├── metadata.go
│   │   ├── metadata_test.go
│   │   ├── remote.go
│   │   ├── remote_test.go
│   │   ├── response.go
│   │   ├── server.go
│   │   ├── service.go
│   │   ├── service_test.go
│   │   ├── version.go
│   │   └── version_test.go
│   ├── config_center
│   │   ├── config_center.go
│   │   ├── config_center_test.go
│   │   ├── etcd
│   │   ├── listener.go
│   │   └── nacos
│   ├── filter
│   │   ├── accesslog
│   │   ├── auth
│   │   ├── chain.go
│   │   ├── context
│   │   ├── cors
│   │   ├── filter.go
│   │   ├── filter_test.go
│   │   ├── geoip
│   │   ├── limiter
│   │   ├── metrics
│   │   ├── openapi
│   │   ├── plugin.go
│   │   ├── plugins
│   │   ├── redis_limiter_v2
│   │   ├── request
│   │   ├── response
│   │   └── trace
│   ├── http
│   │   ├── config_manager.go
│   │   ├── config_manager_test.go
│   │   ├── grpc_invoker.go
│   │   ├── router_manger.go
│   │   ├── router_manger_example_test.go
│   │   ├── router_manger_test.go
│   │   ├── service.go
│   │   ├── service_registry.go
│   │   ├── service_test.go
│   │   ├── version_control.go
│   │   ├── version_control_test.go
│   │   ├── web_console.go
│   │   └── web_console_test.go
│   ├── logger
│   │   ├── access_log.go
│   │   ├── access_log_test.go
│   │   ├── blog_logger.go
│   │   ├── data
│   │   ├── logger.go
│   │   ├── option.go
│   │   └── zap.go
│   ├── metrics
│   │   ├── http_metrics.go
│   │   └── prometheus
│   ├── plugin
│   │   └── plugin_example.go
│   ├── protocol
│   │   ├── grpc
│   │   ├── http
│   │   ├── protocol.go
│   │   ├── request.go
│   │   └── result.go
│   ├── registry
│   │   ├── dns
│   │   ├── etcd
│   │   ├── event.go
│   │   ├── listener.go
│   │   ├── metadata.go
│   │   ├── metadata_test.go
│   │   ├── nacos
│   │   ├── registry.go
│   │   └── service.go
│   ├── remoting
│   │   ├── alert
│   │   ├── etcd
│   │   ├── kafka
│   │   ├── listener.go
│   │   ├── nacos
│   │   ├── redis
│   │   ├── s3
│   │   └── user
│   └── tracing
│       ├── span.go
│       ├── tracer.go
│       └── tracer_test.go
├── run.sh
└── test.http
```

## 泛化调用
![泛化调用](doc/proto.png)

## 接口错误码
[错误码](https://confluence.bybit.com/pages/viewpage.action?pageId=36872618)
- 4001	service instantce not found	服务发现未找到服务实例
- 4002	filter not found	定义的filter未找到
- 4004	route not registered	路由不存在
- 4005  route registered, but routeKey invalid 路由配置错误
- 4006  filter param error      filter参数错误
- 5000	internal default error	系统默认服务端错误
- 5001	error when grpc connect failed	grpc调用连接失败
- 5002	generic upstream error	通用上游服务端异常
- 5003	grpc response status is not ok	grpc返回状态不正常，详细错误码：grpc错误码
- 5004	grpc response unknown error	grpc返回未知异常
- 6001	rate limit blocked	限流异常
- 6002  auth internal error    auth内部错误
- 6003  openapi internal error openapi内部错误
- 6004  biz rate limiter internal error  biz_rate_limiter内部错误
- 7000	extract proto descriptor error, serve or method not found	proto未找到服务或方法
- 7001	unmarshal json into proto message error	
- 7002	marshal proto message into json error

## protobuf annotation
> 我们设计了针对protobuf的插件，用于定义接口在业务网关要实现的业务规则，包括：负载均衡策略，各类过滤器组合，接口路由定义，服务发现定义等。
> protobuf插件通过git pipeline自动触发构建。

```proto
syntax = "proto3";
package module;

option go_package = "module";
import "bgw/v1/annotations.proto"; ## 插件引入，支持业务网关的annotation

message HelloMessage {
  string name = 1;
  int32 age = 2;
}

message HelloResult{
  string message = 1;
  string response_name = 2;
}

message HelloReply {
  int32 ret_code = 1;
  string ret_msg = 2;
  HelloResult result = 3;
}

service HelloService {
  option (bgw.v1.service_options) = {
    registry : "HelloService"; ## 服务发现，注册到nacos的服务名，默认public namespace，DEFAULT_GROUP
    filters:[
        {
          name:FILTER_AUTH;
          args:"--allowGuest=true";
        }
    ];
    selector: SELECTOR_ROUND_ROBIN; ## 服务级别的负载均衡策略
    protocol: PROTOCOL_GRPC;
    timeout: 5; ## 自定义调用超时，方法级别会覆盖service级别
  };

  rpc SayHello (HelloMessage) returns (HelloReply){
    option (bgw.v1.method_options) = {
      path: "/hello"; ## 路由url路径
      paths: [
        "/hello1",
        "/hello3",
        "/hello4",
      ]
      http_method: HTTP_METHOD_POST; ## http方法
      timeout: 5; ## 自定义调用超时
      filters:[
        {
          name: auth; ## 方法级别流量过滤器，这个接口使用鉴权
        }
      ];
    };
  };

  rpc SayHello2 (HelloMessage) returns (HelloReply){
    option (bgw.v1.method_options) = {
      path: "/hello2";
      http_method: HTTP_METHOD_POST;
    };
  };
}

```

## proto自动化部署过程
> 业务网关提供动态路由功能，即在不重启业务网关的情况下动态更新路由。
> 业务网关通过对proto文件的注解信息进行解析，生成路由规则。
proto注解使用的是protobuf的option语法。其元信息定义在annotations.proto文件中。
业务方在自己的proto仓库下先引入annotations.proto,然后利用其中的元信息进行注解,然后配置好pipepline的相关信息
提交tag触发pipeline将路由信息推到对应环境.

![pipeline 过程](doc/pipeline/lifecycle.png)

## access log
> 业务网关记录接口访问日志，与业务网关本身的日志记录分开存储，日志格式满足风控及标准功能需求。
[业务网关access log说明](https://confluence.bybit.com/pages/viewpage.action?pageId=31897371)
### demo
```
2021-09-17 07:48:37     INFO    BGW(1.0.0)      200     0       aa81cb77178b11ec806b0a20b4414e32        2.378621ms      gateway-infra-test.bybit.com    10.18.1.80      0       Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/93.0.4577.82 Safari/537.36        http://www.asset-test-1.bybit.com/      POST    /usdt/option/public/optionmp/query/config/global
```

### access log字段说明

| index | field             | comment              |
| ----- | ----------------- | -------------------- |
| 1     | 访问时间          | 访问时间             |
| 2     | 日志级别          | INFO/ERROR           |
| 3     | 业务网关          | BGW(1.0.0)           |
| 4     | http code         | 标准http code        |
| 5     | 业务异常码        | 业务网关定义的错误码 |
| 6     | trace id          | skywalking/jaeger    |
| 7     | latency           | 访问时延             |
| 8     | request host      | 访问域名             |
| 9     | remote ip         | 客户端ip             |
| 10    | member id         | 登录用户id           |
| 11    | account id        | 登录用户account_id   |
| 12    | userAgent         | http UA              |
| 13    | referer           | http referer         |
| 14    | http method       | GET/POST             |
| 15    | request path      | http path            |
| 16    | 设备指纹,platform | pcweb                |

## 开发及测试服务器
  | ID  | 主机名               | IP                    | 备注                                       |
  |-------------------|-----------------|------------------------------------------| ------------ |
  | 1   | fbu-test-1        | 动态pod ip              | [测试环境](http://api2.fbu-test-1.bybit.com) |
  | 2   | 其余环境名             | 动态pod ip              | [测试环境](http://api2.环境名.bybit.com)        |
  | 3  | test-gateway      | 10.110.54.155         | 开发服务器                                    |

## s3 related

## GeoIP

### install geoipupdate in amazon linux
resource: /usr/share/GeoIP/ ,  secret: /etc/GeoIP.conf
```bash
yum remove -y GeoIP
wget https://github.com/maxmind/geoipupdate/releases/download/v4.3.0/geoipupdate_4.3.0_linux_amd64.rpm
rpm -i geoipupdate_4.3.0_linux_amd64.rpm
geoipupdate
```
### install geoipupdate in mac
resource: /usr/local/var/GeoIP/ ,  secret: /usr/local/etc/GeoIP.conf
```bash
brew install geoipupdate
geoipupdate
```

### 日志查询相关
> 业务网关日志分为：接口访问日志、业务网关本身的日志，访问日志索引为bgw，业务日志索引为bgw-app。

以期权testnet日志为例：
+ [业务网关access log查询](https://logs-infra-ops.go.akamai-access.com/app/discover#/?_g=(filters:!(),refreshInterval:(pause:!t,value:0),time:(from:now-15m,to:now))&_a=(columns:!(),filters:!(('$state':(store:appState),meta:(alias:!n,disabled:!f,index:fbdc34b0-2fed-11ec-94e7-efb6f86f6491,key:app.keyword,negate:!f,params:(query:bgw),type:phrase),query:(match_phrase:(app.keyword:bgw))),('$state':(store:appState),meta:(alias:!n,disabled:!f,index:fbdc34b0-2fed-11ec-94e7-efb6f86f6491,key:env.keyword,negate:!f,params:(query:testnet),type:phrase),query:(match_phrase:(env.keyword:testnet)))),index:fbdc34b0-2fed-11ec-94e7-efb6f86f6491,interval:auto,query:(language:kuery,query:''),sort:!(!('@timestamp',desc))))
+ [业务网关业务日志查询](https://logs-infra-ops.go.akamai-access.com/app/discover#/?_g=(filters:!(),refreshInterval:(pause:!t,value:0),time:(from:now-15m,to:now))&_a=(columns:!(),filters:!(('$state':(store:appState),meta:(alias:!n,disabled:!f,index:fbdc34b0-2fed-11ec-94e7-efb6f86f6491,key:app.keyword,negate:!f,params:(query:bgw-app),type:phrase),query:(match_phrase:(app.keyword:bgw-app))),('$state':(store:appState),meta:(alias:!n,disabled:!f,index:fbdc34b0-2fed-11ec-94e7-efb6f86f6491,key:env.keyword,negate:!f,params:(query:testnet),type:phrase),query:(match_phrase:(env.keyword:testnet)))),index:fbdc34b0-2fed-11ec-94e7-efb6f86f6491,interval:auto,query:(language:kuery,query:''),sort:!(!('@timestamp',desc))))
+ [kibana登录方式](https://confluence.yijin.io/pages/viewpage.action?pageId=36873834)

各个业务线日志索引前缀：
+ 期权日志前缀：yj_option_*      
  + accesslog日志标签为bgw，业务日志标签为bgw-app
+ 法币入金日志查询：yj_rd_fbu_*   
  + accesslog日志标签为bgw，业务日志标签为bgw-app
+ 用户服务bgw日志前缀：yj_rd_cht_*  
  + accesslog日志标签为bgw-user，业务日志标签为bgw-user
+ 中台bgw日志前缀：yi_rd_cht_*     
  + accesslog日志标签为bgw，业务日志标签为bgw
+ 期货site-bgw日志前缀：yj_rd_fbu_*  
  + accesslog日志标签为fbu-site-bgw，业务日志标签为fbu-site-bgw
+ 期货openapi-bgw日志前缀：yj_rd_fbu_* 
  + accesslog日志标签为fbu-openapi-bgw，业务日志标签为fbu-openapi-bgw

### 监控面板metrics
+ [期权](https://owl-grafana-infra-ops.go.akamai-access.com/d/thYGIJKGz/bgw-option-testnet?orgId=3&from=now-6h&to=now)
+ [效能平台测试环境](http://grafana.bdot-devbase.bybit.com/d/thYGIJKGz/bgw-bdot?orgId=1&refresh=30s&from=now-6h&to=now)
+ [法币入金](https://owl-grafana-infra-ops.go.akamai-access.com/d/thYGIJKGzz/bgw-fiat-mainnet?orgId=9)
+ [用户服务/中台](https://owl-grafana-infra-ops.go.akamai-access.com/d/XdswYlE7z/bgw-prod?orgId=9)
+ [期货site/openapi](https://owl-grafana-infra-ops.go.akamai-access.com/d/thYGIJKssGz/bgw-fbu?orgId=14&refresh=1m)

### 链路跟踪trace
+ [原生jaeger](https://jaeger-infra-ops.go.akamai-access.com/search?end=1641456423282000&limit=20&lookback=6h&maxDuration&minDuration&service=prod-bgw-BGW&start=1641434823282000)
+ [效能平台](https://rd-by.go.akamai-access.com/monitor-and-alarm/monitor/trace)

  各个业务线服务名:
  + testnet为testnet-bgw-BGW，prod为prod-bgw-BGW
