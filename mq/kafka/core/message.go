package core

import (
	"context"

	"github.com/IBM/sarama"
)

///////////////////////////////////
// 生产者消息
///////////////////////////////////

type ProducerMessage struct {
	ctx context.Context
	msg *sarama.ProducerMessage
}

func (pm *ProducerMessage) Ctx() context.Context {
	return pm.ctx
}

func (pm *ProducerMessage) Topic() string {
	return pm.msg.Topic
}

func (pm *ProducerMessage) AddHeader(key, val string) {
	pm.msg.Headers = append(pm.msg.Headers, sarama.RecordHeader{
		Key:   []byte(key),
		Value: []byte(val),
	})
}

func (pm *ProducerMessage) SetMetaData(metadata interface{}) {
	pm.msg.Metadata = metadata
}

///////////////////////////////////
// 消费者消息
///////////////////////////////////

type ConsumerMessage struct {
	ctx  context.Context
	sess sarama.ConsumerGroupSession
	msg  *sarama.ConsumerMessage
}

func (cm *ConsumerMessage) Ctx() context.Context {
	return cm.ctx
}

func (cm *ConsumerMessage) SetCtx(ctx context.Context) {
	cm.ctx = ctx
}

func (cm *ConsumerMessage) Topic() string {
	return cm.msg.Topic
}

func (cm *ConsumerMessage) Value() []byte {
	return cm.msg.Value
}

func (cm *ConsumerMessage) Offset() int64 {
	return cm.msg.Offset
}

func (cm *ConsumerMessage) Commit() {
	cm.sess.MarkMessage(cm.msg, "")
	cm.sess.Commit()
}

func (cm *ConsumerMessage) Header() []*sarama.RecordHeader {
	return cm.msg.Headers
}
