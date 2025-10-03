package logger

import (
	rotate "github.com/lestrrat-go/file-rotatelogs"
)

type Writer interface {
	check(Level) Writer
	Write(p []byte) (n int, err error)
}

type SingleWriter struct {
	r     *rotate.RotateLogs
	level Level
}

func (sw SingleWriter) check(dst Level) Writer {
	if sw.level == NoneLevel || sw.level == dst || sw.level.less(dst) {
		return sw
	}
	return nil
}

func (sw SingleWriter) Write(p []byte) (n int, err error) {
	return sw.r.Write(p)
}

type GroupWriter []*SingleWriter

func (gw GroupWriter) check(dst Level) Writer {
	g := GroupWriter{}
	for _, sw := range gw {
		if sw.level == NoneLevel || sw.level == dst || sw.level.less(dst) {
			g = append(g, sw)
		}
	}
	return g
}

func (gw GroupWriter) Write(p []byte) (n int, err error) {
	if gw == nil || len(gw) == 0 {
		return
	}
	for _, sw := range gw {
		n, err = sw.r.Write(p)
	}
	return
}
