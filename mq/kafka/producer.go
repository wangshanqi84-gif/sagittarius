package kafka

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/wangshanqi84-gif/sagittarius/mq/kafka/core"

	"github.com/IBM/sarama"
)

type ErrHandler func(err error)
type SuccessHandler func(message *sarama.ProducerMessage)

type ProducerOption func(*producerOption)

type producerOption struct {
	timeout         time.Duration
	maxMessageBytes int
	retry           int
	ints            []sarama.ProducerInterceptor
	model           string
	builder         core.IMessageBuilder
	clientID        string
	topics          map[string]string // K-别名 V-topic名
	notifyDisable   bool
}

func ProducerTimeout(timeout time.Duration) ProducerOption {
	return func(o *producerOption) {
		o.timeout = timeout
	}
}

func ProducerMaxMessageBytes(maxMessageBytes int) ProducerOption {
	return func(o *producerOption) {
		o.maxMessageBytes = maxMessageBytes
	}
}

func ProducerRetry(retry int) ProducerOption {
	return func(o *producerOption) {
		o.retry = retry
	}
}

func ProducerInterceptor(ints ...sarama.ProducerInterceptor) ProducerOption {
	return func(o *producerOption) {
		o.ints = append(o.ints, ints...)
	}
}

func ProducerMessageBuilder(builder core.IMessageBuilder) ProducerOption {
	return func(o *producerOption) {
		o.builder = builder
	}
}

func ProducerModel(model string) ProducerOption {
	return func(o *producerOption) {
		o.model = model
	}
}

func ProducerNotifyDisable(notifyDisable bool) ProducerOption {
	return func(o *producerOption) {
		o.notifyDisable = notifyDisable
	}
}

func ProducerClientID(clientID string) ProducerOption {
	return func(o *producerOption) {
		o.clientID = clientID
	}
}

func Topics(topics map[string]string) ProducerOption {
	return func(o *producerOption) {
		o.topics = make(map[string]string)
		for k, v := range topics {
			o.topics[k] = v
		}
	}
}

type Producer struct {
	core.IProducer
	Version sarama.KafkaVersion
}

func NewProducer(ctx context.Context, brokers []string, opts ...ProducerOption) (*Producer, error) {
	cfg := sarama.NewConfig()
	// 设置sarama日志
	sarama.Logger = log.New(os.Stdout, "[sarama] ", log.LstdFlags)
	// 设定版本
	cfg.Version = Version
	// 幂等参数设定 强制要求
	if cfg.Version.IsAtLeast(sarama.V0_11_0_0) {
		cfg.Producer.Idempotent = true
	}
	cfg.Net.MaxOpenRequests = 1
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	// 返回设置
	cfg.Producer.Return.Successes = true
	cfg.Producer.Return.Errors = true
	// option设置
	option := producerOption{
		retry:           defaultProducerRetryTimes,
		maxMessageBytes: defaultProducerMaxMessageBytes,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&option)
		}
	}
	if option.builder == nil {
		return nil, fmt.Errorf("producer message builder is nil")
	}
	if option.retry > 0 {
		cfg.Producer.Retry.Max = option.retry
	}
	if option.maxMessageBytes > 0 {
		cfg.Producer.MaxMessageBytes = option.maxMessageBytes
	}
	if option.timeout > 0 {
		cfg.Producer.Timeout = option.timeout
	}
	if len(option.ints) > 0 {
		cfg.Producer.Interceptors = option.ints
	}
	if len(option.clientID) > 0 {
		cfg.ClientID = option.clientID
	}
	var producer core.IProducer
	var err error
	switch option.model {
	case "async":
		producer, err = core.NewAsyncProducer(ctx, brokers, option.builder, option.topics, cfg)
	default:
		producer, err = core.NewSyncProducer(ctx, brokers, option.builder, option.topics, cfg)
	}
	if err != nil {
		return nil, err
	}
	if option.notifyDisable {
		go func(p core.IProducer) {
			for {
				select {
				case <-p.Error():
				case <-p.Success():
				case <-ctx.Done():
					return
				}
			}
		}(producer)
	}
	return &Producer{
		IProducer: producer,
		Version:   Version,
	}, nil
}
