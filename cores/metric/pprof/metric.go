package pprof

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"sync"
	"time"
)

type Option func(*Metric)

type Metric struct {
	ctx  context.Context
	mux  *http.ServeMux
	port int
	open bool
}

func SetOpenFlag(isOpen bool) Option {
	return func(m *Metric) {
		m.open = isOpen
	}
}

func SetPort(port int) Option {
	return func(m *Metric) {
		m.port = port
	}
}

var _m *Metric
var _once sync.Once

func InitMetric(ctx context.Context, opts ...Option) *Metric {
	_once.Do(func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

		_m = &Metric{
			ctx:  ctx,
			mux:  mux,
			open: true,
		}
		for _, opt := range opts {
			if opt != nil {
				opt(_m)
			}
		}
		if _m.open && _m.port == 0 {
			panic("pprof is open, port is zero")
		}
	})

	return _m
}

func (m *Metric) Start() {
	if !m.open {
		return
	}
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", m.port),
		Handler:      m.mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				panic(err)
			}
		}
	}()
	go func() {
		select {
		case <-m.ctx.Done():
			httpServer.Shutdown(m.ctx)
		}
	}()
}

func (m *Metric) Reports() chan string {
	return nil
}
