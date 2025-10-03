package client

import (
	"encoding/json"
	"encoding/xml"
)

var (
	_binders = map[string]IBinder{
		"application/json": newJsonBinder(),
		"application/xml":  newXmlBinder(),
	}
)

type IBinder interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(body []byte, v interface{}) error
}

type JsonBinder struct{}

func newJsonBinder() *JsonBinder {
	return &JsonBinder{}
}

func (jb *JsonBinder) Unmarshal(body []byte, v interface{}) error {
	return json.Unmarshal(body, v)
}

func (jb *JsonBinder) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

type XmlBinder struct{}

func newXmlBinder() *XmlBinder {
	return &XmlBinder{}
}

func (xb *XmlBinder) Unmarshal(body []byte, v interface{}) error {
	return xml.Unmarshal(body, v)
}

func (xb *XmlBinder) Marshal(v interface{}) ([]byte, error) {
	return xml.Marshal(v)
}
