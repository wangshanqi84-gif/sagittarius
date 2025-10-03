package env

// 服务基础环境变量
// SGT_ENV_SERVICE 当前环境(测试/予发布/生产) testing:测试环境 默认testing
// SGT_CONFIG_SOURCE 配置文件来源(目前支持nacos，etcd，file) 默认consul
// SGT_CONFIG_FORMAT 配置文件格式(目前支持json，xml，yaml) 默认json
// SGT_LOG_PATH 日志路径 默认为"./log"
// SGT_METRIC_DISABLE 是否禁用监控 true为关闭 默认false
// --

const (
	SgtEnvService    = "SGT_ENV_SERVICE"
	SgtConfigSource  = "SGT_CONFIG_SOURCE"
	SgtConfigFormat  = "SGT_CONFIG_FORMAT"
	SgtLogPath       = "SGT_LOG_PATH"
	SgtMetricDisable = "SGT_METRIC_DISABLE"
)

// Nacos相关环境变量
// SGT_NACOS_SERVER_PATH  配置中心nacos地址，使用nacos作为配置中心时必要
// SGT_NACOS_ACCESS nacos的accessKey配置
// SGT_NACOS_SECRET nacos的secretKey配置
// SGT_NACOS_USERNAME nacos用户名
// SGT_NACOS_PASSWORD nacos密码
// --

const (
	SgtNacosServerPath = "SGT_NACOS_SERVER_PATH"
	SgtNacosAccess     = "SGT_NACOS_ACCESS"
	SgtNacosSecret     = "SGT_NACOS_SECRET"
	SgtNacosUsername   = "SGT_NACOS_USERNAME"
	SgtNacosPassword   = "SGT_NACOS_PASSWORD"
)

// Etcd相关环境变量
// SGT_ETCD_ENDPOINTS  配置中心etcd地址列表，使用etcd作为配置中心时必要
// SGT_ETCD_USERNAME etcd用户名
// SGT_ETCD_PASSWORD etcd密码
// SGT_ETCD_DAIL_TIMEOUT 超时时间，默认10s
// --

const (
	SgtEtcdEndpoints   = "SGT_ETCD_ENDPOINTS"
	SgtEtcdUsername    = "SGT_ETCD_USERNAME"
	SgtEtcdPassword    = "SGT_ETCD_PASSWORD"
	SgtEtcdDailTimeout = "SGT_ETCD_DAIL_TIMEOUT"
)

// Jaeger相关环境变量
// SGT_JAEGER_ADDR jaeger地址(链路追踪收集)可选
// --

const (
	SgtJaegerAddr = "SGT_JAEGER_ADDR"
)

// Consul相关环境变量
// SGT_CONSUL_HTTP_ADDR consul http地址
// --

const (
	SgtConsulAddr = "SGT_CONSUL_HTTP_ADDR"
)

// Sentry相关环境变量
// SGT_EVN_SENTRY_DNS sentry地址(错误报警) 可选
// --

const (
	SgtEvnSentryDns = "SGT_EVN_SENTRY_DNS"
)
