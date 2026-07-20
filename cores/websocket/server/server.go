package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Conn struct {
	server     *Engine
	c          *websocket.Conn
	msgCh      chan []byte
	ctx        context.Context
	cancel     func()
	remoteAddr string
	header     http.Header
	params     map[string]string
}

func (c *Conn) Close() error {
	return c.c.Close()
}

func (c *Conn) write() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case msg := <-c.msgCh:
			if err := c.c.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				log.Println(fmt.Sprintf("websocket write message error:%v", err))
			}
		}
	}
}

func (c *Conn) serve() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if c.server.timeOut > 0 {
				_ = c.c.SetReadDeadline(time.Now().Add(c.server.timeOut))
			}
			ctx, err := Read(c.ctx, c)
			if err != nil {
				return
			}
			cores := c.server.findCore(ctx.header.(IHeader).MsgID())
			if cores == nil || len(cores) == 0 {
				cores = c.server.defaultCore()
			}
			ctx.cores = cores
			ctx.do()
			c.server.pool.Put(ctx.reset())
		}
	}
}

func (c *Conn) Header() http.Header {
	return c.header
}

func (c *Conn) Param(key string) string {
	return c.params[key]
}

func (c *Conn) ParamInt64(key string) int64 {
	if _, has := c.params[key]; !has {
		return 0
	}
	v, err := strconv.ParseInt(c.params[key], 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func (c *Conn) SetParam(key string, value interface{}) {
	c.params[key] = fmt.Sprintf("%v", value)
}

type Engine struct {
	*Group
	port    string
	path    string
	timeOut time.Duration

	mu               sync.Mutex
	activeConn       map[*Conn]struct{}
	handlers         map[int32][]core
	defaults         []core
	pool             sync.Pool
	mux              *Mux
	coder            IEncoder
	connCloseHandler core
	httpServer       *http.Server
	onStop           []func()
	onConnect        []func(*Conn) error
	onDisconnect     []func(*Conn)
	bodyReader       func(c *Context, v interface{}) error
}

func NewServer(opts ...Option) *Engine {
	engine := &Engine{
		mux: newMux(),
	}
	group := &Group{
		svr: engine,
	}
	engine.Group = group
	for _, opt := range opts {
		opt(engine)
	}
	engine.pool.New = func() interface{} {
		return newContext(engine.bodyReader)
	}
	return engine
}

func (s *Engine) ConnCloseHandler(handler core) {
	s.connCloseHandler = handler
}

func (s *Engine) Start(ctx context.Context) error {
	if s.coder == nil {
		panic("must set encoder.")
	}
	if s.port == "" {
		panic("must set listen port.")
	}
	if s.path == "" {
		panic("must set websocket path")
	}
	s.mux.Use(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Println(fmt.Sprintf(string(debug.Stack())))
				}
			}()
			handler.ServeHTTP(w, r)
		})
	})
	s.mux.HandleFunc(s.path, func(w http.ResponseWriter, r *http.Request) {
		if !websocket.IsWebSocketUpgrade(r) {
			return
		}
		c, err := s.mux.upGrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		cCtx := context.WithValue(ctx, "upgrade", time.Now().Format("2006-01-02 15:04:05.000"))
		cCtx = context.WithValue(cCtx, "remote", c.RemoteAddr().String())

		nCtx, fn := context.WithCancel(cCtx)
		cn := Conn{
			ctx:        nCtx,
			cancel:     fn,
			c:          c,
			msgCh:      make(chan []byte),
			server:     s,
			header:     r.Header.Clone(),
			params:     make(map[string]string),
			remoteAddr: c.RemoteAddr().String(),
		}
		q := r.URL.Query()
		for k, vs := range q {
			cn.params[k] = vs[0]
		}
		go func(c *Conn) {
			s.trackConn(c, true)
			defer s.trackConn(c, false)

			cn.serve()

			if len(s.onDisconnect) > 0 {
				for _, f := range s.onDisconnect {
					f(&cn)
				}
			}
		}(&cn)
		go func(c *Conn) {
			cn.write()
		}(&cn)

		if len(s.onConnect) > 0 {
			for _, f := range s.onConnect {
				if err = f(&cn); err != nil {
					return
				}
			}
		}
	})
	s.httpServer = &http.Server{
		Addr:         ":" + s.port,
		Handler:      s.mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	log.Println(fmt.Sprintf("websocket path:%v, port:%v init done", s.path, s.port))
	if err := s.httpServer.ListenAndServe(); err != nil {
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
	return s.httpServer.Close()
}

func (s *Engine) trackConn(c *Conn, add bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeConn == nil {
		s.activeConn = make(map[*Conn]struct{})
	}
	if add {
		s.activeConn[c] = struct{}{}
	} else {
		delete(s.activeConn, c)
	}
}

func (s *Engine) addCore(id int32, cores ...core) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.handlers == nil {
		s.handlers = make(map[int32][]core)
	}

	if _, has := s.handlers[id]; has {
		panic(fmt.Sprintf("server router id:%d already exist", id))
	}
	s.handlers[id] = cores
}

func (s *Engine) findCore(id int32) []core {
	if _, has := s.handlers[id]; !has {
		return nil
	}
	return s.handlers[id]
}

func (s *Engine) defaultCore() []core {
	return s.defaults
}

type Option func(*Engine)

func TimeOut(timeOut time.Duration) Option {
	return func(e *Engine) {
		e.timeOut = timeOut
	}
}

func Port(port string) Option {
	return func(e *Engine) {
		e.port = port
	}
}

func WsPath(path string) Option {
	return func(e *Engine) {
		e.path = path
	}
}

func Encoder(encoder IEncoder) Option {
	return func(e *Engine) {
		e.coder = encoder
	}
}

func OnStop(fs ...func()) Option {
	return func(e *Engine) {
		e.onStop = append(e.onStop, fs...)
	}
}

func OnConnect(cs ...func(*Conn) error) Option {
	return func(e *Engine) {
		e.onConnect = append(e.onConnect, cs...)
	}
}

func OnDisconnect(cs ...func(*Conn)) Option {
	return func(e *Engine) {
		e.onDisconnect = append(e.onDisconnect, cs...)
	}
}

func BodyReader(f func(c *Context, v interface{}) error) Option {
	return func(e *Engine) {
		e.bodyReader = f
	}
}
