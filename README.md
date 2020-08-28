# confd 项目
项目基于[nacos-confd](https://github.com/nacos-group/nacos-confd)修改扩展，用于Nacos 1.2.0 增加权限控制后，拉取配置中心配置.

## 新增功能
- [x] confd新增权限认证
- [x] confd增加yaml解析

## 构建confd流程
go环境准备～～～

```
$ mkdir -p $GOPATH/src/github.com/kelseyhightower
$ git clone https://github.com/lyln/nacos-confd.git $GOPATH/src/github.com/kelseyhightower/confd
$ cd $GOPATH/src/github.com/kelseyhightower/confd
$ make
```

notes:
```
$GOPATH/src/github.com/kelseyhightower下项目重命名为confd

```

## confd权限使用
参考配置
/etc/confd/confd.toml
```
backend = "nacos"
confdir = "/etc/confd"
#log-level = "debug"
namespace = "dev"
interval = 5
nodes = [
  "nacos_server",
  ]
scheme = "http"
watch = true
NacosUsername = "nacos"
NacosPassword = "nacos"

```

## 其他疑惑
欢迎共同学习探讨。
![qr](https://lyln.oss-cn-beijing.aliyuncs.com/wx/irisloveli.jpg?230x230)
