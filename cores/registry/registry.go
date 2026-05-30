package registry

import (
	"context"
)

// Service 服务发现信息
type Service struct {
	ID          string `json:"id"`          // udid启动生成
	Namespace   string `json:"namespace"`   // 服务所属明明空间
	Product     string `json:"product"`     // 服务所属产品
	ServiceName string `json:"serviceName"` // 服务名称
	// Hosts 按协议维度的注册地址，key 为 proto（http/rpc/websocket/socketio）。
	// 设计约束：每种协议至多一个 endpoint；多协议可并存（如 http + websocket）。
	Hosts    map[string]string `json:"hosts"`
	Tags     string            `json:"tags"`
	Metadata map[string]string `json:"metadata"` // 元数据
}

// Endpoint 返回指定协议的注册地址。
func (s *Service) Endpoint(proto string) (string, bool) {
	if s == nil || len(s.Hosts) == 0 {
		return "", false
	}
	addr, ok := s.Hosts[NormalizeProto(proto)]
	return addr, ok && addr != ""
}

/////////////////////////////////////////
// 服务发现 实现接口即可支持多种服务发现中间件
// v1 : etcd, consul
/////////////////////////////////////////

// Watcher 服务发现接口
type Watcher interface {
	Start() ([]*Service, error)
	Stop() error
}

// Discovery 服务注册/发现接口
type Discovery interface {
	Register(ctx context.Context, service *Service) error
	Deregister(ctx context.Context, service *Service) error
	Stop(ctx context.Context, service *Service) error
	Watcher(ctx context.Context, namespace string, product string, serviceName string, proto string) (Watcher, error)
}
