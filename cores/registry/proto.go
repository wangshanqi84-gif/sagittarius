package registry

import "strings"

// 框架支持的监听/注册协议。每种协议在同一服务实例上最多一个端口。
const (
	ProtoHTTP      = "http"
	ProtoRPC       = "rpc"
	ProtoWebSocket = "websocket"
	ProtoSocketIO  = "socketio"
)

// SupportedProtos 可用于启动校验与配置校验。
var SupportedProtos = map[string]struct{}{
	ProtoHTTP:      {},
	ProtoRPC:       {},
	ProtoWebSocket: {},
	ProtoSocketIO:  {},
}

func NormalizeProto(proto string) string {
	return strings.ToLower(strings.TrimSpace(proto))
}

func IsSupportedProto(proto string) bool {
	_, ok := SupportedProtos[NormalizeProto(proto)]
	return ok
}
