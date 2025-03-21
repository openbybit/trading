# gsymbol

## unit test

> Use the following command to unit test

``` go
go test -gcflags=all=-l -v
```

## 使用文档

* 服务启动时,需要调用Start方法初始化
* 使用gsymbol.GetXXX获取symbol
* future接口上支持了brokerID查询symbol,但当前数据源并没有提供ExhibitSiteList字段,故还不能使用brokerID查询功能
* gsymbol/future为历史实现,不建议使用

## 参考文档

* [mirana](https://uponly.larksuite.com/wiki/wikusRrDlj8ZBmOH1pwpmCPAucx)
