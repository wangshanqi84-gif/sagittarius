package server

import (
	"context"
	"crypto/tls"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type Option func(o *Server)

func Network(network string) Option {
	return func(s *Server) {
		s.network = network
	}
}

func Address(addr string) Option {
	return func(s *Server) {
		s.address = addr
	}
}

func TLS(tlsCfg *tls.Config) Option {
	return func(s *Server) {
		s.tlsCfg = tlsCfg
	}
}

func UnaryInterceptor(in ...grpc.UnaryServerInterceptor) Option {
	return func(s *Server) {
		s.unaryInts = in
	}
}

func StreamInterceptor(in ...grpc.StreamServerInterceptor) Option {
	return func(s *Server) {
		s.streamInts = in
	}
}

func Options(opts ...grpc.ServerOption) Option {
	return func(s *Server) {
		s.opts = opts
	}
}

func OnStop(f func()) Option {
	return func(s *Server) {
		s.onStop = f
	}
}

type Server struct {
	*grpc.Server
	network    string
	address    string
	tlsCfg     *tls.Config
	health     *health.Server
	opts       []grpc.ServerOption
	unaryInts  []grpc.UnaryServerInterceptor
	streamInts []grpc.StreamServerInterceptor
	onStop     func()
}

func NewServer(opts ...Option) *Server {
	srv := &Server{
		network: "tcp",
		address: ":9901",
		health:  health.NewServer(),
	}
	for _, o := range opts {
		o(srv)
	}
	grpcOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(srv.unaryInts...),
		grpc.ChainStreamInterceptor(srv.streamInts...),
	}
	if srv.tlsCfg != nil {
		grpcOpts = append(grpcOpts, grpc.Creds(credentials.NewTLS(srv.tlsCfg)))
	}
	if len(srv.opts) > 0 {
		grpcOpts = append(grpcOpts, srv.opts...)
	}
	srv.Server = grpc.NewServer(grpcOpts...)
	// 健康检查
	grpc_health_v1.RegisterHealthServer(srv.Server, srv.health)
	// 反射注册
	reflection.Register(srv.Server)
	return srv
}

func (s *Server) Start(ctx context.Context) error {
	sock, err := net.Listen(s.network, s.address)
	if err != nil {
		return err
	}
	s.health.Resume()
	return s.Serve(sock)
}

func (s *Server) Stop(_ context.Context) error {
	if s.onStop != nil {
		s.onStop()
	}
	// 健康检查关闭
	s.health.Shutdown()
	// 优雅关闭
	s.GracefulStop()
	return nil
}
