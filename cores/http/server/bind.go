package server

import (
	"encoding/json"
	"encoding/xml"
	"net/url"

	"github.com/go-playground/form/v4"
)

var (
	_binders = map[string]IBinder{
		"application/x-www-form-urlencoded": newFormBinder(),
		"application/json":                  newJsonBinder(),
		"application/xml":                   newXmlBinder(),
		"query":                             newQueryBinder(),
	}
)

type IBinder interface {
	Unmarshal(c *Context, v interface{}) error
}

type FormBinder struct {
	decoder *form.Decoder
}

func newFormBinder() *FormBinder {
	d := form.NewDecoder()
	d.SetTagName("json")
	return &FormBinder{
		decoder: d,
	}
}

func (fb *FormBinder) Unmarshal(c *Context, v interface{}) error {
	if err := c.Request().ParseForm(); err != nil {
		return err
	}
	vs, err := url.ParseQuery(string(c.reqBody))
	if err != nil {
		return err
	}
	return fb.decoder.Decode(v, vs)
}

type QueryBinder struct {
	decoder *form.Decoder
}

func newQueryBinder() *QueryBinder {
	d := form.NewDecoder()
	d.SetTagName("json")
	return &QueryBinder{
		decoder: d,
	}
}

func (qb *QueryBinder) Unmarshal(c *Context, v interface{}) error {
	return qb.decoder.Decode(v, c.Request().URL.Query())
}

type JsonBinder struct{}

func newJsonBinder() *JsonBinder {
	return &JsonBinder{}
}

func (jb *JsonBinder) Unmarshal(c *Context, v interface{}) error {
	return json.Unmarshal(c.reqBody, v)
}

type XmlBinder struct{}

func newXmlBinder() *XmlBinder {
	return &XmlBinder{}
}

func (xb *XmlBinder) Unmarshal(c *Context, v interface{}) error {
	return xml.Unmarshal(c.reqBody, v)
}
