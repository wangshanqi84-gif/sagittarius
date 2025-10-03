package server

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	gCtx "github.com/wangshanqi84-gif/sagittarius/cores/context"
	gErrors "github.com/wangshanqi84-gif/sagittarius/cores/errors"

	"github.com/go-playground/form/v4"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
)

type core func(*Context)

type Context struct {
	ctx       context.Context
	r         *http.Request
	w         http.ResponseWriter
	pathParam map[string]string

	reqBody        []byte
	reqData        interface{}
	logWithoutResp bool
	respData       interface{}

	cores []core
	index int8
	srv   *Engine
}

func newContext() *Context {
	c := &Context{
		index:          0,
		r:              nil,
		w:              nil,
		cores:          nil,
		srv:            nil,
		respData:       nil,
		reqData:        nil,
		reqBody:        nil,
		logWithoutResp: false,
		pathParam:      make(map[string]string),
		ctx:            context.TODO(),
	}
	return c
}

func (c *Context) reset() *Context {
	c.index = 0
	c.cores = nil
	c.srv = nil
	c.ctx = context.TODO()
	c.respData = nil
	c.reqData = nil
	c.reqBody = nil
	c.logWithoutResp = false
	c.pathParam = make(map[string]string)
	return c
}

func (c *Context) do() {
	for c.index < int8(len(c.cores)) {
		c.cores[c.index](c)
		c.index++
	}
}

func (c *Context) addPathParam(key string, value string) {
	c.pathParam[key] = value
}

func (c *Context) GetPathParam(key string) string {
	return c.pathParam[key]
}

func (c *Context) GetPathParamInt64(key string) int64 {
	if value, has := c.pathParam[key]; !has {
		return 0
	} else {
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0
		}
		return intValue
	}
}

func (c *Context) Ctx() context.Context {
	return c.ctx
}

func (c *Context) Writer() http.ResponseWriter {
	return c.w
}

func (c *Context) TraceID() string {
	var traceID string
	span := opentracing.SpanFromContext(c.ctx)
	if span == nil {
		traceID = ""
	} else {
		if sc, ok := span.Context().(jaeger.SpanContext); ok {
			traceID = sc.TraceID().String()
		}
	}
	return traceID
}

func (c *Context) RemoteAddr() string {
	ip := c.r.Header.Get("X-Real-IP")
	if net.ParseIP(ip) != nil {
		return ip
	}
	ip = c.r.Header.Get("X-Forward-For")
	for _, i := range strings.Split(ip, ",") {
		if net.ParseIP(i) != nil {
			return i
		}
	}
	ip, _, err := net.SplitHostPort(c.r.RemoteAddr)
	if err != nil {
		return ""
	}
	if net.ParseIP(ip) != nil {
		return ip
	}
	return ""
}

func (c *Context) WithValue(key any, value any) {
	c.ctx = context.WithValue(c.ctx, key, value)
}

func (c *Context) Body() []byte {
	return c.reqBody
}

// SetBody 入参格式和header的Content-Type保持一只
func (c *Context) SetBody(body []byte) {
	c.reqBody = body
}

func (c *Context) Request() *http.Request {
	return c.r
}

func (c *Context) LogWithoutResp() {
	c.logWithoutResp = true
}

func (c *Context) ResponseData() interface{} {
	return c.respData
}

func (c *Context) Path() string {
	return c.r.URL.Path
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

func (c *Context) ParseHeaderAtom(atom interface{}) {
	t := reflect.TypeOf(atom).Elem()
	v := reflect.ValueOf(atom).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		scheme := field.Tag.Get("scheme")
		if scheme == "" {
			continue
		}
		value := c.r.Header.Get(scheme)
		if value != "" {
			v.Field(i).SetString(value)
		}
	}
}

func (c *Context) Bind(atom interface{}, v interface{}) error {
	// 解析query
	queryBinder := _binders["query"]
	if queryBinder != nil {
		if atom != nil {
			err := queryBinder.Unmarshal(c, atom)
			if err != nil {
				return err
			}
		}
		if v != nil {
			err := queryBinder.Unmarshal(c, v)
			if err != nil {
				return err
			}
		}
	}
	// 解析body
	if len(c.reqBody) > 0 {
		accept := c.Request().Header.Get("Content-Type")
		if accept == "" {
			// 默认使用json
			accept = "application/json"
		}
		// 这里检查使用哪个binder
		for k, binder := range _binders {
			if strings.Contains(accept, k) {
				err := binder.Unmarshal(c, v)
				if err != nil {
					return err
				}
				break
			}
		}
	}
	if v != nil {
		c.reqData = v
	}
	return nil
}

func (c *Context) HttpError(code int, message string) error {
	c.w.Header().Add("Content-Type", "text/plain")
	c.w.WriteHeader(code)

	c.buildRespData(code, -1, message, nil)
	_, err := c.w.Write([]byte(message))
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) JsonCustom(data interface{}) error {
	c.w.Header().Add("Content-Type", "application/json")
	c.w.WriteHeader(http.StatusOK)

	if data == nil {
		return nil
	}
	c.respData = data
	bs, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = c.w.Write(bs)
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) FormCustom(data interface{}) error {
	c.w.Header().Add("Content-Type", "application/x-www-form-urlencoded")
	c.w.WriteHeader(http.StatusOK)

	if data == nil {
		return nil
	}
	c.respData = data
	e := form.NewEncoder()
	vs, err := e.Encode(data)
	if err != nil {
		return err
	}
	_, err = c.w.Write([]byte(vs.Encode()))
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) XmlCustom(data interface{}) error {
	c.w.Header().Add("Content-Type", "application/xml")
	c.w.WriteHeader(http.StatusOK)

	if data == nil {
		return nil
	}
	c.respData = data
	bs, err := xml.Marshal(data)
	if err != nil {
		return err
	}
	_, err = c.w.Write(bs)
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) CustomBinary(data []byte) error {
	c.w.Header().Add("Content-Type", "text/plain")
	c.w.WriteHeader(http.StatusOK)

	if data == nil {
		return nil
	}
	c.respData = data
	_, err := c.w.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) JsonOK(data interface{}) error {
	c.w.Header().Add("Content-Type", "application/json")
	c.w.WriteHeader(http.StatusOK)

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
	c.buildRespData(http.StatusOK, 0, "", data)
	_, err = c.w.Write(bs)
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) FormOK(data interface{}) error {
	c.w.Header().Add("Content-Type", "application/x-www-form-urlencoded")
	c.w.WriteHeader(http.StatusOK)

	body := map[string]interface{}{
		"status":  0,
		"message": "success",
	}
	if data != nil {
		body["data"] = data
	}
	e := form.NewEncoder()
	vs, err := e.Encode(body)
	if err != nil {
		return err
	}
	c.buildRespData(http.StatusOK, 0, "", data)
	_, err = c.w.Write([]byte(vs.Encode()))
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) XmlOK(data interface{}) error {
	c.w.Header().Add("Content-Type", "application/xml")
	c.w.WriteHeader(http.StatusOK)

	body := map[string]interface{}{
		"status":  0,
		"message": "success",
	}
	if data != nil {
		body["data"] = data
	}
	bs, err := xml.Marshal(body)
	if err != nil {
		return err
	}
	c.buildRespData(http.StatusOK, 0, "", data)
	_, err = c.w.Write(bs)
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) JsonErr(err error) error {
	c.w.Header().Add("Content-Type", "application/json")
	c.w.WriteHeader(http.StatusOK)

	ge := gErrors.Cause(err, gCtx.FromLangClientContext(c.ctx))
	body := map[string]interface{}{
		"status":  ge.Code(),
		"message": ge.Message(),
	}
	bs, err := json.Marshal(body)
	if err != nil {
		return err
	}
	c.buildRespData(http.StatusOK, ge.Code(), ge.Message(), nil)
	_, err = c.w.Write(bs)
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) FormErr(err error) error {
	c.w.Header().Add("Content-Type", "application/x-www-form-urlencoded")
	c.w.WriteHeader(http.StatusOK)

	ge := gErrors.Cause(err, gCtx.FromLangClientContext(c.ctx))
	body := map[string]interface{}{
		"status":  ge.Code(),
		"message": ge.Message(),
	}
	e := form.NewEncoder()
	vs, err := e.Encode(body)
	if err != nil {
		return err
	}
	c.buildRespData(http.StatusOK, ge.Code(), ge.Message(), nil)
	_, err = c.w.Write([]byte(vs.Encode()))
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) XmlErr(err error) error {
	c.w.Header().Add("Content-Type", "application/xml")
	c.w.WriteHeader(http.StatusOK)

	ge := gErrors.Cause(err, gCtx.FromLangClientContext(c.ctx))
	body := map[string]interface{}{
		"status":  ge.Code(),
		"message": ge.Message(),
	}
	bs, err := xml.Marshal(body)
	if err != nil {
		return err
	}
	c.buildRespData(http.StatusOK, ge.Code(), ge.Message(), nil)
	_, err = c.w.Write(bs)
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) buildRespData(httpCode int, status int, message string, body interface{}) {
	data := map[string]interface{}{
		"httpCode": httpCode,
	}
	if httpCode != http.StatusOK {
		data["message"] = message
	} else {
		bd := map[string]interface{}{
			"status":  status,
			"message": message,
		}
		if status == 0 && body != nil {
			bd["data"] = body
		}
		data["body"] = bd
	}
	c.respData = data
}
