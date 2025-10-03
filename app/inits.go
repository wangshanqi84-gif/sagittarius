package app

import (
	"context"
	"strings"

	"github.com/wangshanqi84-gif/sagittarius/app/config"
	"github.com/wangshanqi84-gif/sagittarius/consul"
	"github.com/wangshanqi84-gif/sagittarius/cores/env"
	"github.com/wangshanqi84-gif/sagittarius/cores/logger"
	"github.com/wangshanqi84-gif/sagittarius/cores/metric"
	"github.com/wangshanqi84-gif/sagittarius/cores/metric/local"
	"github.com/wangshanqi84-gif/sagittarius/cores/metric/pprof"
	"github.com/wangshanqi84-gif/sagittarius/cores/metric/sentry"
	"github.com/wangshanqi84-gif/sagittarius/cores/registry"
	cConsul "github.com/wangshanqi84-gif/sagittarius/cores/registry/consul"
	cEtcd "github.com/wangshanqi84-gif/sagittarius/cores/registry/etcd"
	cNacos "github.com/wangshanqi84-gif/sagittarius/cores/registry/nacos"
	"github.com/wangshanqi84-gif/sagittarius/cores/tracing"
	"github.com/wangshanqi84-gif/sagittarius/cores/tracing/jaeger"
	"github.com/wangshanqi84-gif/sagittarius/etcd"
	gLog "github.com/wangshanqi84-gif/sagittarius/logger"
	"github.com/wangshanqi84-gif/sagittarius/nacos"
)

func initLogger(cfg *config.LogConfig) {
	if cfg != nil {
		// 日志配置
		var opts []logger.Option
		// 设置日志路径
		if env.GetEnv(env.SgtLogPath) != "" {
			opts = append(opts, logger.SetPath(env.GetEnv(env.SgtLogPath)))
		}
		// 设置日志有效期
		if cfg.SaveDays > 0 {
			opts = append(opts, logger.SetSaveDays(cfg.SaveDays))
		}
		// 设置日志分割
		if cfg.Rotation != "" {
			rot := strings.ToLower(cfg.Rotation)
			rotation := logger.RotationDay
			if rot == "hour" {
				rotation = logger.RotationHour
			}
			opts = append(opts, logger.SetRotation(rotation))
		}
		// 设置格式
		if cfg.Format == logger.JsonFormat || cfg.Format == logger.ConsoleFormat {
			opts = append(opts, logger.SetFormat(cfg.Format))
		}
		// 初始化日志
		gLog.InitLogger(cfg.Level, opts...)
	} else {
		// 初始化日志
		gLog.InitLogger("")
	}
}

func initTracer(fullName string) tracing.Tracer {
	addr := env.GetEnv(env.SgtJaegerAddr)
	// tracer配置
	var opts []jaeger.Option
	// jaeger收集器配置
	if addr != "" {
		opts = append(opts, jaeger.WithAddr(addr))
	}
	// 初始化tracer
	return jaeger.NewTracer(fullName, opts...)
}

func initDiscovery(ctx context.Context, cfg *config.ServiceConfig) registry.Discovery {
	if cfg.Discovery == nil || cfg.Discovery.Used == "" {
		return nil
	}
	switch cfg.Discovery.Used {
	case "consul":
		// 获取环境配置
		addr := env.GetEnv(env.SgtConsulAddr)
		if addr == "" {
			return nil
		}
		addr = strings.TrimRight(addr, "/")
		// 创建客户端
		var clientOpts []consul.Option
		c := consul.NewConsulClient(addr, clientOpts...)
		// 生成服务发现
		discoveryOpts := []cConsul.Option{
			cConsul.Context(ctx),
		}
		return cConsul.NewDiscovery(c, discoveryOpts...)
	case "nacos":
		// 获取环境配置
		path, accessKey, secretKey, userName, password := env.GetNacosEnv()
		if path == "" {
			return nil
		}
		// 创建客户端
		ncopts := []nacos.Option{
			nacos.WithRunEnv(env.GetRunEnv()),
			nacos.WithLogger(gLog.GetGen()),
			nacos.WithServerPath(path),
		}
		if accessKey != "" {
			ncopts = append(ncopts, nacos.WithAccessKey(accessKey))
		}
		if secretKey != "" {
			ncopts = append(ncopts, nacos.WithSecretKey(secretKey))
		}
		if userName != "" {
			ncopts = append(ncopts, nacos.WithUserName(userName))
		}
		if password != "" {
			ncopts = append(ncopts, nacos.WithPassword(password))
		}
		c := nacos.NewNamingClient(r.info.Namespace, r.info.Product, r.info.ServiceName, ncopts...)
		// 生成服务发现
		discoveryOpts := []cNacos.Option{
			cNacos.Namespace(r.info.Namespace),
			cNacos.Product(r.info.Product),
		}
		return cNacos.NewDiscovery(c, discoveryOpts...)
	case "etcd":
		// 获取环境配置
		eps, userName, password, dailTimeout := env.GetEtcdEnv()
		if eps == "" {
			return nil
		}
		ss := strings.Split(eps, ",")
		// 创建客户端
		edopts := []etcd.Option{
			etcd.Endpoints(ss),
			etcd.Username(userName),
			etcd.Password(password),
			etcd.DialTimeout(dailTimeout),
		}
		c := etcd.NewEtcdClient(edopts...)
		// 生成服务发现
		discoveryOpts := []cEtcd.Option{
			cEtcd.Context(ctx),
			cEtcd.Namespace(r.info.Namespace),
			cEtcd.Product(r.info.Product),
		}
		return cEtcd.NewDiscovery(c, discoveryOpts...)
	}
	return nil
}

func initMetric(ctx context.Context, fullName string, cfg []*config.ServerConfig) []metric.IMetric {
	if strings.ToLower(env.GetEnv(env.SgtMetricDisable)) == "true" {
		return nil
	}
	// 找到最大配置端口
	port := 0
	for _, c := range cfg {
		if c.Port > port {
			port = c.Port
		}
	}
	if port == 0 {
		port = 8801
	} else {
		port += 1
	}
	return []metric.IMetric{
		local.InitMetric(ctx),
		pprof.InitMetric(ctx, pprof.SetPort(port)),
		sentry.InitMetric(ctx, sentry.SetServerName(fullName)),
	}
}
