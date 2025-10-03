package server

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Middleware func(http.Handler) http.Handler

type Mux struct {
	*http.ServeMux
	middlewares []Middleware
	upGrader    websocket.Upgrader
}

func newMux() *Mux {
	return &Mux{
		ServeMux: http.NewServeMux(),
		upGrader: websocket.Upgrader{
			WriteBufferSize:  2048,
			ReadBufferSize:   2048,
			HandshakeTimeout: time.Second * 5,
			CheckOrigin: func(r *http.Request) bool {
				if r.Method != "GET" {
					return false
				}
				return true
			},
		},
	}
}

func (m *Mux) Use(middlewares ...Middleware) {
	m.middlewares = append(m.middlewares, middlewares...)
}

func (m *Mux) Handle(pattern string, handler http.Handler) {
	handler = applyMiddlewares(handler, m.middlewares...)
	m.ServeMux.Handle(pattern, handler)
}

func (m *Mux) HandleFunc(pattern string, handler http.HandlerFunc) {
	newHandler := applyMiddlewares(handler, m.middlewares...)
	m.ServeMux.Handle(pattern, newHandler)
}

func applyMiddlewares(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return handler
}
