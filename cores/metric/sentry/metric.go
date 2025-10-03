package sentry

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/wangshanqi84-gif/sagittarius/cores/env"

	sentrygo "github.com/getsentry/sentry-go"
	"github.com/google/uuid"
)

type Option func(*Metric)

func SetServerName(serverName string) Option {
	return func(m *Metric) {
		m.serverName = serverName
	}
}

type Metric struct {
	ctx        context.Context
	dns        string
	isAlive    bool
	serverName string
}

var _m *Metric
var _once sync.Once

func InitMetric(ctx context.Context, opts ...Option) *Metric {
	_once.Do(func() {
		_m = &Metric{
			ctx:     ctx,
			isAlive: false,
		}
		for _, opt := range opts {
			if opt != nil {
				opt(_m)
			}
		}
		_m.dns = env.GetEnv(env.SgtEvnSentryDns)
		if _m.serverName == "" {
			u, _ := uuid.NewUUID()
			_m.serverName = u.String()
		}
	})
	return _m
}

func (m *Metric) Start() {
	if m.dns != "" {
		if err := sentrygo.Init(sentrygo.ClientOptions{
			Dsn:              m.dns,
			AttachStacktrace: true,
			ServerName:       m.serverName,
		}); err != nil {
			log.Println(fmt.Sprintf("init sentry error:%v", err))
		}
		m.isAlive = true
	}
}

func (m *Metric) Reports() chan string {
	return nil
}

func ErrorReport(ctx context.Context, err error) {
	if err == nil || !_m.isAlive {
		return
	}
	hub := sentrygo.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentrygo.CurrentHub().Clone()
		ctx = sentrygo.SetHubOnContext(ctx, hub)
	}
	hub.CaptureException(err)
	hub.Flush(5 * time.Second)
}
