package proxy

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/wangshanqi84-gif/sagittarius/app"
	"github.com/wangshanqi84-gif/sagittarius/app/config"
	httpClient "github.com/wangshanqi84-gif/sagittarius/cores/http/client"
	rpcClient "github.com/wangshanqi84-gif/sagittarius/cores/rpc/client"
	"github.com/wangshanqi84-gif/sagittarius/logger"
	"github.com/wangshanqi84-gif/sagittarius/mq/kafka"
	kfkCore "github.com/wangshanqi84-gif/sagittarius/mq/kafka/core"
	"github.com/wangshanqi84-gif/sagittarius/mq/rocket/consumer"
	"github.com/wangshanqi84-gif/sagittarius/mq/rocket/producer"
	"github.com/wangshanqi84-gif/sagittarius/mysql"
	"github.com/wangshanqi84-gif/sagittarius/redis"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/google/uuid"
	gPrometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

var (
	_sqlClient      = sync.Map{}
	_redisClient    = sync.Map{}
	_client         = sync.Map{}
	_rocketProducer = sync.Map{}
	_rocketConsumer = sync.Map{}
	_kafkaProducer  = sync.Map{}
	_kafkaConsumer  = sync.Map{}

	_sqlMutex            = sync.Mutex{}
	_redisMutex          = sync.Mutex{}
	_clientMutex         = sync.Mutex{}
	_rocketProducerMutex = sync.Mutex{}
	_rocketConsumerMutex = sync.Mutex{}
	_kafkaProducerMutex  = sync.Mutex{}
	_kafkaConsumerMutex  = sync.Mutex{}
)

// InitSqlClient 初始化mysql客户端
func InitSqlClient(name string, opts ...mysql.Option) (*mysql.Client, error) {
	_sqlMutex.Lock()
	defer _sqlMutex.Unlock()
	if c, has := _sqlClient.Load(name); has {
		return c.(*mysql.Client), nil
	}
	cfg := app.Router().Config().GetDatabase(name)
	if cfg == nil {
		return nil, errors.New(fmt.Sprintf("app init sql client, config is nil, name:%s", name))
	}
	if cfg.Master == "" {
		return nil, errors.New(fmt.Sprintf("app init sql client, master is nil, name:%s", name))
	}
	if cfg.MaxIdle > 0 {
		opts = append(opts, mysql.MaxIdle(cfg.MaxIdle))
	}
	if cfg.MaxOpen > 0 {
		opts = append(opts, mysql.MaxOpen(cfg.MaxOpen))
	}
	if cfg.MaxLifeTime != "" {
		td, err := time.ParseDuration(cfg.MaxLifeTime)
		if err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("app init sql client, config maxlifetime, value:%s", cfg.MaxLifeTime))
		}
		opts = append(opts, mysql.MaxLifeTime(td))
	}
	if cfg.MaxIdleTime != "" {
		td, err := time.ParseDuration(cfg.MaxIdleTime)
		if err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("app init sql client, config maxidletime, value:%s", cfg.MaxIdleTime))
		}
		opts = append(opts, mysql.MaxLifeTime(td))
	}
	opts = append(opts, mysql.Logger(logger.GetLogger()))
	c, err := mysql.NewClient(cfg.Master, cfg.Slaves, opts...)
	if err != nil {
		return nil, err
	}
	_sqlClient.Store(name, c)
	return c, nil
}

// InitRedisClient 初始化redis客户端
func InitRedisClient(name string, opts ...redis.Option) (*redis.Client, error) {
	_redisMutex.Lock()
	defer _redisMutex.Unlock()
	if c, has := _redisClient.Load(name); has {
		return c.(*redis.Client), nil
	}
	cfg := app.Router().Config().GetRedis(name)
	if cfg == nil {
		return nil, errors.New(fmt.Sprintf("app init redis client, config is nil, name:%s", name))
	}
	if cfg.Addr == "" {
		return nil, errors.New(fmt.Sprintf("app init redis client, addr is nil, name:%s", name))
	}
	addr := strings.Split(cfg.Addr, ",")
	opts = append(opts, redis.Addrs(addr), redis.Name(name))
	if cfg.Model != "" {
		opts = append(opts, redis.Model(cfg.Model))
	}
	if cfg.ReadTimeout > 0 {
		opts = append(opts, redis.ReadTimeout(cfg.ReadTimeout))
	}
	if cfg.WriteTimeout > 0 {
		opts = append(opts, redis.WriteTimeout(cfg.WriteTimeout))
	}
	if cfg.IdleTimeout > 0 {
		opts = append(opts, redis.IdleTimeout(cfg.IdleTimeout))
	}
	if cfg.DataBase >= 0 {
		opts = append(opts, redis.DB(cfg.DataBase))
	}
	if cfg.MaxRetry > 0 {
		opts = append(opts, redis.Retry(cfg.MaxRetry))
	}
	if cfg.PoolSize > 0 {
		opts = append(opts, redis.PoolSize(cfg.PoolSize))
	}
	if cfg.MinIdleConn > 0 {
		opts = append(opts, redis.MinIdleConn(cfg.MinIdleConn))
	}
	if cfg.Username != "" {
		opts = append(opts, redis.Username(cfg.Username))
	}
	if cfg.Password != "" {
		opts = append(opts, redis.Password(cfg.Password))
	}
	c, err := redis.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	_redisClient.Store(name, c)
	return c, nil
}

// InitRPCClient 初始化grpc client
func InitRPCClient(ctx context.Context, name string, opts ...rpcClient.Option) (*grpc.ClientConn, error) {
	_clientMutex.Lock()
	defer _clientMutex.Unlock()
	fullKey, cfg := app.Router().Config().GetClient(name, "rpc")
	if c, has := _client.Load(fullKey); has {
		return c.(*grpc.ClientConn), nil
	}
	if cfg == nil {
		return nil, errors.New(fmt.Sprintf("app init rpc client, config is nil, name:%s", name))
	}
	if cfg.UnUseDiscovery && len(strings.Split(cfg.EndPoints, ",")) == 0 {
		return nil, errors.New("client endpoints is nil")
	}
	if app.Router().Discovery() == nil && len(strings.Split(cfg.EndPoints, ",")) == 0 {
		return nil, errors.New("client endpoints is nil")
	}
	if cfg.EndPoints != "" {
		eps := strings.Split(cfg.EndPoints, ",")
		opts = append(opts, rpcClient.WithEps(eps...))
	}
	if !cfg.UnUseDiscovery && app.Router().Discovery() != nil {
		// 开始服务发现
		watcher, err := app.Router().Discovery().Watcher(app.Router().Ctx(), cfg.Namespace, cfg.Product, cfg.ServiceName, "rpc")
		if err != nil {
			return nil, err
		}
		opts = append(opts, rpcClient.WithWatcher(watcher))
	}
	var timeout time.Duration
	var err error
	if cfg.Timeout != "" {
		timeout, err = time.ParseDuration(cfg.Timeout)
		if err != nil {
			return nil, err
		}
	}
	opts = append(opts, rpcClient.WithUnaryInterceptor(
		rpcClient.RetryClientUnaryInterceptor(cfg.Retry),
		rpcClient.LangClientUnaryInterceptor(),
		rpcClient.TimeoutClientUnaryInterceptor(timeout),
		rpcClient.TracingClientUnaryInterceptor(app.Router().Ctx(), app.Router().Tracer()),
		gPrometheus.UnaryClientInterceptor),
	)
	c, err := rpcClient.DialContext(ctx, opts...)
	if err != nil {
		return nil, err
	}
	_client.Store(fullKey, c)
	return c, nil
}

func InitHttpClientUseConfig(ctx context.Context, cfg *config.ClientConfig) (*httpClient.Client, error) {
	_clientMutex.Lock()
	defer _clientMutex.Unlock()

	fullName := strings.TrimLeft(
		fmt.Sprintf("%s.%s.%s", cfg.Namespace, cfg.Product, cfg.ServiceName),
		".",
	)
	fullKey := fmt.Sprintf("%s-%s", fullName, "http")
	if c, has := _client.Load(fullKey); has {
		return c.(*httpClient.Client), nil
	}
	if cfg.UnUseDiscovery && len(strings.Split(cfg.EndPoints, ",")) == 0 {
		return nil, errors.New("client endpoints is nil")
	}
	if app.Router().Discovery() == nil && len(strings.Split(cfg.EndPoints, ",")) == 0 {
		return nil, errors.New("client endpoints is nil")
	}
	var opts []httpClient.Option
	if cfg.EndPoints != "" {
		eps := strings.Split(cfg.EndPoints, ",")
		opts = append(opts, httpClient.WithEps(eps...))
	}
	if cfg.Timeout == "" {
		cfg.Timeout = "5s"
	}
	td, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return nil, err
	}
	opts = append(opts, httpClient.WithTimeout(td))

	if !cfg.UnUseDiscovery && app.Router().Discovery() != nil {
		// 开始服务发现
		watcher, err := app.Router().Discovery().Watcher(app.Router().Ctx(), cfg.Namespace, cfg.Product, cfg.ServiceName, "http")
		if err != nil {
			return nil, err
		}
		opts = append(opts, httpClient.WithWatcher(watcher))
	}
	if cfg.Retry > 0 {
		opts = append(opts, httpClient.WithRetry(cfg.Retry))
	}
	opts = append(opts, httpClient.WithSyncTimeout(cfg.SyncTimeout))
	opts = append(opts, httpClient.WithInterceptors(
		httpClient.TracingInterceptor(app.Router().Ctx(), app.Router().Tracer()),
		httpClient.SyncTimeoutInterceptor(),
		httpClient.WithLangInterceptor(),
	))
	c := httpClient.NewClient(ctx, opts...)
	_client.Store(fullKey, c)
	return c, nil
}

// InitHttpClient 初始化http client
func InitHttpClient(ctx context.Context, name string, opts ...httpClient.Option) (*httpClient.Client, error) {
	_clientMutex.Lock()
	defer _clientMutex.Unlock()
	fullKey, cfg := app.Router().Config().GetClient(name, "http")
	if c, has := _client.Load(fullKey); has {
		return c.(*httpClient.Client), nil
	}
	if cfg == nil {
		return nil, errors.New(fmt.Sprintf("app init http client, config is nil, name:%s", name))
	}
	if cfg.UnUseDiscovery && len(strings.Split(cfg.EndPoints, ",")) == 0 {
		return nil, errors.New("client endpoints is nil")
	}
	if app.Router().Discovery() == nil && len(strings.Split(cfg.EndPoints, ",")) == 0 {
		return nil, errors.New("client endpoints is nil")
	}
	if cfg.EndPoints != "" {
		eps := strings.Split(cfg.EndPoints, ",")
		opts = append(opts, httpClient.WithEps(eps...))
	}
	if cfg.Timeout == "" {
		cfg.Timeout = "5s"
	}
	td, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return nil, err
	}
	opts = append(opts, httpClient.WithTimeout(td))
	if !cfg.UnUseDiscovery && app.Router().Discovery() != nil {
		// 开始服务发现
		watcher, err := app.Router().Discovery().Watcher(app.Router().Ctx(), cfg.Namespace, cfg.Product, cfg.ServiceName, "http")
		if err != nil {
			return nil, err
		}
		opts = append(opts, httpClient.WithWatcher(watcher))
	}
	if cfg.Retry > 0 {
		opts = append(opts, httpClient.WithRetry(cfg.Retry))
	}
	opts = append(opts, httpClient.WithSyncTimeout(cfg.SyncTimeout))
	opts = append(opts, httpClient.WithInterceptors(
		httpClient.TracingInterceptor(app.Router().Ctx(), app.Router().Tracer()),
		httpClient.SyncTimeoutInterceptor(),
		httpClient.WithLangInterceptor(),
	))
	c := httpClient.NewClient(ctx, opts...)
	_client.Store(fullKey, c)
	return c, nil
}

// InitRocketProducer 初始化rocket producer
func InitRocketProducer(ctx context.Context, name string, opts ...producer.Option) (*producer.Producer, error) {
	_rocketProducerMutex.Lock()
	defer _rocketProducerMutex.Unlock()
	if c, has := _rocketProducer.Load(name); has {
		return c.(*producer.Producer), nil
	}
	cfg := app.Router().Config().RocketProducer(name)
	if cfg == nil {
		return nil, errors.New(fmt.Sprintf("app init rocket producer, config is nil, name:%s", name))
	}
	if cfg.Brokers == "" {
		return nil, errors.New("rocket brokers is nil")
	}
	if len(cfg.Topic) == 0 {
		topics := make(map[string]string)
		for _, topic := range cfg.Topic {
			topics[topic.Alias] = topic.Name
		}
		opts = append(opts, producer.WithTopics(topics))
	}
	bks := strings.Split(cfg.Brokers, ",")
	opts = append(opts, producer.WithNameServer(bks))
	if cfg.Timeout != "" {
		cfg.Timeout = "5s"
	}
	td, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return nil, err
	}
	opts = append(opts, producer.WithTimeout(td))
	if cfg.MaxRetry == 0 {
		cfg.MaxRetry = 2
	}
	opts = append(opts, producer.WithRetry(cfg.MaxRetry))
	if cfg.AccessKey != "" || cfg.SecretKey != "" || cfg.SecurityToken != "" {
		opts = append(opts, producer.WithCredentials(primitive.Credentials{
			AccessKey:     cfg.AccessKey,
			SecretKey:     cfg.SecretKey,
			SecurityToken: cfg.SecurityToken,
		}))
	}
	opts = append(opts,
		producer.WithTracer(app.Router().Tracer()),
		producer.WithInterceptors([]primitive.Interceptor{producer.LogInterceptor(logger.GetLogger())}),
	)
	p, err := producer.NewProducer(ctx, opts...)
	if err != nil {
		return nil, err
	}
	_rocketProducer.Store(name, p)
	return p, nil
}

// InitRocketConsumer 初始化rocket consumer
func InitRocketConsumer(ctx context.Context, name string, opts ...consumer.Option) (*consumer.PushConsumer, error) {
	_rocketConsumerMutex.Lock()
	defer _rocketConsumerMutex.Unlock()
	if c, has := _rocketConsumer.Load(name); has {
		return c.(*consumer.PushConsumer), nil
	}
	cfg := app.Router().Config().RocketConsumer(name)
	if cfg == nil {
		return nil, errors.New(fmt.Sprintf("app init rocket consumer, config is nil, name:%s", name))
	}
	if cfg.Brokers == "" {
		return nil, errors.New("rocket brokers is nil")
	}
	bks := strings.Split(cfg.Brokers, ",")
	opts = append(opts, consumer.WithNameServer(bks))
	if cfg.ConsumeTimeout == "" {
		cfg.ConsumeTimeout = "30m"
	}
	td, err := time.ParseDuration(cfg.ConsumeTimeout)
	if err != nil {
		return nil, err
	}
	opts = append(opts, consumer.WithConsumeTimeout(td))
	if cfg.MaxRetry == 0 {
		cfg.MaxRetry = 2
	}
	opts = append(opts, consumer.WithRetry(cfg.MaxRetry))
	if cfg.AccessKey != "" || cfg.SecretKey != "" || cfg.SecurityToken != "" {
		opts = append(opts, consumer.WithCredentials(primitive.Credentials{
			AccessKey:     cfg.AccessKey,
			SecretKey:     cfg.SecretKey,
			SecurityToken: cfg.SecurityToken,
		}))
	}
	if cfg.MaxReconsumeTimes > 0 {
		opts = append(opts, consumer.WithMaxReconsumeTimes(cfg.MaxReconsumeTimes))
	}
	if cfg.Expression == "" {
		cfg.Expression = "*"
	}
	opts = append(opts,
		consumer.WithTracer(app.Router().Tracer()),
		consumer.WithInterceptors([]primitive.Interceptor{consumer.LogInterceptor(logger.GetLogger())}),
		consumer.WithFrom(cfg.From),
		consumer.WithGoroutineNums(runtime.NumCPU()*5),
		consumer.WithGroupName(cfg.GroupName),
		consumer.WithModel(cfg.Mode),
		consumer.WithTagExpression(cfg.Expression),
	)
	con, err := consumer.NewPushConsumer(ctx, opts...)
	if err != nil {
		return nil, err
	}
	_rocketConsumer.Store(name, con)
	return con, nil
}

// InitKafkaProducer 初始化kafka生产者
func InitKafkaProducer(name string, opts ...kafka.ProducerOption) (*kafka.Producer, error) {
	_kafkaProducerMutex.Lock()
	defer _kafkaProducerMutex.Unlock()
	if c, has := _kafkaProducer.Load(name); has {
		return c.(*kafka.Producer), nil
	}
	cfg := app.Router().Config().KafkaProducer(name)
	if cfg == nil {
		return nil, errors.New(fmt.Sprintf("app init kafka producer, config is nil, name:%s", name))
	}
	if cfg.Brokers == "" {
		return nil, errors.New(fmt.Sprintf("app init kafka producer, brokers is nil, name:%s", name))
	}
	if len(cfg.Topic) > 0 {
		topics := make(map[string]string)
		for _, topic := range cfg.Topic {
			topics[topic.Alias] = topic.Name
		}
		opts = append(opts, kafka.Topics(topics))
	}
	t := app.Router().Tracer()
	if t == nil {
		return nil, errors.New("app init kafka producer, tracer is nil")
	}
	opts = append(opts, kafka.ProducerMessageBuilder(kfkCore.NewMessageBuilder(t)))
	opts = append(opts, kafka.ProducerNotifyDisable(cfg.DisableNotify))
	brokers := strings.Split(cfg.Brokers, ",")
	if cfg.Timeout != "" {
		td, err := time.ParseDuration(cfg.Timeout)
		if err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("app init kafka producer, config timeout, value:%s", cfg.Timeout))
		}
		opts = append(opts, kafka.ProducerTimeout(td))
	}
	if cfg.MaxMessageBytes > 0 {
		opts = append(opts, kafka.ProducerMaxMessageBytes(cfg.MaxMessageBytes))
	}
	if cfg.MaxRetry > 0 {
		opts = append(opts, kafka.ProducerRetry(cfg.MaxRetry))
	}
	if cfg.Mode != "" {
		opts = append(opts, kafka.ProducerModel(cfg.Mode))
	}
	clientID := app.Router().Service().ServiceName + ":" + app.Router().Service().ID
	opts = append(opts, kafka.ProducerClientID(clientID))
	p, err := kafka.NewProducer(app.Router().Ctx(), brokers, opts...)
	if err != nil {
		return nil, err
	}
	_kafkaProducer.Store(name, p)
	return p, nil
}

// InitKafkaConsumer 初始化kafka消费者
func InitKafkaConsumer(name string, opts ...kafka.ConsumerOption) (*kafka.Consumer, error) {
	_kafkaConsumerMutex.Lock()
	defer _kafkaConsumerMutex.Unlock()
	if c, has := _kafkaConsumer.Load(name); has {
		return c.(*kafka.Consumer), nil
	}
	cfg := app.Router().Config().KafkaConsumer(name)
	if cfg == nil {
		return nil, errors.New(fmt.Sprintf("app init kafka consumer, config is nil, name:%s", name))
	}
	if cfg.Brokers == "" {
		return nil, errors.New(fmt.Sprintf("app init kafka consumer, brokers is nil, name:%s", name))
	}
	if len(cfg.Topics) == 0 {
		return nil, errors.New(fmt.Sprintf("app init kafka consumer, topics is nil, name:%s", name))
	}
	if cfg.Group == "" {
		// 这里改为当Group为空时 为服务生成一个随机唯一的Group
		cfg.Group = fmt.Sprintf("%s-%s", app.Router().Service().ServiceName, uuid.New().String())
	}
	t := app.Router().Tracer()
	if t == nil {
		return nil, errors.New("app init kafka consumer, tracer is nil")
	}
	opts = append(opts, kafka.ConsumerMessageBuilder(kfkCore.NewMessageBuilder(t)))
	brokers := strings.Split(cfg.Brokers, ",")
	if cfg.MaxWaitTime != "" {
		td, err := time.ParseDuration(cfg.MaxWaitTime)
		if err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("app init kafka consumer, config maxwaittime, value:%s", cfg.MaxWaitTime))
		}
		opts = append(opts, kafka.ConsumerMaxWaitTime(td))
	}
	if cfg.CommitRetry > 0 {
		opts = append(opts, kafka.ConsumerCommitRetry(cfg.CommitRetry))
	}
	if cfg.OffsetInitial != 0 {
		opts = append(opts, kafka.ConsumerOffsetInitial(cfg.OffsetInitial))
	}
	if cfg.ReBalance != "" {
		rbs := strings.Split(cfg.ReBalance, ",")
		if len(rbs) > 0 && rbs[0] != "" {
			opts = append(opts, kafka.ConsumerReBalance(rbs))
		}
	}
	if cfg.TopicCreateEnable {
		opts = append(opts, kafka.ConsumerTopicCreateEnable(cfg.TopicCreateEnable))
	}
	if !cfg.AutoCommit {
		opts = append(opts, kafka.ConsumerAutoCommit(cfg.AutoCommit))
	}
	clientID := app.Router().Service().ServiceName + ":" + app.Router().Service().ID
	opts = append(opts, kafka.ConsumerClientID(clientID))
	topicMap := make(map[string]string)
	for _, topic := range cfg.Topics {
		topicMap[topic.Alias] = topic.Name
	}
	c, err := kafka.NewConsumer(app.Router().Ctx(), cfg.Group, brokers, topicMap, opts...)
	if err != nil {
		return nil, err
	}
	_kafkaConsumer.Store(name, c)
	return c, nil
}
