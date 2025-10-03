package server

import (
	"context"
	"io"
)

type IHeader interface {
	MsgID() int32
	Trace() []byte
}

type IEncoder interface {
	Read(ctx context.Context, bytes []byte) (IHeader, []byte, error)
	Write(ctx context.Context, id int32, data []byte) ([]byte, error)
}

func Read(ctx context.Context, conn *Conn) (*Context, error) {
	c := conn.server.pool.Get().(*Context)
	c.Build(ctx, conn)
	// read message
	mt, data, err := conn.c.ReadMessage()
	if err != nil || mt == -1 {
		if conn.server.connCloseHandler != nil {
			conn.server.connCloseHandler(c)
		}
		if err == nil {
			err = io.EOF
		}
		return nil, err
	}
	c.header, c.data, err = conn.server.coder.Read(ctx, data)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func Write(ctx context.Context, id int32, data []byte, conn *Conn) error {
	msg, err := conn.server.coder.Write(ctx, id, data)
	if err != nil {
		return err
	}
	conn.msgCh <- msg
	return nil
}
