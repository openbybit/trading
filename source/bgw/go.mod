module bgw

go 1.19

require (
	code.bydev.io/cht/backend-bj/user-service/buf-user-gen.git/pkg v0.0.0-20230720082312-3edd3552bbe7
	code.bydev.io/cht/customer/kyc-stub.git/pkg v0.0.0-20230607082547-d6be3194d16f
	code.bydev.io/fbu/future/api/openapigen.git/pkg v0.0.0-20221205062513-39965b3d2b0a
	code.bydev.io/fbu/future/bufgen.git/pkg v0.0.0-20230519072535-d910887cc0ac
	code.bydev.io/fbu/future/sdk.git/pkg/future v0.0.0-20230411080013-40368194d7ef
	code.bydev.io/fbu/future/sdk.git/pkg/scmeta v0.0.0-20230309073054-6e1201fd0a21
	code.bydev.io/fbu/future/sdk.git/pkg/scmeta/scsrc v0.0.0-20231207104158-a2c995794b54
	code.bydev.io/fbu/gateway/gway.git/galert v0.0.0-20230817092925-56a9709a2f18
	code.bydev.io/fbu/gateway/gway.git/gapp v0.0.0-20230731055551-bb3fd770109c
	code.bydev.io/fbu/gateway/gway.git/gbsp v0.0.0-20231128023959-8fd304faef8e
	code.bydev.io/fbu/gateway/gway.git/gcompliance v0.0.0-20231026032004-8e70e9b96f0d
	code.bydev.io/fbu/gateway/gway.git/gconfig v0.0.0-20230831082700-3eb14baafa5c
	code.bydev.io/fbu/gateway/gway.git/gcore v0.0.0-20230828025047-9bf39f142e06
	code.bydev.io/fbu/gateway/gway.git/generic v0.0.0-20230306021408-7a0c92c51f77
	code.bydev.io/fbu/gateway/gway.git/geo v0.0.0-20230830102241-d4132958b45f
	code.bydev.io/fbu/gateway/gway.git/getcd v0.0.0-20230303030441-be688745d971
	code.bydev.io/fbu/gateway/gway.git/ggrpc v0.0.0-20230831082700-3eb14baafa5c
	code.bydev.io/fbu/gateway/gway.git/ghttp v0.0.0-20230303030441-be688745d971
	code.bydev.io/fbu/gateway/gway.git/gkafka v0.0.0-20230828025047-9bf39f142e06
	code.bydev.io/fbu/gateway/gway.git/glog v0.0.0-20230814023225-b0fc4158cf12
	code.bydev.io/fbu/gateway/gway.git/gmetric v0.0.0-20231121070751-d82add688d0c
	code.bydev.io/fbu/gateway/gway.git/gnacos v0.0.0-20230522080054-c2a2f489e112
	code.bydev.io/fbu/gateway/gway.git/gopeninterest v0.0.0-20230911075128-4789db4af785
	code.bydev.io/fbu/gateway/gway.git/gredis v0.0.0-20230612094657-098e98c9f6c8
	code.bydev.io/fbu/gateway/gway.git/groute v0.0.0-20230519022656-988878b4d2f8
	code.bydev.io/fbu/gateway/gway.git/gs3 v0.0.0-20230303030441-be688745d971
	code.bydev.io/fbu/gateway/gway.git/gsechub v0.0.0-20230303030441-be688745d971
	code.bydev.io/fbu/gateway/gway.git/gsmp v0.0.0-20230328062353-aff05f17894d
	code.bydev.io/fbu/gateway/gway.git/gsymbol v0.0.0-20230519022656-988878b4d2f8
	code.bydev.io/fbu/gateway/gway.git/gtrace v0.0.0-20231023024234-05d49709da42
	code.bydev.io/fbu/gateway/proto.git/pkg v0.0.0-20231103075510-3557a550d528
	code.bydev.io/frameworks/byone v0.4.1
	code.bydev.io/frameworks/nacos-sdk-go/v2 v2.1.7
	code.bydev.io/frameworks/sarama v1.0.2
	code.bydev.io/lib/bat/core.git/pkg/bat v1.0.1
	code.bydev.io/lib/bat/core.git/pkg/blabel v1.0.3
	code.bydev.io/lib/bat/core.git/pkg/bstd v1.0.2
	code.bydev.io/lib/bat/solutions.git/pkg/bkfk v1.1.1-0.20220616122751-ebbc65c5c738
	code.bydev.io/public-lib/sec/sec-sign.git v0.0.3
	git.bybit.com/codesec/sechub-sdk-go v1.0.5
	git.bybit.com/gtd/gopkg/solutions/risksign.git v0.0.0-20220705141637-684416973f32
	git.bybit.com/gtdmicro/stub v0.0.0-20220505105334-02aef9643a6b
	git.bybit.com/svc/go/pkg/bconst v0.0.0-20220620041220-299600bec1d7
	git.bybit.com/svc/mod/pkg/bplatform v0.0.0-20220831072442-8748b3339d53
	git.bybit.com/svc/mod/pkg/bproto v0.0.0-20220624033201-0dd95d246844
	git.bybit.com/svc/stub/pkg/pb v0.0.0-20230914065439-8626dcead02e
	git.bybit.com/svc/stub/pkg/svc v0.0.0-20230529101905-d24f6e2b8301
	github.com/Workiva/go-datastructures v1.0.52
	github.com/agiledragon/gomonkey/v2 v2.10.1
	github.com/armon/go-radix v1.0.0
	github.com/buger/jsonparser v1.1.1
	github.com/coocood/freecache v1.2.3
	github.com/dustinxie/lockfree v0.0.0-20210712051436-ed0ed42fd0d6
	github.com/fasthttp/router v1.4.11
	github.com/fasthttp/websocket v1.5.1-rc.5
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.3
	github.com/hashicorp/go-version v1.6.0
	github.com/jhump/protoreflect v1.14.0
	github.com/jinzhu/copier v0.3.5
	github.com/json-iterator/go v1.1.12
	github.com/miekg/dns v1.0.14
	github.com/oliveagle/jsonpath v0.0.0-20180606110733-2e52cf6e6852
	github.com/opentracing/opentracing-go v1.2.1-0.20220228012449-10b1cf09e00b
	github.com/pkg/errors v0.9.1
	github.com/rs/xid v1.4.0
	github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/segmentio/encoding v0.3.6
	github.com/smartystreets/goconvey v1.8.1
	github.com/spf13/cobra v1.6.1
	github.com/spf13/viper v1.15.0
	github.com/stretchr/testify v1.8.4
	github.com/tj/assert v0.0.3
	github.com/uber/jaeger-client-go v2.30.0+incompatible
	github.com/valyala/bytebufferpool v1.0.0
	github.com/valyala/fasthttp v1.38.0
	github.com/valyala/fasttemplate v1.2.2
	go.etcd.io/etcd/client/v3 v3.5.7
	go.uber.org/atomic v1.10.0
	go.uber.org/automaxprocs v1.5.1
	golang.org/x/net v0.9.0
	golang.org/x/text v0.9.0
	golang.org/x/time v0.3.0
	google.golang.org/grpc v1.56.2
	google.golang.org/protobuf v1.30.0
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	code.bydev.io/fbu/future/sdk.git/pkg/sccalc v0.0.0-20230411080013-40368194d7ef // indirect
	code.bydev.io/frameworks/cmdh-go v1.1.12 // indirect
	code.bydev.io/frameworks/infra-go-sdk/pkg/erms v1.0.9 // indirect
	code.bydev.io/frameworks/otelsarama v1.0.1 // indirect
	code.bydev.io/frameworks/sechub-go v1.0.6 // indirect
	code.bydev.io/lib/bat/core.git/pkg/bdebug v1.0.0 // indirect
	code.bydev.io/lib/bat/core.git/pkg/bdig v1.0.0 // indirect
	code.bydev.io/lib/bat/core.git/pkg/blog v1.0.5 // indirect
	code.bydev.io/lib/bat/core.git/pkg/bobs v1.0.0 // indirect
	code.bydev.io/lib/bat/core.git/pkg/bstack v1.0.0 // indirect
	code.bydev.io/lib/gopkg/bdecimal.git v1.0.0 // indirect
	code.bydev.io/lib/gopkg/localvault.git v0.0.0-20220818104455-b6da9f3bf24a // indirect
	emperror.dev/errors v0.8.1 // indirect
	git.bybit.com/svc/go/pkg/bstd v0.0.0-20220407042734-4ea53f3ec20b // indirect
	github.com/Luzifer/go-openssl/v4 v4.1.0 // indirect
	github.com/Shopify/sarama v1.38.1 // indirect
	github.com/aliyun/alibaba-cloud-sdk-go v1.62.108 // indirect
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/aws/aws-sdk-go v1.44.109 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bytedance/godlp v1.2.15 // indirect
	github.com/cenkalti/backoff/v4 v4.2.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dchest/siphash v1.2.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/eapache/go-resiliency v1.3.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20230111030713-bf00bc1b83b6 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/frankban/quicktest v1.14.4 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-redis/redis/v8 v8.11.5 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.7.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.3 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/klauspost/compress v1.15.14 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/nacos-group/nacos-sdk-go v1.1.1 // indirect
	github.com/natefinch/atomic v1.0.1 // indirect
	github.com/openzipkin/zipkin-go v0.4.1 // indirect
	github.com/oschwald/geoip2-golang v1.8.0 // indirect
	github.com/oschwald/maxminddb-golang v1.10.0 // indirect
	github.com/pelletier/go-toml/v2 v2.0.6 // indirect
	github.com/petermattis/goid v0.0.0-20221215004737-a150e88a970d // indirect
	github.com/pierrec/lz4/v4 v4.1.17 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.14.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.41.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/savsgio/gotils v0.0.0-20220530130905-52f3993e8d6d // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/smarty/assertions v1.15.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.9.3 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.4.2 // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	github.com/willf/bitset v1.1.11 // indirect
	github.com/willf/bloom v2.0.3+incompatible // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	go.etcd.io/etcd/api/v3 v3.5.7 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.7 // indirect
	go.opentelemetry.io/contrib/propagators/jaeger v1.12.0 // indirect
	go.opentelemetry.io/otel v1.19.0 // indirect
	go.opentelemetry.io/otel/exporters/jaeger v1.11.2 // indirect
	go.opentelemetry.io/otel/exporters/otlp/internal/retry v1.11.2 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.11.2 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.11.2 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.11.2 // indirect
	go.opentelemetry.io/otel/exporters/zipkin v1.11.2 // indirect
	go.opentelemetry.io/otel/sdk v1.11.2 // indirect
	go.opentelemetry.io/otel/trace v1.19.0 // indirect
	go.opentelemetry.io/proto/otlp v0.19.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	go.uber.org/zap v1.24.0 // indirect
	golang.org/x/crypto v0.5.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.1-0.20190411184413-94d9e492cc53 // indirect
)

replace (
	github.com/uber/jaeger-client-go => code.bydev.io/public-lib/infra/trace/jaeger-client-go.git v1.0.0
	go.opentelemetry.io/otel => go.opentelemetry.io/otel v1.14.0
	gopkg.in/natefinch/lumberjack.v2 => gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

exclude git.bybit.com/svc/stub/pkg/svc v0.0.0-20200717092246-a6ea7ad86f42
