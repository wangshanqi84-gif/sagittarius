package nacos

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"

	"github.com/wangshanqi84-gif/sagittarius/cores/logger"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

type Option func(*options)

type options struct {
	timeOut    uint64
	logger     *logger.Logger
	accessKey  string
	secretKey  string
	serverPath string
	runEnv     string
	userName   string
	password   string
}

// WithTimeOut client time out
func WithTimeOut(timeOut uint64) Option {
	return func(o *options) {
		o.timeOut = timeOut
	}
}

// WithLogger 日志
func WithLogger(logger *logger.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithAccessKey access-key 鉴权
func WithAccessKey(accessKey string) Option {
	return func(o *options) {
		o.accessKey = accessKey
	}
}

// WithSecretKey secret-key 鉴权
func WithSecretKey(secretKey string) Option {
	return func(o *options) {
		o.secretKey = secretKey
	}
}

// WithServerPath nacos server addr
func WithServerPath(path string) Option {
	return func(o *options) {
		o.serverPath = path
	}
}

// WithRunEnv current - env online or testing
func WithRunEnv(runEnv string) Option {
	return func(o *options) {
		o.runEnv = runEnv
	}
}

// WithUserName nacos username
func WithUserName(userName string) Option {
	return func(o *options) {
		o.userName = userName
	}
}

// WithPassword users password
func WithPassword(password string) Option {
	return func(o *options) {
		o.password = password
	}
}

func NewNamingClient(namespace string, product string, name string, opts ...Option) naming_client.INamingClient {
	o := &options{
		timeOut:  5000,
		userName: "nacos",
		password: "nacos",
	}
	for _, opt := range opts {
		opt(o)
	}
	if namespace == "" || product == "" || name == "" {
		panic("service undefined")
	}
	clientConfig := constant.ClientConfig{
		TimeoutMs:           o.timeOut,
		NamespaceId:         namespace,
		AppName:             fmt.Sprintf("%s.%s", product, name),
		NotLoadCacheAtStart: true,
	}
	if o.secretKey != "" {
		clientConfig.SecretKey = o.secretKey
	}
	if o.accessKey != "" {
		clientConfig.AccessKey = o.accessKey
	}
	if o.logger != nil {
		clientConfig.CustomLogger = &Logger{
			Logger: o.logger,
		}
	}
	if o.userName != "" {
		clientConfig.Username = o.userName
	}
	if o.password != "" {
		clientConfig.Password = o.password
	}
	us, err := url.Parse(o.serverPath)
	if err != nil {
		panic(err)
	}
	if us.Scheme == "" || us.Host == "" {
		panic("nacos-server config center path error")
	}
	ss := strings.Split(us.Host, ":")
	if len(ss) != 2 {
		panic("nacos-server config center host error")
	}
	addr := ss[0]
	port, err := strconv.ParseInt(ss[1], 10, 64)
	if err != nil {
		panic(err)
	}
	if us.Path == "" {
		us.Path = "/nacos"
	} else {
		us.Path = strings.TrimRight(us.Path, "/")
	}
	serverConfig := []constant.ServerConfig{
		{
			Scheme:      us.Scheme,
			IpAddr:      addr,
			Port:        uint64(port),
			ContextPath: us.Path,
		},
	}
	cli, err := clients.NewNamingClient(vo.NacosClientParam{
		ClientConfig:  &clientConfig,
		ServerConfigs: serverConfig,
	})
	if err != nil {
		panic(err)
	}
	return cli
}

func NewClient(namespace string, product string, name string, opts ...Option) config_client.IConfigClient {
	o := &options{
		timeOut:  5000,
		userName: "nacos",
		password: "nacos",
	}
	for _, opt := range opts {
		opt(o)
	}
	if namespace == "" || product == "" || name == "" {
		panic("service undefined")
	}
	clientConfig := constant.ClientConfig{
		TimeoutMs:           o.timeOut,
		NamespaceId:         namespace,
		AppName:             fmt.Sprintf("%s.%s", product, name),
		NotLoadCacheAtStart: true,
	}
	if o.secretKey != "" {
		clientConfig.SecretKey = o.secretKey
	}
	if o.accessKey != "" {
		clientConfig.AccessKey = o.accessKey
	}
	if o.logger != nil {
		clientConfig.CustomLogger = &Logger{
			Logger: o.logger,
		}
	}
	if o.userName != "" {
		clientConfig.Username = o.userName
	}
	if o.password != "" {
		clientConfig.Password = o.password
	}
	us, err := url.Parse(o.serverPath)
	if err != nil {
		panic(err)
	}
	if us.Scheme == "" || us.Host == "" {
		panic("nacos-server config center path error")
	}
	ss := strings.Split(us.Host, ":")
	if len(ss) != 2 {
		panic("nacos-server config center host error")
	}
	addr := ss[0]
	port, err := strconv.ParseInt(ss[1], 10, 64)
	if err != nil {
		panic(err)
	}
	if us.Path == "" {
		us.Path = "/nacos"
	} else {
		us.Path = strings.TrimRight(us.Path, "/")
	}
	serverConfig := []constant.ServerConfig{
		{
			Scheme:      us.Scheme,
			IpAddr:      addr,
			Port:        uint64(port),
			ContextPath: us.Path,
		},
	}
	cli, err := clients.NewConfigClient(vo.NacosClientParam{
		ClientConfig:  &clientConfig,
		ServerConfigs: serverConfig,
	})
	if err != nil {
		panic(err)
	}
	return cli
}
