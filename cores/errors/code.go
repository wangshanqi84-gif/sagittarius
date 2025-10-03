package errors

import (
	"errors"
	"fmt"
	"strings"
)

const (
	unknownCode        = 500
	unKnowErrorMessage = "Server busy, please try again later"
)

type Error struct {
	code    int
	message string
	mapping map[string]string
}

func (e *Error) Error() string { return fmt.Sprintf("%d - %s", e.code, e.message) }

func (e *Error) Code() int {
	return e.code
}

func (e *Error) Code32() int32 {
	return int32(e.code)
}

func (e *Error) Message() string {
	return e.message
}

func (e *Error) langMessage(lang string) string {
	if _, has := e.mapping[lang]; has {
		return e.mapping[lang]
	}
	for k, v := range e.mapping {
		if strings.Contains(lang, k) {
			return v
		}
	}
	return e.message
}

func (e *Error) Lang(lang string, message string) *Error {
	e.mapping[lang] = message
	return e
}

func (e *Error) Is(err error) bool {
	if se := new(Error); errors.As(err, &se) {
		return se.code == e.code
	}
	return false
}

// withMessage 信息组合error
type withMessage struct {
	cause error
	msg   string
}

func (w *withMessage) Error() string { return w.msg + "| " + w.cause.Error() }

func (w *withMessage) Cause() error { return w.cause }

func (w *withMessage) Unwrap() error { return w.cause }

// New 创建错误
func New(code int, defaultMessage string) *Error {
	return &Error{
		code:    code,
		message: defaultMessage,
		mapping: make(map[string]string),
	}
}

// WithMessage 追加错误信息
func WithMessage(err error, message string) error {
	if err == nil {
		return nil
	}
	return &withMessage{
		cause: err,
		msg:   message,
	}
}

func Cause(err error, lang string) *Error {
	type causer interface {
		Cause() error
	}

	for err != nil {
		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
	}
	if _, ok := err.(*Error); ok {
		e := err.(*Error)
		return New(e.code, e.langMessage(lang))
	}
	return New(unknownCode, unKnowErrorMessage)
}
