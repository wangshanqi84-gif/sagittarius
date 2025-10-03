package context

import (
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"
)

type Metadata struct {
	metadata.MD
}

func (m Metadata) ForeachKey(handler func(key, val string) error) error {
	for k, values := range m.MD {
		for _, v := range values {
			if err := handler(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m Metadata) Set(key, val string) {
	m.MD[key] = append(m.MD[key], val)
}

func (m Metadata) Get(key string) string {
	if _, has := m.MD[key]; !has {
		return ""
	}
	if len(m.MD[key]) == 0 {
		return ""
	}
	return m.MD[key][0]
}

const (
	_uberCtxServiceKey    = "_uber_ctx_service_key"
	_uberCtxTimeoutKey    = "_uber_ctx_timeout_key"
	_uberCtxLangKey       = "lang"
	_uberCtxLangAcceptKey = "Accept-Language"
)

func GetUberMeta(md Metadata) string {
	return md.Get(_uberCtxServiceKey)
}

func SetUberMeta(md Metadata, sk string) {
	md.Set(_uberCtxServiceKey, sk)
}

func GetUberHttpHeader(h http.Header) string {
	return h.Get(_uberCtxServiceKey)
}

func SetUberHttpHeader(h http.Header, sk string) {
	h.Set(_uberCtxServiceKey, sk)
}

func GetUberHttpTimeoutHeader(h http.Header) string {
	return h.Get(_uberCtxTimeoutKey)
}

func SetUberHttpTimeoutHeader(h http.Header, t string) {
	h.Set(_uberCtxTimeoutKey, t)
}

func GetUberHttpLangHeader(h http.Header) string {
	lang := h.Get(_uberCtxLangAcceptKey)
	if lang == "" {
		lang = h.Get(_uberCtxLangKey)
	}
	if lang != "" {
		lang = strings.Split(lang, ",")[0]
	}
	return lang
}

func SetUberHttpLangHeader(h http.Header, lang string) {
	h.Set(_uberCtxLangKey, lang)
}

func GetUberLangHeader(md Metadata) string {
	return md.Get(_uberCtxLangKey)
}

func SetUberLangHeader(md Metadata, lang string) {
	md.Set(_uberCtxLangKey, lang)
}
