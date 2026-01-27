package core

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/IBM/sarama"
	"github.com/pkg/errors"
)

type Handler func(ctx context.Context, msg *ConsumerMessage)

type GroupConsumer struct {
	ctx           context.Context
	topics        []string
	autoCommit    bool
	builder       IMessageBuilder
	group         sarama.ConsumerGroup
	kafkaVer      sarama.KafkaVersion
	handlers      map[string][]Handler
	workerNumbers int
	workerCh      chan *ConsumerMessage
}

func (gc *GroupConsumer) Setup(sess sarama.ConsumerGroupSession) error {
	return nil
}

func (gc *GroupConsumer) Cleanup(sess sarama.ConsumerGroupSession) error {
	return nil
}

func (gc *GroupConsumer) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			cm := gc.builder.ConsumerMessage(gc.ctx, msg, gc.kafkaVer)
			// 这里阻塞写入chan 因此为了效率 消费者应该异步处理
			cm.sess = sess
			gc.workerCh <- cm
			if gc.autoCommit {
				// 标记
				// sarama会自动进行提交 默认间隔1秒
				sess.MarkMessage(msg, "")
			}
		case <-sess.Context().Done():
			// 关闭时进行提交
			sess.Commit()
			return nil
		}
	}
}

func (gc *GroupConsumer) RegisterHandler(topic string, hs ...Handler) error {
	if _, has := gc.handlers[topic]; has {
		return errors.New("topic already registered")
	}
	has := false
	for _, t := range gc.topics {
		if topic == t {
			has = true
			break
		}
	}
	if !has {
		return errors.New("topic no monitoring")
	}
	gc.handlers[topic] = append(gc.handlers[topic], hs...)
	return nil
}

func (gc *GroupConsumer) start(topics []string, handler sarama.ConsumerGroupHandler) {
	go func() {
		for {
			select {
			case <-gc.ctx.Done():
				return
			default:
			}
			err := gc.group.Consume(gc.ctx, topics, handler)
			if err != nil {
				switch err {
				case sarama.ErrClosedClient, sarama.ErrClosedConsumerGroup:
					// 退出
					return
				default:
					log.Println("kafka consumer error:", err)
				}
			}
		}
	}()
	go func() {
		defer gc.group.Close()

		for {
			select {
			case err := <-gc.group.Errors():
				log.Println("kafka consumer group error:", err)
			case <-gc.ctx.Done():
				return
			}
		}
	}()
}

func NewGroupConsumer(
	ctx context.Context,
	cfg *sarama.Config,
	groupName string,
	brokers []string,
	topicCreateEnable bool,
	autoCommit bool,
	builder IMessageBuilder,
	topics []string,
	workerNumbers int,
	seqNumbers int) (*GroupConsumer, error) {
	// 初始化client
	c, err := sarama.NewClient(brokers, cfg)
	if err != nil {
		return nil, err
	}
	// topic检查
	if !topicCreateEnable {
		partitionTopics, err := c.Topics()
		if err != nil {
			return nil, err
		}
		// 开始检查
		var needCreate []string
		for _, ct := range topics {
			has := false
			for _, pt := range partitionTopics {
				if ct != pt {
					continue
				}
				has = true
				break
			}
			if !has {
				needCreate = append(needCreate, ct)
			}
		}
		// 判断结果
		if len(needCreate) > 0 {
			return nil, fmt.Errorf("kafka topic not find, topics:%s", strings.Join(needCreate, ","))
		}
	}
	// 根据client创建consumerGroup
	group, err := sarama.NewConsumerGroupFromClient(groupName, c)
	if err != nil {
		return nil, err
	}
	gc := GroupConsumer{
		ctx:           ctx,
		topics:        topics,
		autoCommit:    autoCommit,
		builder:       builder,
		group:         group,
		kafkaVer:      cfg.Version,
		workerNumbers: workerNumbers,
		handlers:      make(map[string][]Handler),
	}
	if seqNumbers > 0 {
		gc.workerCh = make(chan *ConsumerMessage, seqNumbers)
	} else {
		gc.workerCh = make(chan *ConsumerMessage)
	}
	// 启动worker
	for i := 0; i < gc.workerNumbers; i++ {
		go func() {
			for {
				select {
				case msg, ok := <-gc.workerCh:
					if !ok {
						return
					}
					if _, has := gc.handlers[msg.Topic()]; has {
						for _, hd := range gc.handlers[msg.Topic()] {
							hd(msg.Ctx(), msg)
						}
					}
				case <-gc.ctx.Done():
					return
				}
			}
		}()
	}
	return &gc, nil
}

func (gc *GroupConsumer) Start() {
	gc.start(gc.topics, gc)
}
