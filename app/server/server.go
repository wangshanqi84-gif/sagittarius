package server

import (
	"fmt"
	netHttp "net/http"
	"strings"

	"github.com/wangshanqi84-gif/sagittarius/app"
	"github.com/wangshanqi84-gif/sagittarius/app/config"
	httpSrv "github.com/wangshanqi84-gif/sagittarius/cores/http/server"
	"github.com/wangshanqi84-gif/sagittarius/cores/registry"
	rpcSrv "github.com/wangshanqi84-gif/sagittarius/cores/rpc/server"
	ioSrv "github.com/wangshanqi84-gif/sagittarius/cores/socketio/server"
	wsSrv "github.com/wangshanqi84-gif/sagittarius/cores/websocket/server"
	"github.com/wangshanqi84-gif/sagittarius/logger"

	grpcPrometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

func mustServerPort(cfg *config.ServiceConfig, proto string) int {
	svr, err := cfg.ServerByProto(proto)
	if err != nil {
		panic(err.Error())
	}
	return svr.Port
}

func InitRPCServer(opts ...rpcSrv.Option) (*rpcSrv.Server, error) {
	cfg, err := app.Router().Config()
	if err != nil {
		return nil, err
	}
	// 初始化server
	// 找到rpc配置
	port := mustServerPort(cfg, registry.ProtoRPC)
	opts = append(opts, rpcSrv.Address(fmt.Sprintf(":%d", port)))
	opts = append(opts, rpcSrv.UnaryInterceptor(
		rpcSrv.RecoverServerInterceptor(logger.GetLogger()),
		rpcSrv.LangServerUnaryInterceptor(),
		grpcPrometheus.UnaryServerInterceptor,
		rpcSrv.TracingServerUnaryInterceptor(app.Router().Tracer()),
		rpcSrv.AccessServerUnaryInterceptor(logger.GetAccess(), !cfg.AccessRequestDisable),
	))
	opts = append(opts, rpcSrv.Options([]grpc.ServerOption{
		grpc.MaxRecvMsgSize(1024 * 1024 * 16),
	}...))
	srv := rpcSrv.NewServer(opts...)
	grpcPrometheus.EnableHandlingTimeHistogram()
	grpcPrometheus.Register(srv.Server)

	go func() {
		err := netHttp.ListenAndServe(fmt.Sprintf(":%d", port+1), promhttp.Handler())
		if err != nil {
			logger.Gen(app.Router().Ctx(), "init rpc server metrics error:%v", err)
		}
	}()
	return srv, nil
}

func InitWebSocketServer(opts ...wsSrv.Option) (*wsSrv.Engine, error) {
	cfg, err := app.Router().Config()
	if err != nil {
		return nil, err
	}
	// 初始化server
	// 找到websocket配置
	opts = append(opts, wsSrv.Port(fmt.Sprintf("%d", mustServerPort(cfg, registry.ProtoWebSocket))))
	// 初始化server
	var options []wsSrv.Option
	path := fmt.Sprintf("/%s/%s/ws", app.Router().Service().Product,
		strings.Join(strings.Split(app.Router().Service().ServiceName, "."), "/"))
	options = append(options, wsSrv.WsPath(path))
	opts = append(options, opts...)
	srv := wsSrv.NewServer(opts...)
	srv.Use(
		wsSrv.PanicHandler(logger.GetLogger()),
		wsSrv.TracingHandler(app.Router().Tracer()),
		wsSrv.LogHandler(logger.GetAccess(), !cfg.AccessRequestDisable),
	)
	return srv, nil
}

func InitSocketIOServer(opts ...ioSrv.Option) (*ioSrv.Engine, error) {
	cfg, err := app.Router().Config()
	if err != nil {
		return nil, err
	}
	// 初始化server
	// 找到rpc配置
	opts = append(opts, ioSrv.Port(fmt.Sprintf("%d", mustServerPort(cfg, registry.ProtoSocketIO))))
	// 初始化server
	srv := ioSrv.NewServer(opts...)
	srv.Use(
		ioSrv.PanicHandler(logger.GetLogger()),
		ioSrv.TracingHandler(app.Router().Tracer()),
		ioSrv.WithLangHandler(),
		ioSrv.LogHandler(logger.GetAccess(), !cfg.AccessRequestDisable),
	)
	return srv, nil
}

func InitHttpServer(opts ...httpSrv.Option) (*httpSrv.Engine, error) {
	cfg, err := app.Router().Config()
	if err != nil {
		return nil, err
	}
	// 初始化server
	// 找到rpc配置
	opts = append(opts, httpSrv.Addr(fmt.Sprintf(":%d", mustServerPort(cfg, registry.ProtoHTTP))))
	srv := httpSrv.New(opts...)
	srv.Use(
		httpSrv.PanicHandler(logger.GetLogger()),
		httpSrv.TracingHandler(app.Router().Tracer()),
		httpSrv.LogHandler(logger.GetAccess(), !cfg.AccessRequestDisable),
		httpSrv.WithLangHandler(),
		httpSrv.SyncTimeoutHandler(logger.GetLogger()),
	)
	return srv, nil
}
