package producer

import (
	"context"
	"fmt"
	"time"

	"github.com/wangshanqi84-gif/sagittarius/cores/logger"

	"github.com/apache/rocketmq-client-go/v2/primitive"
)

func LogInterceptor(lgr *logger.Logger) primitive.Interceptor {
	return func(ctx context.Context, req, reply interface{}, next primitive.Invoker) error {
		begin := time.Now()
		msg := req.(*primitive.Message)
		err := next(ctx, req, reply)
		if reply == nil {
			return err
		}
		result := reply.(*primitive.SendResult)
		cost := time.Since(begin).Nanoseconds() / int64(time.Millisecond)
		messageData := fmt.Sprintf("topic:%s, tags:%s, keys:%s, body:%s\n",
			msg.Topic, msg.GetTags(), msg.GetKeys(), string(msg.Body))

		lgr.Info(ctx, "[rocket] Send message, cost - [%d], status - [%d], data - {%s}",
			cost, result.Status, messageData)
		return err
	}
}
