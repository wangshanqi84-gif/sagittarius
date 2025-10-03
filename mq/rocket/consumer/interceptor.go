package consumer

import (
	"context"
	"fmt"
	"time"

	"github.com/wangshanqi84-gif/sagittarius/cores/logger"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

func LogInterceptor(lgr *logger.Logger) primitive.Interceptor {
	return func(ctx context.Context, req, reply interface{}, next primitive.Invoker) error {
		begin := time.Now()
		msgs := req.([]*primitive.MessageExt)
		err := next(ctx, req, reply)
		if reply == nil {
			return err
		}
		result := reply.(*consumer.ConsumeResultHolder)
		cost := time.Since(begin).Nanoseconds() / int64(time.Millisecond)
		var messageData string
		for _, msg := range msgs {
			messageData += fmt.Sprintf("msgId:%s, topic:%s, tags:%s, keys:%s, body:%s\n", msg.MsgId,
				msg.Topic, msg.GetTags(), msg.GetKeys(), string(msg.Body))
		}
		lgr.Info(ctx, "[rocket] consume message, cost - [%d], status - [%d], data - {\n%s}",
			cost, result.ConsumeResult, messageData)
		return err
	}
}
