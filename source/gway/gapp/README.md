# gapp

- 生命周期管理
  - start -> stop -> shutdown -> destroy
- Endpoint管理
  - 默认会启动6480端口进行监听，若启动address为空，则忽略启动
  - 可通过root path查看所有注册的endpoint，默认提供prometheus监控, pprof接口，health check endpoint实现
    - health check接口需要用户单独注册，且需要提供HealthFunc callback实现health校验逻辑
  - 使用标准http.HandlerFunc注册Endpoint,底层使用httprouter库管理path,可以使用:naming的方式匹配path
    - 详见: [httprouter]<https://github.com/julienschmidt/httprouter>
    - 注意: 不要直接使用httprouter相关方法,可通过gapp.ParamsFromContext获取path中的参数
  - 支持admin handler
- 退出超时
- daemon管理(暂未实现)
  - github.com/ochinchina/go-daemon

## 使用方法

```go
package gapp_test

import (
 "context"
 "fmt"
 "testing"

 "code.bydev.io/fbu/gateway/gway.git/gapp"
)

func TestApp(t *testing.T) {
 svc := &demoService{}
 gapp.Run(gapp.WithDefaultEndpoints(), gapp.WithHealth(svc.Health), gapp.WithLifecycles(svc))
}

type demoService struct {
}

func (s *demoService) Health() (bool, interface{}) {
 return true, nil
}

func (s *demoService) OnLifecycle(ctx context.Context, event gapp.LifecycleEvent) error {
 switch event {
 case gapp.LifecycleStart:
  fmt.Println("start")
 case gapp.LifecycleStop:
  fmt.Println("stop")
 }
 return nil
}

```
