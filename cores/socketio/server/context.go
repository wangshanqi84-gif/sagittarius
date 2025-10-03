package server

import (
	"context"
	"encoding/json"

	gCtx "github.com/wangshanqi84-gif/sagittarius/cores/context"
	gErrors "github.com/wangshanqi84-gif/sagittarius/cores/errors"

	socketio "github.com/googollee/go-socket.io"
)

type core func(*Context)

type Context struct {
	ctx   context.Context
	conn  socketio.Conn
	cores []core
	index int8

	data  string
	event string
	resp  string
}

func newContext() *Context {
	c := &Context{
		index: 0,
		cores: nil,
		conn:  nil,
		data:  "",
		event: "",
		resp:  "",
		ctx:   context.TODO(),
	}
	return c
}

func (c *Context) reset() *Context {
	c.index = 0
	c.conn = nil
	c.cores = nil
	c.data = ""
	c.event = ""
	c.resp = ""
	c.ctx = context.TODO()
	return c
}

func (c *Context) do() {
	for c.index < int8(len(c.cores)) {
		c.cores[c.index](c)
		c.index++
	}
}

func (c *Context) Ctx() context.Context {
	return c.ctx
}

func (c *Context) Conn() socketio.Conn {
	return c.conn
}

func (c *Context) Abort() {
	c.index = int8(len(c.cores))
}

func (c *Context) Next() {
	c.index++
	for c.index < int8(len(c.cores)) {
		c.cores[c.index](c)
		c.index++
	}
}

func (c *Context) JsonBind(v interface{}) error {
	if len(c.data) == 0 {
		return nil
	}
	return json.Unmarshal([]byte(c.data), v)
}

func (c *Context) JsonOK(data interface{}) error {
	body := map[string]interface{}{
		"status":  0,
		"message": "success",
	}
	if data != nil {
		body["data"] = data
	}
	bs, err := json.Marshal(body)
	if err != nil {
		return err
	}
	c.resp = string(bs)
	return nil
}

func (c *Context) JsonErr(err error) error {
	ge := gErrors.Cause(err, gCtx.FromLangClientContext(c.ctx))
	body := map[string]interface{}{
		"status":  ge.Code(),
		"message": ge.Message(),
	}
	bs, err := json.Marshal(body)
	if err != nil {
		return err
	}
	c.resp = string(bs)
	return nil
}

func (c *Context) JsonCustom(v interface{}) error {
	if v == nil {
		return nil
	}
	bs, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.resp = string(bs)
	return nil
}
