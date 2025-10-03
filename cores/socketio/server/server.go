package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	skio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
)

type Option func(*Engine)

func Port(port string) Option {
	return func(e *Engine) {
		e.port = port
	}
}

func OnStop(fs ...func()) Option {
	return func(e *Engine) {
		e.onStop = append(e.onStop, fs...)
	}
}

func AllowNamespaces(ns []string) Option {
	return func(e *Engine) {
		e.namespaces = ns
	}
}

func ConnCloseHandler(handler core) Option {
	return func(e *Engine) {
		e.connCloseHandler = handler
	}
}

func PingTimeout(t time.Duration) Option {
	return func(e *Engine) {
		e.pingTimeout = t
	}
}

func PingInterval(t time.Duration) Option {
	return func(e *Engine) {
		e.pingInterval = t
	}
}

type Engine struct {
	httpSrv *http.Server
	sioSrv  *skio.Server

	port             string
	namespaces       []string
	path             string
	onStop           []func()
	connCloseHandler core
	pingTimeout      time.Duration
	pingInterval     time.Duration

	pool  sync.Pool
	cores []core
}

func NewServer(opts ...Option) *Engine {
	engine := &Engine{
		pingTimeout:  time.Minute,
		pingInterval: time.Second * 20,
		path:         "/socket.io/",
	}
	for _, opt := range opts {
		opt(engine)
	}
	engine.pool.New = func() interface{} {
		return newContext()
	}
	engine.newSocketIO()
	return engine
}

func (s *Engine) Server() *skio.Server {
	return s.sioSrv
}

func (s *Engine) Use(cores ...core) {
	s.cores = append(s.cores, cores...)
}

func (s *Engine) OnEvent(namespace string, event string, f core) {
	s.sioSrv.OnEvent(namespace, event, func(conn skio.Conn, data string) string {
		cCtx, ok := conn.Context().(*Context)
		if ok {
			cCtx.data = data
			cCtx.event = event
			cCtx.cores = append(cCtx.cores, append(s.cores, f)...)

			cCtx.do()

			cCtx.reset()
			cCtx.conn = conn
			return cCtx.resp
		}
		return ""
	})
}

func (s *Engine) OnEventForAllNamespace(event string, f core) {
	if len(s.namespaces) == 0 {
		return
	}
	for _, ns := range s.namespaces {
		if ns[0] != '/' {
			ns = fmt.Sprintf("/%s", ns)
		}
		s.sioSrv.OnEvent(ns, event, func(conn skio.Conn, data string) string {
			cCtx, ok := conn.Context().(*Context)
			if ok {
				cCtx.data = data
				cCtx.event = event
				cCtx.cores = append(cCtx.cores, append(s.cores, f)...)

				cCtx.do()

				cCtx.reset()
				cCtx.conn = conn
				return cCtx.resp
			}
			return ""
		})
	}
}

func (s *Engine) Start(ctx context.Context) error {
	if s.port == "" {
		panic("must set listen port.")
	}
	go func() {
		if err := s.sioSrv.Serve(); err != nil {
			panic(err)
		}
	}()
	mux := http.NewServeMux()
	mux.Handle(s.path, s.sioSrv)
	s.httpSrv = &http.Server{
		Addr:         ":" + s.port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	if err := s.httpSrv.ListenAndServe(); err != nil {
		if err != http.ErrServerClosed {
			return err
		}
	}
	return nil
}

func (s *Engine) Stop(_ context.Context) error {
	if len(s.onStop) > 0 {
		for _, f := range s.onStop {
			f()
		}
	}

	s.sioSrv.Close()
	s.httpSrv.Close()
	return nil
}

func (s *Engine) newSocketIO() {
	s.sioSrv = skio.NewServer(&engineio.Options{
		PingTimeout:  s.pingTimeout,
		PingInterval: s.pingInterval,
		Transports: []transport.Transport{
			&polling.Transport{
				CheckOrigin: func(r *http.Request) bool {
					return true
				},
				Client: &http.Client{
					Timeout: time.Minute,
				},
			},
			&websocket.Transport{
				CheckOrigin: func(r *http.Request) bool {
					return true
				},
				WriteBufferSize:  2048,
				ReadBufferSize:   2048,
				HandshakeTimeout: time.Second * 10,
			},
		},
	})
	s.sioSrv.OnConnect("/", func(conn skio.Conn) error {
		return nil
	})
	s.sioSrv.OnDisconnect("/", func(conn skio.Conn, msg string) {
		log.Println("root disconnect, cCtx:", msg)
	})
	for _, ns := range s.namespaces {
		if ns[0] != '/' {
			ns = fmt.Sprintf("/%s", ns)
		}
		s.sioSrv.OnConnect(ns, func(conn skio.Conn) error {
			cCtx := s.pool.Get().(*Context)
			cCtx.conn = conn
			conn.SetContext(cCtx)
			log.Println("connect, id:", conn.URL(), conn.ID())
			return nil
		})
		s.sioSrv.OnDisconnect(ns, func(conn skio.Conn, msg string) {
			cCtx, ok := conn.Context().(*Context)
			if ok {
				if s.connCloseHandler != nil {
					s.connCloseHandler(cCtx)
				}
				s.pool.Put(cCtx.reset())
			}
			log.Println(ns, "disconnect,disconnect, id:", conn.URL(), conn.ID())
		})
	}
}
