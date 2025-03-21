# gconfig

配置中心

- 支持高级接口和底层接口两套接口
  - Load方法用于加载配置,并自动unmarshal到结构体中,并自动监听数据变更
  - Get/Put/Delete/Listen用于提供原始接口调用
- 支持从nacos中动态加载配置
- 支持mock实现
- LoadFile方法用于从静态文件中加载配置

## 使用方式

```go
type Config struct {
    Enable  bool               `yaml:"enable" json:"enable"` // 是否开启次功能,如果不开启则默认返回true
    UidList []int64            `yaml:"uids" json:"uids"`     // uid白名单列表
    uidMap  map[int64]struct{} // 构建后数据
}

func (c *Config) OnLoaded() {
    c.uidMap = make(map[int64]struct{})
    for _, uid := range c.UidList {
    c.uidMap[uid] = struct{}{}
    }
}

var config = &Config{}

func main() {
    // 1: 使用nacos设置全局Configure
    gconfig.SetDefault(nacos.New(nacos.Config{Address: "xxx"}))
    // 2: 加载配置,注意,需要传入的是指针的指针
    gconfig.Load(context.Background(), "ipcheck_whitelist", &config, json.Unmarshal, nil)
}
```
