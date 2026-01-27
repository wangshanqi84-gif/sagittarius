package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/wangshanqi84-gif/sagittarius/configuration"
	cfgEtcd "github.com/wangshanqi84-gif/sagittarius/configuration/etcd"
	"github.com/wangshanqi84-gif/sagittarius/configuration/file"
	cfgNacos "github.com/wangshanqi84-gif/sagittarius/configuration/nacos"
	"github.com/wangshanqi84-gif/sagittarius/cores/env"
	"github.com/wangshanqi84-gif/sagittarius/cores/registry"
	"github.com/wangshanqi84-gif/sagittarius/etcd"
	"github.com/wangshanqi84-gif/sagittarius/logger"
	"github.com/wangshanqi84-gif/sagittarius/nacos"
)

/////////////////////////////////////////////////
// 服务强制定义项
/////////////////////////////////////////////////

// 服务发现明完整为namespace.product.name

// ServiceDefine 服务定义信息
type ServiceDefine struct {
	Namespace   string // 所属命名空间
	Product     string // 产品 product
	ServiceName string // 服务名(两段式推荐)
}

/////////////////////////////////////////////////

// LogConfig 服务日志配置
type LogConfig struct {
	// 日志分割方式
	Rotation string `yaml:"rotation" json:"rotation" xml:"rotation"`
	// 日志保存天数
	SaveDays int `yaml:"saveDays" json:"saveDays" xml:"saveDays"`
	// 日志级别
	Level string `yaml:"level" json:"level" xml:"level"`
	// 日志格式
	Format string `yaml:"format" json:"format" xml:"format"`
}

// ServerConfig 启动服务配置
type ServerConfig struct {
	// 协议类型 http/rpc/websocket
	Proto string `yaml:"proto" json:"proto" xml:"proto"`
	// 启动端口
	Port int `yaml:"port" json:"port" xml:"port"`
}

// DiscoveryConfig 服务发现配置
type DiscoveryConfig struct {
	// 服务发现方式 目前etcd/consul
	Used string `yaml:"used" json:"used" xml:"used"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	// 名称
	Name string `yaml:"name" json:"name" xml:"name"`
	// 主库dns
	Master string `yaml:"master" json:"master" xml:"master"`
	// 从库dns
	Slaves []string `yaml:"slaves" json:"slaves" xml:"slaves"`
	// 最大连接数
	MaxOpen int `yaml:"maxOpen" json:"maxOpen" xml:"maxOpen"`
	// 最大空闲连接数
	MaxIdle int `yaml:"maxIdle" json:"maxIdle" xml:"maxIdle"`
	// 链接可复用最大时间
	MaxLifeTime string `yaml:"maxLifeTime" json:"maxLifeTime" xml:"maxLifeTime"`
	// 链接池链接最大空闲时长
	MaxIdleTime string `yaml:"maxIdleTime" json:"maxIdleTime" xml:"maxIdleTime"`
}

// RedisConfig redis缓存配置
type RedisConfig struct {
	// 名称
	Name string `yaml:"name" json:"name" xml:"name"`
	// 模式 singleton/sentinel/cluster 默认singleton
	Model string `yaml:"model" json:"model" xml:"model"`
	// 地址 多地址则用','分割
	Addr string `yaml:"addr" json:"addr" xml:"addr"`
	// redis db
	DataBase int `yaml:"database" json:"database" xml:"database"`
	// 最大重试次数
	MaxRetry int `yaml:"maxRetry" json:"maxRetry" xml:"maxRetry"`
	// 客户端关闭空闲连接的时间。应该小于服务器的超时时间。默认为5分钟。-1禁用空闲超时检查。
	IdleTimeout int `yaml:"idleTimeout" json:"idleTimeout" xml:"idleTimeout"`
	// 命令读超时，默认3(秒单位)
	ReadTimeout int `yaml:"readTimeout" json:"readTimeout" xml:"readTimeout"`
	// 命令写超时，默认等于读超时(秒单位)
	WriteTimeout int `yaml:"writeTimeout" json:"writeTimeout" xml:"writeTimeout"`
	// 连接池大小 默认100
	PoolSize int `yaml:"poolSize" json:"poolSize" xml:"poolSize"`
	// 最小连接数量 默认35
	MinIdleConn int `yaml:"minIdleConn" json:"minIdleConn" xml:"minIdleConn"`
	// 账户(默认为default)
	Username string `yaml:"username" json:"username" xml:"username"`
	// 密码
	Password string `yaml:"password" json:"password" xml:"password"`
}

// ClientConfig 下游客户端配置
type ClientConfig struct {
	// 下游服务namespace
	Namespace string `yaml:"namespace" json:"namespace" xml:"namespace"`
	// 下游服务产品
	Product string `yaml:"product" json:"product" xml:"product"`
	// 服务名称
	ServiceName string `yaml:"serviceName" json:"serviceName" xml:"serviceName"`
	// 下游服务协议 rpc/http
	Proto string `yaml:"proto" json:"proto" xml:"proto"`
	// endpoints 多个','分割
	EndPoints string `yaml:"endpoints" json:"endpoints" xml:"endpoints"`
	// 是否禁止服务发现
	UnUseDiscovery bool `yaml:"unUseDiscovery" json:"unUseDiscovery" xml:"unUseDiscovery"`
	// 重试次数
	Retry int `yaml:"retry" json:"retry" xml:"retry"`
	// 超时时间
	Timeout string `yaml:"timeout" json:"timeout" xml:"timeout"`
	// 代理超时时间
	DialTimeout string `yaml:"dialTimeout" json:"dialTimeout" xml:"dialTimeout"`
	// 连接保活时间
	KeepAlive string `yaml:"keepAlive" json:"keepAlive" xml:"keepAlive"`
	// 同步下游超时时间
	SyncTimeout bool `yaml:"syncTimeout" json:"syncTimeout" xml:"syncTimeout"`
	// 最大空闲连接数
	MaxIdleConns int `yaml:"maxIdleConns" json:"maxIdleConns" xml:"maxIdleConns"`
	// 每个主机最大空闲连接数
	MaxIdleConnsPerHost int `yaml:"maxIdleConnsPerHost" json:"maxIdleConnsPerHost" xml:"maxIdleConnsPerHost"`
	// 空闲连接超时时间
	IdleConnTimeout string `yaml:"idleConnTimeout" json:"idleConnTimeout" xml:"idleConnTimeout"`
	// TLS握手超时时间
	TLSHandshakeTimeout string `yaml:"tlsHandshakeTimeout" json:"tlsHandshakeTimeout" xml:"tlsHandshakeTimeout"`
	// tls配置
	TLS *struct {
		// 目标主机名
		ServerName string `yaml:"serverName" json:"serverName" xml:"serverName"`
		// 安全证书
		CertFile string `yaml:"certFile" json:"certFile" xml:"certFile"`
		// key证书
		KeyFile string `yaml:"keyFile" json:"keyFile" xml:"keyFile"`
		// ca证书
		CAFile string `yaml:"caFile" json:"caFile" xml:"caFile"`
	} `yaml:"tls" json:"tls" xml:"tls"`
}

type ProducerTopic struct {
	Alias string `yaml:"alias" json:"alias" xml:"alias"`
	Name  string `yaml:"name" json:"name" xml:"name"`
}

// RocketProducerConfig rocket producer配置
type RocketProducerConfig struct {
	// 名称
	Name string `yaml:"name" json:"name" xml:"name"`
	// topics别名映射
	Topic []*ProducerTopic `yaml:"topic" json:"topic" xml:"topic"`
	// brokers 使用','分割
	Brokers string `yaml:"brokers" json:"brokers" xml:"brokers"`
	// 发送超时时间
	Timeout string `yaml:"timeout" json:"timeout" xml:"timeout"`
	// 鉴权用accessKey
	AccessKey string `yaml:"accessKey" json:"accessKey" xml:"accessKey"`
	// 鉴权用secretKey
	SecretKey string `yaml:"secretKey" json:"secretKey" xml:"secretKey"`
	// 鉴权用securityToken
	SecurityToken string `yaml:"securityToken" json:"securityToken" xml:"securityToken"`
	// 写入最大重试次数
	MaxRetry int `yaml:"maxRetry" json:"maxRetry" xml:"maxRetry"`
}

// RocketConsumerConfig rocket consumer配置
type RocketConsumerConfig struct {
	// 名称
	Name string `yaml:"name" json:"name" xml:"name"`
	// brokers 使用','分割
	Brokers string `yaml:"brokers" json:"brokers" xml:"brokers"`
	// 队列消息超时时间
	ConsumeTimeout string `yaml:"consumeTimeout" json:"consumeTimeout" xml:"consumeTimeout"`
	// 鉴权用accessKey
	AccessKey string `yaml:"accessKey" json:"accessKey" xml:"accessKey"`
	// 鉴权用secretKey
	SecretKey string `yaml:"secretKey" json:"secretKey" xml:"secretKey"`
	// 鉴权用securityToken
	SecurityToken string `yaml:"securityToken" json:"securityToken" xml:"securityToken"`
	// 读取最大重试次数
	MaxRetry int `yaml:"maxRetry" json:"maxRetry" xml:"maxRetry"`
	// 消费模式 0:broadcasting 1:clustering
	Mode int `yaml:"mode" json:"mode" xml:"mode"`
	// 消费起点 0:最近一次提交 1:记录的最早一次提交 2:当前时间
	From int `yaml:"from" json:"from" xml:"from"`
	// 消费组名称
	GroupName string `yaml:"groupName" json:"groupName" xml:"groupName"`
	// 消费重试次数
	MaxReconsumeTimes int32 `yaml:"maxReconsumeTimes" json:"maxReconsumeTimes" xml:"maxReconsumeTimes"`
	// Tag标签过滤器
	Expression string `yaml:"expression" json:"expression" xml:"expression"`
}

// KafkaProducerConfig kafka producer配置
type KafkaProducerConfig struct {
	// 名称
	Name string `yaml:"name" json:"name" toml:"name"`
	// topics别名映射
	Topic []*ProducerTopic `yaml:"topic" json:"topic" xml:"topic"`
	// brokers 使用','分割
	Brokers string `yaml:"brokers" json:"brokers" xml:"brokers"`
	// 禁止发送结果通知 默认false false情况下必须监听success和error
	DisableNotify bool `yaml:"disableNotify" json:"disableNotify" toml:"disableNotify"`
	// 超时时间
	Timeout string `yaml:"timeout" json:"timeout" toml:"timeout"`
	// 最大消息字节
	MaxMessageBytes int `yaml:"maxMessageBytes" json:"maxMessageBytes" toml:"maxMessageBytes"`
	// 写入最大重试次数
	MaxRetry int `yaml:"maxRetry" json:"maxRetry" toml:"maxRetry"`
	// 模式 sync/async 默认async
	Mode string `yaml:"mode" json:"mode" toml:"mode"`
}

// KafkaConsumerConfig kafka consumer配置
type KafkaConsumerConfig struct {
	// 名称
	Name string `yaml:"name" json:"name" toml:"name"`
	// brokers 使用','分割
	Brokers string `yaml:"brokers" json:"brokers" toml:"brokers"`
	// 组名
	Group string `yaml:"group" json:"group" toml:"group"`
	//  topics别名映射
	Topics []*ProducerTopic `yaml:"topics" json:"topics" xml:"topics"`
	// 分区策略 使用','分割 sticky/roundrobin/range 默认sticky
	ReBalance string `yaml:"reBalance" json:"reBalance" toml:"reBalance"`
	// 消费offset最新或最早 -1/-2 默认-1最新
	OffsetInitial int64 `yaml:"offsetInitial" json:"offsetInitial" toml:"offsetInitial"`
	// 提交最大重试次数
	CommitRetry int `yaml:"commitRetry" json:"commitRetry" toml:"commitRetry"`
	// 返回消息最大等待时间
	MaxWaitTime string `yaml:"maxWaitTime" json:"maxWaitTime" toml:"maxWaitTime"`
	// 是否允许自动创建不存在的topic 默认false
	TopicCreateEnable bool `yaml:"topicCreateEnable" json:"topicCreateEnable" toml:"topicCreateEnable"`
	// 是否允许自动创提交offset
	AutoCommit bool `yaml:"autoCommit" json:"autoCommit" toml:"autoCommit"`
	// handler池数量
	WorkerNumbers int `yaml:"workerNumbers" json:"workerNumbers" toml:"workerNumbers"`
	// handler队列缓冲区大小
	SeqNumbers int `yaml:"seqNumbers" json:"seqNumbers" toml:"seqNumbers"`
}

type ServiceConfig struct {
	// access日志禁止输出request信息 默认false
	AccessRequestDisable bool `yaml:"accessRequestDisable" json:"accessRequestDisable" xml:"accessRequestDisable"`
	// 日志配置
	Log *LogConfig `yaml:"log" json:"log" xml:"log"`
	// 启动服务配置
	Svrs []*ServerConfig `yaml:"servers" json:"servers" xml:"servers"`
	// 服务发现配置
	Discovery *DiscoveryConfig `yaml:"discovery" json:"discovery" xml:"discovery"`
	// 数据库配置
	Databases []*DatabaseConfig `yaml:"databases" json:"databases" xml:"databases"`
	// redis配置
	Rds []*RedisConfig `yaml:"redis" json:"redis" xml:"redis"`
	// 下游服务配置
	Clients []*ClientConfig `yaml:"clients" json:"clients" xml:"clients"`
	// rocket producer配置
	RocketProducers []*RocketProducerConfig `yaml:"rocketProducers" json:"rocketProducers" xml:"rocketProducers"`
	// rocket consumer配置
	RocketConsumers []*RocketConsumerConfig `yaml:"rocketConsumers" json:"rocketConsumers" xml:"rocketConsumers"`
	// kafka producer配置
	KafkaProducers []*KafkaProducerConfig `yaml:"kafkaProducers" json:"kafkaProducers" xml:"kafkaProducers"`
	// kafka consumer配置
	KafkaConsumers []*KafkaConsumerConfig `yaml:"kafkaConsumers" json:"kafkaConsumers" xml:"kafkaConsumers"`
}

func (c *ServiceConfig) GetDatabase(name string) *DatabaseConfig {
	for _, db := range c.Databases {
		if db.Name == name {
			return db
		}
	}
	return nil
}

func (c *ServiceConfig) GetRedis(name string) *RedisConfig {
	for _, rds := range c.Rds {
		if rds.Name == name {
			return rds
		}
	}
	return nil
}

func (c *ServiceConfig) GetClient(name string, proto string) (string, *ClientConfig) {
	for _, cli := range c.Clients {
		fullName := strings.TrimLeft(
			fmt.Sprintf("%s.%s.%s", cli.Namespace, cli.Product, cli.ServiceName),
			".",
		)
		if name == fullName && proto == strings.ToLower(cli.Proto) {
			return fmt.Sprintf("%s-%s", fullName, strings.ToLower(cli.Proto)), cli
		}
	}
	return "", nil
}

func (c *ServiceConfig) RocketProducer(name string) *RocketProducerConfig {
	for _, p := range c.RocketProducers {
		if p.Name == name {
			return p
		}
	}
	return nil
}

func (c *ServiceConfig) RocketConsumer(name string) *RocketConsumerConfig {
	for _, con := range c.RocketConsumers {
		if con.Name == name {
			return con
		}
	}
	return nil
}

func (c *ServiceConfig) KafkaProducer(name string) *KafkaProducerConfig {
	for _, p := range c.KafkaProducers {
		if p.Name == name {
			return p
		}
	}
	return nil
}

func (c *ServiceConfig) KafkaConsumer(name string) *KafkaConsumerConfig {
	for _, con := range c.KafkaConsumers {
		if con.Name == name {
			return con
		}
	}
	return nil
}

/////////////////////////////////////////////////

const (
	defaultConfigSource = "nacos"
	defaultConfigFormat = "json"
)

var (
	configSourceMap = map[string]struct{}{
		"nacos": {},
		"etcd":  {},
		"file":  {},
	}
)

type Option func(*option)

type option struct {
	path string
}

func WithPath(path string) Option {
	return func(o *option) {
		o.path = path
	}
}

func Initialize(ctx context.Context, info *registry.Service, v interface{}, opts ...Option) (configuration.IConfig, error) {
	var err error
	o := option{}
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	// 获取配置来源
	source := strings.ToLower(env.GetEnv(env.SgtConfigSource))
	if source == "" {
		source = defaultConfigSource
	}
	if _, has := configSourceMap[source]; !has {
		return nil, errors.New(fmt.Sprintf("config source ignore, source:%v", source))
	}
	if source == "file" && o.path == "" {
		return nil, errors.New("config file does not exist")
	}
	format := strings.ToLower(env.GetEnv(env.SgtConfigFormat))
	if format == "" {
		format = defaultConfigFormat
	}
	var cfg configuration.IConfig
	var name string
	switch source {
	case "nacos":
		path, accessKey, secretKey, userName, password := env.GetNacosEnv()
		if path == "" {
			return nil, errors.New("nacos-server config center path undefined")
		}
		ncopts := []nacos.Option{
			nacos.WithRunEnv(env.GetRunEnv()),
			nacos.WithLogger(logger.GetGen()),
			nacos.WithServerPath(path),
		}
		if accessKey != "" {
			ncopts = append(ncopts, nacos.WithAccessKey(accessKey))
		}
		if secretKey != "" {
			ncopts = append(ncopts, nacos.WithSecretKey(secretKey))
		}
		if userName != "" {
			ncopts = append(ncopts, nacos.WithUserName(userName))
		}
		if password != "" {
			ncopts = append(ncopts, nacos.WithPassword(password))
		}
		cfg = cfgNacos.NewConfigClient(ctx, info.Namespace, info.Product, info.ServiceName, format, ncopts...)
		name = fmt.Sprintf("%s.%s.config", info.Product, info.ServiceName)
	case "etcd":
		eps, userName, password, dailTimeout := env.GetEtcdEnv()
		if eps == "" {
			return nil, errors.New("etcd-server config center endpoints undefined")
		}
		ss := strings.Split(eps, ",")
		edopts := []etcd.Option{
			etcd.Endpoints(ss),
			etcd.Username(userName),
			etcd.Password(password),
			etcd.DialTimeout(dailTimeout),
		}
		cfg = cfgEtcd.NewConfigClient(ctx, info.Namespace, info.Product, info.ServiceName, format, edopts...)
		name = fmt.Sprintf("%s/%s/config", info.Product, info.ServiceName)
	case "file":
		cfg = file.NewConfigClient(ctx, format)
		name = o.path
	}
	if err = cfg.GetConfig(name, v); err != nil {
		return nil, err
	}
	return cfg, nil
}

func Custom(ctx context.Context, info *registry.Service, opts ...Option) (configuration.IConfig, error) {
	o := option{}
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	// 获取配置来源
	source := strings.ToLower(env.GetEnv(env.SgtConfigSource))
	if source == "" {
		source = defaultConfigSource
	}
	if _, has := configSourceMap[source]; !has {
		return nil, errors.New(fmt.Sprintf("config source ignore, source:%v", source))
	}
	if source == "file" && o.path == "" {
		return nil, errors.New("config file does not exist")
	}
	format := strings.ToLower(env.GetEnv(env.SgtConfigFormat))
	if format == "" {
		format = defaultConfigFormat
	}
	var cfg configuration.IConfig
	switch source {
	case "nacos":
		path, accessKey, secretKey, userName, password := env.GetNacosEnv()
		if path == "" {
			return nil, errors.New("nacos-server config center path undefined")
		}
		ncopts := []nacos.Option{
			nacos.WithRunEnv(env.GetRunEnv()),
			nacos.WithLogger(logger.GetGen()),
			nacos.WithServerPath(path),
		}
		if accessKey != "" {
			ncopts = append(ncopts, nacos.WithAccessKey(accessKey))
		}
		if secretKey != "" {
			ncopts = append(ncopts, nacos.WithSecretKey(secretKey))
		}
		if userName != "" {
			ncopts = append(ncopts, nacos.WithUserName(userName))
		}
		if password != "" {
			ncopts = append(ncopts, nacos.WithPassword(password))
		}
		cfg = cfgNacos.NewConfigClient(ctx, info.Namespace, info.Product, info.ServiceName, format, ncopts...)
	case "etcd":
		eps, userName, password, dailTimeout := env.GetEtcdEnv()
		if eps == "" {
			return nil, errors.New("etcd-server config center endpoints undefined")
		}
		ss := strings.Split(eps, ",")
		edopts := []etcd.Option{
			etcd.Endpoints(ss),
			etcd.Username(userName),
			etcd.Password(password),
			etcd.DialTimeout(dailTimeout),
		}
		cfg = cfgEtcd.NewConfigClient(ctx, info.Namespace, info.Product, info.ServiceName, format, edopts...)
	case "file":
		cfg = file.NewConfigClient(ctx, format)
	}
	return cfg, nil
}
