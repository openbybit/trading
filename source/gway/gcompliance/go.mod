module code.bydev.io/fbu/gateway/gway.git/gcompliance

go 1.18

require (
	code.bydev.io/cht/customer/kyc-stub.git/pkg v0.0.0-20230607082547-d6be3194d16f
	code.bydev.io/fbu/gateway/gway.git/gcore v0.0.0-20230307093621-e4a2bb9d72c2
	code.bydev.io/fbu/gateway/gway.git/ggrpc v0.0.0-20230404064851-478080f88224
	code.bydev.io/fbu/gateway/gway.git/gmetric v0.0.0-20230731055551-bb3fd770109c
	github.com/agiledragon/gomonkey/v2 v2.9.0
	github.com/golang/mock v1.6.0
	github.com/json-iterator/go v1.1.12
	github.com/smartystreets/goconvey v1.7.2
	go.uber.org/atomic v1.10.0
	google.golang.org/grpc v1.56.2
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20181017120253-0766667cb4d1 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/prometheus/client_golang v1.14.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.39.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/smartystreets/assertions v1.2.0 // indirect
	github.com/stretchr/testify v1.8.1 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

replace (
	code.bydev.io/fbu/gateway/gway.git/ggrpc => ../ggrpc
	google.golang.org/grpc => google.golang.org/grpc v1.29.1
)
