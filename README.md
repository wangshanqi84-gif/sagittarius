# 微服务框架golang版本

## 环境变量说明
NACOS配置
> SGT_NACOS_SERVER_PATH - 配置中心server地址(必要)  
> SGT_NACOS_ACCESS - 配置中心鉴权accessKey  
> SGT_NACOS_SECRET - 配置中心鉴权secretKey  
> SGT_NACOS_USERNAME - 配置中心Username  
> SGT_NACOS_PASSWORD - 配置中心password  

ETCD配置
> SGT_ETCD_ENDPOINTS - etcd地址
> SGT_ETCD_USERNAME - etcd用户名
> SGT_ETCD_PASSWORD - etcd密码
> SGT_ETCD_DAIL_TIMEOUT - 超时时间 默认"10s"

Consul配置
> SGT_CONSUL_HTTP_ADDR - consul地址

Sentry配置
> SGT_EVN_SENTRY_DNS - sentry错误报警服务地址

Jaeger配置
> SGT_JAEGER_ADDR - jaeger链路追踪信息收集服务地址

基本配置
> SGT_ENV_SERVICE - 服务环境(testing等 除testing外自定义) 默认"testing"
> SGT_CONFIG_SOURCE - 配置来源 支持file/nacos/etcd 默认"nacos" (基础配置和自定义配置必须同源，且当使用file时，需通过config.WithPath来指定文件路径)
> SGT_CONFIG_FORMAT - 配置格式 支持json/yaml/xml 默认"json"
> SGT_METRIC_DISABLE - 是否禁用监控(runtime/pprof/sentry) true:关闭(不推荐) 默认"false"
> SGT_LOG_PATH - 日志保存路径 默认"./log"

## 集成中间件
Mysql : gorm  
Redis : go-redis & go-redsync/redsync  
服务发现: etcd & consul & nacos  
配置中心: nacos & etcd  
消息队列: kafka & rocketMQ  
链路追踪: jaeger  
监控: pprof & runtime & sentry  

## 支持服务类型
http & https & grpc & websocket & socket.io  

## 配置使用示例

### 日志配置
rotation - 日志分割方式(day/hour) 默认day  
saveDays - 日志保存天数 默认3  
level - 日志级别(debug/info/warn/error) 默认debug  
format - 日志格式(console/json) 默认console
```json
{
  "rotation": "hour",
  "saveDays": 7,
  "level": "debug",
  "format": "json"
}
```

### 启动服务配置
proto - 协议类型(http/rpc/websocket)  
port - 端口号  
特别说明：pprof使用的端口号为所有配置server的最大端口号+1  
```json
[
  {
    "proto": "http",
    "port": 11001
  },
  {
    "proto": "websocket",
    "port": 11002
  }
]
```

### 服务发现配置
used - 服务发现方式(etcd/consul/nacos) 需配合对应的环境变量配置
```json
{
  "used": "consul"
}
```

### 数据库配置
name - 名称 配置检索使用  
master - 主库dns 主从配置将进行读写分离  
slaves - 从库dns  
maxOpen - 链接池最大连接数 配置前请通知DBA  
maxIdle - 链接池最大空闲连接数 配置前请通知DBA  
maxLifeTime - 链接可复用最大时间  
maxIdleTime - 链接池链接最大空闲时长
```json
[
  {
    "name":"xxxxxx", 
    "master":"app:**********@tcp(192.168.0.19:3306)/game?charset=utf8mb4&parseTime=true&loc=Local&timeout=5s&readTimeout=5s", 
    "slaves":["app:**********@tcp(192.168.0.19:3306)/game?charset=utf8mb4&parseTime=true&loc=Local&timeout=5s&readTimeout=5s"], 
    "maxOpen": 100, 
    "maxIdle": 50, 
    "maxLifeTime": "15m", 
    "maxIdleTime": "1h"
  }
]
```

### redis配置
name - 名称 配置检索使用  
model - redis模式(singleton/sentinel/cluster) 默认singleton  
addr - 地址 多地址则用','分割  
database - db库  
maxRetry - 最大重试次数  
idleTimeout - 客户端关闭空闲连接的时间 应该小于服务器的超时时间 默认为300秒 -1禁用空闲超时检查  
readTimeout - 命令读超时 默认3(秒单位)  
writeTimeout - 命令写超时 默认等于读超时(秒单位)
poolSize - 连接池大小 默认100 配置前请通知DBA  
minIdleConn - 最小连接数量 默认35 配置前请通知DBA  
username - 用户名  
password - 密码
```json
[
  {
    "name": "xxxxxx",
    "model": "singleton",
    "addr": "192.168.0.19:6379",
    "database": 0,
    "maxRetry": 3,
    "idleTimeout": 600,
    "readTimeout": 5,
    "writeTimeout": 3,
    "poolSize": 150,
    "minIdleConn": 50,
    "username": "master",
    "password": "******"
  }
]
```

### 下游服务配置
namespace - 下游服务namespace  
product - 下游服务产品  
serviceName - 服务名称 namespace.product.serviceName为服务发现名  
proto - 下游服务协议(rpc/http)  
endpoints - endpoints 多个','分割 如果使用服务发现则该配置为兜底 如果未使用服务发现则该配置生效  
unUseDiscovery - 是否禁止服务发现 默认false 如果服务本身未配置服务发现则调用下游服务无法使用服务发现功能  
retry - 重试次数  
timeout - 超时时间 注意这里的超时时间为单次请求超时 所以接口的最坏超时需要乘以retry  
syncTimeout - 链路超时同步 即调用下游服务如果超时，则下游调用的下游服务同样超时
```json
[
  {
    "serviceName": "pt",
    "proto":"http",
    "endpoints":"https://api.testing.xxxx.xyz",
    "unUseDiscovery":true,
    "retry":2,
    "timeout":"3s"
  },
  {
    "namespace": "aries",
    "product": "common",
    "serviceName": "push.link",
    "proto": "http",
    "retry": 2,
    "timeout":"3s"
  }
]
```

### rocketmq producer配置
name - 名称 配置检索使用  
topic - topic别名映射  
brokers - brokers 使用','分割  
timeout - 发送超时时间  
accessKey - 鉴权用accessKey  
secretKey - 鉴权用secretKey  
securityToken - 鉴权用securityToken  
maxRetry - 写入最大重试次数
```json
[
  {
    "name": "xxxxxxxxx",
    "topic": [{"alias": "play", "name": "game-user_play"}],
    "brokers": "192.168.0.19:9876",
    "timeout": "5s",
    "maxRetry": 1
  }
]
```

### rocketmq consumer配置
name - 名称 配置检索使用  
brokers - brokers 使用','分割  
consumeTimeout - 队列消息超时时间  
accessKey - 鉴权用accessKey  
secretKey - 鉴权用secretKey  
securityToken - 鉴权用securityToken  
maxRetry - 读取最大重试次数  
mode - 消费模式 0:broadcasting 1:clustering  
from - 消费起点 0:最近一次提交 1:记录的最早一次提交 2:当前时间  
groupName - 消费组名称  
maxReconsumeTimes - 消费重试次数  
expression - Tag标签过滤器
```json
[
  {
    "name": "xxxxxxxx",
    "brokers": "192.168.0.19:9876",
    "consumeTimeout": "60m",
    "maxRetry": 5,
    "mode": 1,
    "from": 0,
    "groupName": "game_over-topic",
    "maxReconsumeTimes": 1,
    "expression": "*"
  }
]
```

### 其他
accessRequestDisable - access日志禁止输出request信息 默认false

```
标准配置由框架读取，自定义配置需要独立配置文件  
配置会自动进行监听
```go
package conf

import "github.com/wangshanqi84-gif/sagittarius/app"

type MyConfig struct {
	Value string `json:"value"`
}

func GetMyConfig() *MyConfig {
	var cfg MyConfig
	cli, err := app.Router().ConfigClient()
	if err != nil {
	    panic(err)
	}
	if err = cli.GetConfig("cus-game.config", &cfg); err != nil {
	    panic(err)
	}
	return &cfg
}
```

### 整体例子
```json
{
  "accessRequestDisable": true,
  "log": {
    "rotation": "day",
    "saveDays": 3,
    "level": "debug"
  },
  "servers":[{"proto":"http","port":11001}],
  "discovery":{
    "used": "consul"
  },
  "databases":[
    {
      "name":"xxxxxx",
      "master":"app:**********@tcp(192.168.0.19:3306)/game?charset=utf8mb4&parseTime=true&loc=Local&timeout=5s&readTimeout=5s",
      "slaves":["app:**********@tcp(192.168.0.19:3306)/game?charset=utf8mb4&parseTime=true&loc=Local&timeout=5s&readTimeout=5s"],
      "maxOpen": 100,
      "maxIdle": 50,
      "maxLifeTime": "15m",
      "maxIdleTime": "1h"
    }
  ],
  "redis":[
    {
      "name": "xxxxxx",
      "model": "singleton",
      "addr": "192.168.0.19:6379",
      "database": 0,
      "maxRetry": 3,
      "password": "******"
    }
  ],
  "clients":[
    {
      "serviceName": "pt",
      "proto":"http",
      "endpoints":"https://api.testing.xxxxxx.xyz",
      "unUseDiscovery":true,
      "retry":2,
      "timeout":"3s"
    },
    {
      "namespace": "aries",
      "product": "common",
      "serviceName": "push.link",
      "proto": "http",
      "retry": 2,
      "timeout":"3s"
    }
  ],
  "producers": [
    {
      "name": "producer",
      "brokers": "192.168.0.19:9876",
      "timeout": "5s",
      "maxRetry": 1
    }
  ],
  "consumers": [
    {
      "name": "consumer",
      "brokers": "192.168.0.19:9876",
      "consumeTimeout": "60m",
      "maxRetry": 5,
      "mode": 1,
      "from": 0,
      "groupName": "game_over-topic",
      "maxReconsumeTimes": 1,
      "expression": "*"
    }
  ]
}
```

## 代码示例

### 框架启动

```go
package main

import (
    "flag"
    "fmt"

    "github.com/wangshanqi84-gif/sagittarius/app"
    "github.com/wangshanqi84-gif/sagittarius/app/config"
    "github.com/wangshanqi84-gif/sagittarius/cores/env"
    "github.com/wangshanqi84-gif/sagittarius/app/server"
    httpSrv "github.com/wangshanqi84-gif/sagittarius/cores/http/server"
)

const (
    namespace   = "xxxxx"
    product     = "xx"
    serviceName = "game.room"
)

var (
    confPath = flag.String("confPath", "", "confPath is none")
)

func init() {
    flag.Parse()

    var opts []config.Option
    if *confPath != "" {
        // 仅测试环境支持通过启动项传入配置文件地址
        opts = append(opts, config.WithPath(*confPath))
    }
    // 初始化框架&服务定义
    app.InitRouter(&config.ServiceDefine{
        ConfNamespace: namespace,
        Namespace:     namespace,
        Product:       product,
        ServiceName:   serviceName,
    }, opts...)
}

func NewServer() *httpSrv.Engine {
    srv := server.InitHttpServer(httpSrv.OnStop(func() {
        fmt.Println("on http server stop....")
    }))
    // 业务接口
    busi := srv.NewGroup("/game/api/v1")
    {
        busi.GET("/info", func(c *httpSrv.Context) { fmt.Println("do something...")})
    }
    return srv
}

func main() {
    // 初始化启动服务
    srv := NewServer()
    // 将server绑定到框架
    app.Router().BindServer(srv)
    // 框架启动
    app.Run()
}
```

### 下游服务调用
```go
package push

import (
    "context"
    "fmt"
    netHttp "net/http"

    "github.com/wangshanqi84-gif/sagittarius/app/proxy"
    httpCli "github.com/wangshanqi84-gif/sagittarius/cores/http/client"
    gErrors "github.com/wangshanqi84-gif/sagittarius/cores/errors"
    "github.com/wangshanqi84-gif/sagittarius/logger"

    "github.com/pkg/errors"
)

const (
    Name = "aries.common.push.link"
)

const (
    MessagePushUrl = "/server_api/v1/push/message"
)

type RequestPushMessage struct {
    App     string      `json:"app"`  // required
    Mode    int         `json:"mode"` // 1:app内广播 2:组内广播 3:指定id推送
    Group   string      `json:"group,omitempty"`
    To      []string    `json:"to,omitempty"`
    Version string      `json:"version,omitempty"`
    Tag     string      `json:"tag,omitempty"`
    SubType int32       `json:"subType"`
    Payload interface{} `json:"payload,omitempty"`
}

type ResponsePushMessage struct {
    Status  int    `json:"status"`
    Message string `json:"message"`
}

func MessagePush(ctx context.Context, req *RequestPushMessage) error {
    c, err := proxy.InitHttpClient(ctx, Name)
    if err != nil {
        return errors.WithMessage(err, "| proxy.InitHttpClient")
    }
    var rsp ResponsePushMessage
    httpRsp, err := c.JsonPost(httpCli.Request(ctx, MessagePushUrl), req, &rsp)
    if err != nil {
        return errors.WithMessage(err, "| c.JsonPost")
    }
    if httpRsp.StatusCode != netHttp.StatusOK {
        return errors.New(fmt.Sprintf("post %s, httpCode:%d", MessagePushUrl, httpRsp.StatusCode))
    }
    if rsp.Status != 0 {
        return gErrors.New(rsp.Status, rsp.Message)
    }
    return nil
}
```

### 数据库使用
```go
package db

import (
    "github.com/wangshanqi84-gif/sagittarius/app/proxy"
    "github.com/wangshanqi84-gif/sagittarius/mysql"
)

const dbName = "xxxxx"

type GameDB struct {
    sql *mysql.Client
}

func New() (*GameDB, error) {
    cli, err := proxy.InitSqlClient(dbName)
    if err != nil {
        return nil, err
    }
    return &GameDB{sql: cli}, nil
}
```

### redis使用
```go
package redis

import (
    "github.com/wangshanqi84-gif/sagittarius/app/proxy"
    "github.com/wangshanqi84-gif/sagittarius/redis"
)

const redisName = "xxxxx"

type GameRedis struct {
    rds *redis.Client
}

func New() (*GameRedis, error) {
    cli, err := proxy.InitRedisClient(redisName)
    if err != nil {
        return nil, err
    }
    return &GameRedis{rds: cli}, nil
}
```

