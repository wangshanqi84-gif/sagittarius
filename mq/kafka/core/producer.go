package core

import (
	"context"

	"github.com/IBM/sarama"
	"github.com/pkg/errors"
)

/////////////////////////////////////////
// 生产者相关
/////////////////////////////////////////

var (
	_asyncChanSize = 256
)

// 生产者接口

type MessageData struct {
	Key  []byte
	Data []byte
}

type IProducer interface {
	SendMessage(ctx context.Context, alias string, key []byte, data []byte)
	SendMessages(ctx context.Context, alias string, md []*MessageData)
	Success() chan *sarama.ProducerMessage
	Error() chan *sarama.ProducerError
}

// 同步生产者相关

type SyncProducer struct {
	ctx      context.Context
	builder  IMessageBuilder
	topics   map[string]string
	sp       sarama.SyncProducer
	kafkaVer sarama.KafkaVersion
	errChan  chan *sarama.ProducerError
	succChan chan *sarama.ProducerMessage
}

func NewSyncProducer(ctx context.Context, brokers []string, builder IMessageBuilder,
	topics map[string]string, cfg *sarama.Config) (*SyncProducer, error) {
	// 初始化同步生产者
	p, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "new sync producer")
	}
	sp := SyncProducer{
		ctx:      ctx,
		sp:       p,
		kafkaVer: cfg.Version,
		builder:  builder,
		topics:   make(map[string]string),
		errChan:  make(chan *sarama.ProducerError, 1),
		succChan: make(chan *sarama.ProducerMessage, 1),
	}
	for k, v := range topics {
		sp.topics[k] = v
	}
	go func(producer *SyncProducer) {
		select {
		case <-producer.ctx.Done():
			producer.sp.Close()
		}
	}(&sp)
	return &sp, nil
}

func (sp *SyncProducer) SendMessage(ctx context.Context, alias string, key []byte, data []byte) {
	// 构建消息
	topic := alias
	if _, has := sp.topics[alias]; has {
		topic = sp.topics[alias]
	}
	msg := sp.builder.ProducerMessage(ctx, topic, key, data, sp.kafkaVer)
	_, _, err := sp.sp.SendMessage(msg.msg)
	if err != nil {
		sp.errChan <- &sarama.ProducerError{Msg: msg.msg, Err: err}
	} else {
		sp.succChan <- msg.msg
	}
}

func (sp *SyncProducer) SendMessages(ctx context.Context, alias string, md []*MessageData) {
	// 构建消息
	topic := alias
	if _, has := sp.topics[alias]; has {
		topic = sp.topics[alias]
	}
	var msgs []*sarama.ProducerMessage
	for _, data := range md {
		msg := sp.builder.ProducerMessage(ctx, topic, data.Key, data.Data, sp.kafkaVer)
		msgs = append(msgs, msg.msg)
	}
	if err := sp.sp.SendMessages(msgs); err != nil {
		sp.errChan <- &sarama.ProducerError{Msg: msgs[0], Err: err}
	} else {
		sp.succChan <- msgs[0]
	}
}

func (sp *SyncProducer) Success() chan *sarama.ProducerMessage {
	return sp.succChan
}

func (sp *SyncProducer) Error() chan *sarama.ProducerError {
	return sp.errChan
}

// 异步生产者

type AsyncProducer struct {
	ctx      context.Context
	builder  IMessageBuilder
	topics   map[string]string
	ap       sarama.AsyncProducer
	kafkaVer sarama.KafkaVersion
	errChan  chan *sarama.ProducerError
	succChan chan *sarama.ProducerMessage
}

func NewAsyncProducer(ctx context.Context, brokers []string, builder IMessageBuilder,
	topics map[string]string, cfg *sarama.Config) (*AsyncProducer, error) {
	// 初始化异步生产者
	p, err := sarama.NewAsyncProducer(brokers, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "new async producer")
	}
	ap := AsyncProducer{
		ctx:      ctx,
		builder:  builder,
		ap:       p,
		kafkaVer: cfg.Version,
		topics:   make(map[string]string),
		errChan:  make(chan *sarama.ProducerError, _asyncChanSize),
		succChan: make(chan *sarama.ProducerMessage, _asyncChanSize),
	}
	for k, v := range topics {
		ap.topics[k] = v
	}
	// 必要监听
	go func(producer *AsyncProducer) {
		for {
			select {
			case e := <-producer.ap.Errors():
				producer.errChan <- e
			case msg := <-producer.ap.Successes():
				producer.succChan <- msg
			case <-producer.ctx.Done():
				producer.ap.Close()
				return
			}
		}
	}(&ap)
	return &ap, nil
}

func (ap *AsyncProducer) SendMessage(ctx context.Context, alias string, key []byte, data []byte) {
	// 构建消息
	topic := alias
	if _, has := ap.topics[alias]; has {
		topic = ap.topics[alias]
	}
	msg := ap.builder.ProducerMessage(ctx, topic, key, data, ap.kafkaVer)
	ap.ap.Input() <- msg.msg
}

func (ap *AsyncProducer) SendMessages(ctx context.Context, alias string, md []*MessageData) {
	// 构建消息
	topic := alias
	if _, has := ap.topics[alias]; has {
		topic = ap.topics[alias]
	}
	var msgs []*sarama.ProducerMessage
	for _, data := range md {
		msg := ap.builder.ProducerMessage(ctx, topic, data.Key, data.Data, ap.kafkaVer)
		msgs = append(msgs, msg.msg)
	}
	for _, msg := range msgs {
		ap.ap.Input() <- msg
	}
}

func (ap *AsyncProducer) Success() chan *sarama.ProducerMessage {
	return ap.succChan
}

func (ap *AsyncProducer) Error() chan *sarama.ProducerError {
	return ap.errChan
}
