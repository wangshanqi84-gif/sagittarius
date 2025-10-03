package server

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

type core func(*Context)

type Context struct {
	ctx           context.Context
	conn          *Conn
	cores         []core
	header        interface{}
	data          []byte
	index         int8
	disableAccess bool
	reader        func(c *Context, v interface{}) error
}

func newContext(r func(c *Context, v interface{}) error) *Context {
	c := &Context{
		index:         0,
		header:        nil,
		data:          nil,
		cores:         nil,
		conn:          nil,
		disableAccess: false,
		reader:        JsonReader,
		ctx:           context.TODO(),
	}
	if r != nil {
		c.reader = r
	}
	return c
}

func (c *Context) reset() *Context {
	c.data = nil
	c.header = nil
	c.index = 0
	c.conn = nil
	c.cores = nil
	c.disableAccess = false
	c.ctx = context.TODO()
	return c
}

func (c *Context) Build(ctx context.Context, conn *Conn) {
	c.ctx = ctx
	c.conn = conn
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

func (c *Context) Conn() *Conn {
	return c.conn
}

func (c *Context) DisableAccess() {
	c.disableAccess = true
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

func (c *Context) Bind(v interface{}) error {
	return c.reader(c, v)
}

func (c *Context) Write(id int32, data []byte) error {
	return Write(c.ctx, id, data, c.conn)
}

func (c *Context) Header() interface{} {
	return c.header
}

func (c *Context) Body() []byte {
	return c.data
}

func PBReader(c *Context, v interface{}) error {
	protoV, ok := v.(proto.Message)
	if !ok {
		return errors.New("value must proto.Message")
	}
	if c.data == nil || len(c.data) == 0 {
		return errors.New("nil data")
	}
	if err := proto.Unmarshal(c.data, protoV); err != nil {
		return err
	}
	return nil
}

func JsonReader(c *Context, v interface{}) error {
	if c.data == nil || len(c.data) == 0 {
		return errors.New("nil data")
	}
	if err := json.Unmarshal(c.data, v); err != nil {
		return err
	}
	return nil
}
