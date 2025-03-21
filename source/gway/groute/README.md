# groute

路由模块

## 路由规则

- 支持模糊匹配,以*结尾
- 静态路由支持相同path多个路由
  - 路由类型
    - 独占路由
    - Category/AccountType路由
      - 支持默认路由
    - AllInOne(AIO)路由
  - 优先级和互斥规则
    - AIO可与其他路由共存且优先匹配，且AIO路由只能有1个
    - 独占路由只能有1个
    - 独占路由不可与Category/AccountType路由共存
    - Category/AccountType路由中的Value和AccountType不能存在一样的
  - 数组中顺序
    - AIO路由始终在0位置, Category的默认路由始终在最后
