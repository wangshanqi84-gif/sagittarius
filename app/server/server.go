package server

import (
	"fmt"
	netHttp "net/http"
	"strings"

	"github.com/wangshanqi84-gif/sagittarius/app"
	httpSrv "github.com/wangshanqi84-gif/sagittarius/cores/http/server"
	rpcSrv "github.com/wangshanqi84-gif/sagittarius/cores/rpc/server"
	ioSrv "github.com/wangshanqi84-gif/sagittarius/cores/socketio/server"
	wsSrv "github.com/wangshanqi84-gif/sagittarius/cores/websocket/server"
	"github.com/wangshanqi84-gif/sagittarius/logger"

	grpcPrometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

func InitRPCServer(opts ...rpcSrv.Option) *rpcSrv.Server {
	cfg := app.Router().Config()
	// 初始化server
	// 找到rpc配置
	for _, svr := range cfg.Svrs {
		if strings.ToLower(svr.Proto) == "rpc" {
			opts = append(opts, rpcSrv.Address(fmt.Sprintf(":%d", svr.Port)))
			break
		}
	}
	if len(opts) == 0 {
		panic("undefined rpc server port")
	}
	opts = append(opts, rpcSrv.UnaryInterceptor(
		rpcSrv.LangServerUnaryInterceptor(),
		rpcSrv.RecoverServerInterceptor(logger.GetLogger()),
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

	netHttp.Handle("/metrics", promhttp.Handler())
	return srv
}

func InitWebSocketServer(opts ...wsSrv.Option) *wsSrv.Engine {
	cfg := app.Router().Config()
	// 初始化server
	// 找到websocket配置
	for _, svr := range cfg.Svrs {
		if strings.ToLower(svr.Proto) == "websocket" {
			opts = append(opts, wsSrv.Port(fmt.Sprintf("%d", svr.Port)))
			break
		}
	}
	if len(opts) == 0 {
		panic("undefined websocket server port")
	}
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
	return srv
}

func InitSocketIOServer(opts ...ioSrv.Option) *ioSrv.Engine {
	cfg := app.Router().Config()
	// 初始化server
	// 找到rpc配置
	for _, svr := range cfg.Svrs {
		if strings.ToLower(svr.Proto) == "socketio" {
			opts = append(opts, ioSrv.Port(fmt.Sprintf("%d", svr.Port)))
			break
		}
	}
	if len(opts) == 0 {
		panic("undefined socket.io server port")
	}
	// 初始化server
	srv := ioSrv.NewServer(opts...)
	srv.Use(
		ioSrv.PanicHandler(logger.GetLogger()),
		ioSrv.TracingHandler(app.Router().Tracer()),
		ioSrv.WithLangHandler(),
		ioSrv.LogHandler(logger.GetAccess(), !cfg.AccessRequestDisable),
	)
	return srv
}

func InitHttpServer(opts ...httpSrv.Option) *httpSrv.Engine {
	cfg := app.Router().Config()
	// 初始化server
	// 找到rpc配置
	for _, svr := range cfg.Svrs {
		if strings.ToLower(svr.Proto) == "http" {
			opts = append(opts, httpSrv.Addr(fmt.Sprintf(":%d", svr.Port)))
			break
		}
	}
	if len(opts) == 0 {
		panic("undefined http server port")
	}
	srv := httpSrv.New(opts...)
	srv.Use(
		httpSrv.PanicHandler(logger.GetLogger()),
		httpSrv.TracingHandler(app.Router().Tracer()),
		httpSrv.LogHandler(logger.GetAccess(), !cfg.AccessRequestDisable),
		httpSrv.WithLangHandler(),
		httpSrv.SyncTimeoutHandler(logger.GetLogger()),
	)
	return srv
}
