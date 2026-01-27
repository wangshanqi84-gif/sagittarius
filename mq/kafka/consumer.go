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

type ConsumerOption func(*consumerOption)

type consumerOption struct {
	reBalance         []string
	offsetInitial     int64
	commitRetry       int
	maxWaitTime       time.Duration
	topicCreateEnable bool // 是否允许自动创建不存在的topic 生产环境不建议使用true
	autoCommit        bool
	builder           core.IMessageBuilder
	clientID          string
	workerNumbers     int
	seqNumbers        int
}

func ConsumerReBalance(reBalance []string) ConsumerOption {
	return func(o *consumerOption) {
		o.reBalance = reBalance
	}
}

func ConsumerOffsetInitial(offsetInitial int64) ConsumerOption {
	return func(o *consumerOption) {
		o.offsetInitial = offsetInitial
	}
}

func ConsumerMaxWaitTime(maxWaitTime time.Duration) ConsumerOption {
	return func(o *consumerOption) {
		o.maxWaitTime = maxWaitTime
	}
}

func ConsumerCommitRetry(commitRetry int) ConsumerOption {
	return func(o *consumerOption) {
		o.commitRetry = commitRetry
	}
}

func ConsumerTopicCreateEnable(topicCreateEnable bool) ConsumerOption {
	return func(o *consumerOption) {
		o.topicCreateEnable = topicCreateEnable
	}
}

func ConsumerAutoCommit(autoCommit bool) ConsumerOption {
	return func(o *consumerOption) {
		o.autoCommit = autoCommit
	}
}

func ConsumerMessageBuilder(builder core.IMessageBuilder) ConsumerOption {
	return func(o *consumerOption) {
		o.builder = builder
	}
}

func ConsumerClientID(clientID string) ConsumerOption {
	return func(o *consumerOption) {
		o.clientID = clientID
	}
}

func ConsumerWorkerNumbers(workerNumbers int) ConsumerOption {
	return func(o *consumerOption) {
		o.workerNumbers = workerNumbers
	}
}

func ConsumerSeqNumbers(seqNumbers int) ConsumerOption {
	return func(o *consumerOption) {
		o.seqNumbers = seqNumbers
	}
}

type Consumer struct {
	gc     *core.GroupConsumer
	topics map[string]string
}

func NewConsumer(ctx context.Context, groupName string, brokers []string, topics map[string]string, opts ...ConsumerOption) (*Consumer, error) {
	cfg := sarama.NewConfig()
	// 设置sarama日志
	sarama.Logger = log.New(os.Stdout, "[sarama] ", log.LstdFlags)
	// 设定版本
	cfg.Version = Version
	// 返回设置
	cfg.Consumer.Return.Errors = true
	// 其他设置
	option := consumerOption{
		offsetInitial:     sarama.OffsetNewest,
		commitRetry:       defaultConsumerCommitRetryTimes,
		reBalance:         []string{defaultConsumerRebalanceStrategy},
		maxWaitTime:       defaultConsumerMaxWaitTime,
		topicCreateEnable: false,
		workerNumbers:     defaultConsumerWorkerNumbers,
		seqNumbers:        defaultConsumerSeqNumbers,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&option)
		}
	}
	if len(option.clientID) > 0 {
		cfg.ClientID = option.clientID
	}
	if option.builder == nil {
		return nil, fmt.Errorf("kafka consumer, message builder is nil")
	}
	// 分区分配策略
	for _, rb := range option.reBalance {
		switch rb {
		case "range":
			cfg.Consumer.Group.Rebalance.GroupStrategies = append(cfg.Consumer.Group.Rebalance.GroupStrategies,
				sarama.NewBalanceStrategyRange())
		case "roundrobin":
			cfg.Consumer.Group.Rebalance.GroupStrategies = append(cfg.Consumer.Group.Rebalance.GroupStrategies,
				sarama.NewBalanceStrategyRoundRobin())
		default:
			cfg.Consumer.Group.Rebalance.GroupStrategies = append(cfg.Consumer.Group.Rebalance.GroupStrategies,
				sarama.NewBalanceStrategySticky())
		}
	}

	// 初始消费
	if option.offsetInitial != sarama.OffsetNewest && option.offsetInitial != sarama.OffsetOldest {
		option.offsetInitial = sarama.OffsetNewest
	}
	cfg.Consumer.Offsets.Retry.Max = option.commitRetry
	cfg.Consumer.Offsets.Initial = option.offsetInitial
	cfg.Consumer.MaxWaitTime = option.maxWaitTime
	cfg.Metadata.AllowAutoTopicCreation = option.topicCreateEnable
	if !option.autoCommit {
		cfg.Consumer.Offsets.AutoCommit.Enable = false
	}
	var topicList []string
	for _, v := range topics {
		topicList = append(topicList, v)
	}
	gc, err := core.NewGroupConsumer(ctx, cfg, groupName, brokers, option.topicCreateEnable,
		option.autoCommit, option.builder, topicList, option.workerNumbers, option.seqNumbers)
	if err != nil {
		return nil, err
	}
	c := &Consumer{
		gc:     gc,
		topics: make(map[string]string),
	}
	for k, v := range topics {
		c.topics[k] = v
	}
	return c, nil
}

func (c *Consumer) ConsumeTopics() []string {
	var res []string
	for _, v := range c.topics {
		res = append(res, v)
	}
	return res
}

func (c *Consumer) RegisterHandler(alias string, hs ...core.Handler) error {
	topic := alias
	if _, has := c.topics[alias]; has {
		topic = c.topics[alias]
	}
	return c.gc.RegisterHandler(topic, hs...)
}

func (c *Consumer) TopicWithAlias(alias string) string {
	if _, has := c.topics[alias]; !has {
		return ""
	}
	return c.topics[alias]
}

func (c *Consumer) TopicAlias() []string {
	var alias []string
	for k := range c.topics {
		alias = append(alias, k)
	}
	return alias
}

func (c *Consumer) Start() {
	c.gc.Start()
}
