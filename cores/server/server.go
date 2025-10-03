package server

import "context"

///////////////////////////////
// 服务接口定义
///////////////////////////////

type Server interface {
	Start(context.Context) error
	Stop(context.Context) error
}
