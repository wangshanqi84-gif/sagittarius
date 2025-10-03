package kafka

import (
	"time"

	"github.com/IBM/sarama"
)

/////////////////////////////////////
// 生产者常量定义
/////////////////////////////////////

const (
	defaultProducerRetryTimes      = 3
	defaultProducerMaxMessageBytes = 10000000
)

var (
	Version = sarama.V3_4_0_0
)

/////////////////////////////////////
// 消费者常量定义
/////////////////////////////////////

const (
	defaultConsumerCommitRetryTimes  = 3
	defaultConsumerMaxWaitTime       = 100 * time.Millisecond
	defaultConsumerRebalanceStrategy = "sticky"
)
