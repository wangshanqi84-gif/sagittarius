package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/wangshanqi84-gif/sagittarius/cores/http/client/balancer"
	"github.com/wangshanqi84-gif/sagittarius/cores/http/client/balancer/random"
	"github.com/wangshanqi84-gif/sagittarius/cores/http/crypto"
	"github.com/wangshanqi84-gif/sagittarius/cores/registry"

	"github.com/pkg/errors"
	"golang.org/x/net/http2"
)

type Option func(*clientOptions)

type clientOptions struct {
	serverName          string
	certFile            string
	keyFile             string
	caFile              string
	timeout             time.Duration
	dialTimeout         time.Duration
	keepAlive           time.Duration
	maxIdleConns        int           // 最大空闲连接数
	maxIdleConnsPerHost int           // 每个主机最大空闲连接数
	idleConnTimeout     time.Duration // 空闲连接超时时间
	tlsHandshakeTimeout time.Duration // TLS握手超时时间
	syncTimeout         bool
	eps                 []string
	watcher             registry.Watcher
	balancerName        string
	interceptors        []Interceptor
	retry               int
}

// WithWatcher 服务发现监听
func WithWatcher(watcher registry.Watcher) Option {
	return func(o *clientOptions) {
		o.watcher = watcher
	}
}

// WithBalancerName 负载均衡策略
func WithBalancerName(balancerName string) Option {
	return func(o *clientOptions) {
		o.balancerName = balancerName
	}
}

func WithTimeout(d time.Duration) Option {
	return func(o *clientOptions) {
		o.timeout = d
	}
}

func WithSyncTimeout(sto bool) Option {
	return func(o *clientOptions) {
		o.syncTimeout = sto
	}
}

func WithServerName(serverName string) Option {
	return func(o *clientOptions) {
		o.serverName = serverName
	}
}

func WithCertFile(certFile string) Option {
	return func(o *clientOptions) {
		o.certFile = certFile
	}
}

func WithKeyFile(keyFile string) Option {
	return func(o *clientOptions) {
		o.keyFile = keyFile
	}
}

func WithCAFile(caFile string) Option {
	return func(o *clientOptions) {
		o.caFile = caFile
	}
}

func WithDailTimeout(dialTimeout time.Duration) Option {
	return func(o *clientOptions) {
		o.dialTimeout = dialTimeout
	}
}

func WithKeepAlive(keepAlive time.Duration) Option {
	return func(o *clientOptions) {
		o.keepAlive = keepAlive
	}
}

func WithMaxIdleConns(maxIdleConns int) Option {
	return func(o *clientOptions) {
		o.maxIdleConns = maxIdleConns
	}
}

func WithMaxIdleConnsPerHost(maxIdleConnsPerHost int) Option {
	return func(o *clientOptions) {
		o.maxIdleConnsPerHost = maxIdleConnsPerHost
	}
}

func WithIdleConnTimeout(idleConnTimeout time.Duration) Option {
	return func(o *clientOptions) {
		o.idleConnTimeout = idleConnTimeout
	}
}

func WithTLSHandshakeTimeout(tlsHandshakeTimeout time.Duration) Option {
	return func(o *clientOptions) {
		o.tlsHandshakeTimeout = tlsHandshakeTimeout
	}
}

func WithInterceptors(interceptors ...Interceptor) Option {
	return func(o *clientOptions) {
		o.interceptors = append(o.interceptors, interceptors...)
	}
}

func WithEps(eps ...string) Option {
	return func(o *clientOptions) {
		o.eps = eps
	}
}

func WithRetry(retry int) Option {
	return func(o *clientOptions) {
		o.retry = retry
	}
}

type Client struct {
	httpClient   *http.Client
	syncTimeout  bool
	interceptors []Interceptor
	insecure     bool
	resolver     *resolver
	watcher      registry.Watcher
	retry        int
}

func createTransport(tlsCfg *tls.Config, opt *clientOptions) (http.RoundTripper, error) {
	// 创建基础Transport
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   opt.dialTimeout,
			KeepAlive: opt.keepAlive,
		}).DialContext,
		MaxIdleConns:        opt.maxIdleConns,
		MaxIdleConnsPerHost: opt.maxIdleConnsPerHost,
		IdleConnTimeout:     opt.idleConnTimeout,
		TLSHandshakeTimeout: opt.tlsHandshakeTimeout,
		TLSClientConfig:     tlsCfg,
		ForceAttemptHTTP2:   true,
	}
	if err := http2.ConfigureTransport(transport); err != nil {
		log.Println(fmt.Sprintf("http2.ConfigureTransport err:%v", err))
	}
	return transport, nil
}

func createTLSConfig(opt *clientOptions) (*tls.Config, error) {
	if opt.certFile == "" && opt.keyFile == "" && opt.caFile == "" && opt.serverName == "" {
		return nil, nil
	}
	tlsCfg := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         opt.serverName,
	}
	// 加载客户端证书
	if opt.certFile != "" && opt.keyFile != "" {
		cert, err := tls.LoadX509KeyPair(opt.certFile, opt.keyFile)
		if err != nil {
			return nil, errors.WithMessage(err, "| tls.LoadX509KeyPair")
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	// 加载CA证书
	if opt.caFile != "" {
		cert, err := os.ReadFile(opt.caFile)
		if err != nil {
			return nil, errors.WithMessage(err, "| os.ReadFile")
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(cert) {
			return nil, errors.New("create CA cert failed")
		}
		tlsCfg.RootCAs = caCertPool
	}
	return tlsCfg, nil
}

func NewClient(ctx context.Context, opts ...Option) *Client {
	options := clientOptions{
		timeout:             30 * time.Second,
		dialTimeout:         30 * time.Second,
		keepAlive:           30 * time.Second,
		maxIdleConns:        100,
		maxIdleConnsPerHost: 20,
		idleConnTimeout:     90 * time.Second,
		tlsHandshakeTimeout: 10 * time.Second,
		syncTimeout:         false,
	}
	for _, o := range opts {
		o(&options)
	}
	// 创建TLS配置
	tlsCfg, err := createTLSConfig(&options)
	if err != nil {
		log.Println(fmt.Sprintf("create tls config, err:%v", err))
		tlsCfg = nil
	}
	insecure := tlsCfg == nil
	for idx := 0; idx < len(options.eps); idx++ {
		if !strings.Contains(options.eps[idx], "://") {
			if insecure {
				options.eps[idx] = "http://" + options.eps[idx]
			} else {
				options.eps[idx] = "https://" + options.eps[idx]
			}
		}
	}
	var (
		r *resolver
	)
	if options.watcher != nil {
		var builder balancer.Builder
		switch options.balancerName {
		case "random":
			builder = random.NewBuilder()
		default:
			builder = random.NewBuilder()
		}
		r, _ = newResolver(ctx, options.watcher, builder, options.eps, insecure)
	} else {
		r, _ = newResolver(ctx, nil, random.NewBuilder(), options.eps, insecure)
	}
	transport, err := createTransport(tlsCfg, &options)
	if err != nil {
		log.Println(fmt.Sprintf("http client createTransport err:%v, use default transport.", err))
		transport = http.DefaultTransport
	}
	c := &Client{
		insecure: insecure,
		httpClient: &http.Client{
			Timeout:   options.timeout,
			Transport: transport,
		},
		syncTimeout:  options.syncTimeout,
		resolver:     r,
		watcher:      options.watcher,
		interceptors: options.interceptors,
		retry:        options.retry,
	}
	return c
}

type Req struct {
	ctx        context.Context
	header     http.Header
	queryParam url.Values
	cookies    []*http.Cookie
	crypto     crypto.ICrypto
	url        string
	method     string
	body       interface{}
}

func Request(ctx context.Context, uri string) *Req {
	uri = "/" + strings.TrimLeft(uri, "/")
	return &Req{
		ctx:        ctx,
		url:        uri,
		header:     http.Header{},
		queryParam: url.Values{},
		cookies:    make([]*http.Cookie, 0),
	}
}

func (r *Req) Crypto(c crypto.ICrypto) *Req {
	r.crypto = c
	return r
}

func (r *Req) Cookies(cookies []*http.Cookie) *Req {
	r.cookies = append(r.cookies, cookies...)
	return r
}

func (r *Req) QueryParam(params map[string]string) *Req {
	for k, v := range params {
		r.queryParam.Add(k, v)
	}
	return r
}

func (r *Req) SetHeaders(values map[string]string) *Req {
	for k, v := range values {
		r.header.Set(k, v)
	}
	return r
}

func (r *Req) makeRequest() (*http.Request, error) {
	var (
		err    error
		req    *http.Request
		reader io.Reader
	)
	if r.body != nil {
		var bs []byte
		if _, has := _binders[r.header.Get("Content-Type")]; has {
			bs, err = _binders[r.header.Get("Content-Type")].Marshal(r.body)
			if err != nil {
				return nil, errors.WithMessage(err, fmt.Sprintf("| binder.Marshal:%+v", r.body))
			}
			if r.crypto != nil {
				var s string
				s, err = r.crypto.Encrypt(string(bs))
				if err != nil {
					return nil, err
				}
				bs = []byte(s)
			}
		} else if _, ok := r.body.([]byte); ok {
			bs = r.body.([]byte)
		}
		reader = bytes.NewReader(bs)
	}
	if req, err = http.NewRequest(r.method, r.url, reader); err != nil {
		return nil, err
	}
	for k, vs := range r.header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
		if strings.EqualFold(k, "Host") {
			req.Host = vs[0]
		}
	}
	q := req.URL.Query()
	for k, v := range r.queryParam {
		for _, vv := range v {
			q.Add(k, vv)
		}
	}
	req.URL.RawQuery = q.Encode()
	for _, cookie := range r.cookies {
		req.AddCookie(cookie)
	}
	return req, nil
}

func (c *Client) Get(r *Req) (*http.Response, error) {
	var (
		err  error
		resp *http.Response
	)
	r.method = http.MethodGet
	// Send request
	resp, _, err = c.do(r.ctx, r)
	if err != nil {
		return nil, err
	}
	return resp, err
}

func (c *Client) JsonGet(r *Req, respData interface{}) (*http.Response, error) {
	var (
		err  error
		resp *http.Response
		body []byte
	)
	r.header.Set("Content-Type", "application/json")
	r.method = http.MethodGet
	// Send request
	resp, body, err = c.do(r.ctx, r)
	if err != nil {
		return nil, err
	}
	if respData != nil && len(body) > 0 {
		err = c.bind(r, body, respData)
	}
	return resp, err
}

func (c *Client) JsonPost(r *Req, reqBody interface{}, respData interface{}) (*http.Response, error) {
	var (
		err  error
		resp *http.Response
		body []byte
	)
	r.header.Set("Content-Type", "application/json")
	r.method = http.MethodPost
	r.body = reqBody
	// Send request
	resp, body, err = c.do(r.ctx, r)
	if err != nil {
		return nil, err
	}
	if respData != nil && len(body) > 0 {
		err = c.bind(r, body, respData)
	}
	return resp, err
}

func (c *Client) XmlGet(r *Req, respData interface{}) (*http.Response, error) {
	var (
		err  error
		resp *http.Response
		body []byte
	)
	r.header.Set("Content-Type", "application/xml")
	r.method = http.MethodGet
	// Send request
	resp, body, err = c.do(r.ctx, r)
	if err != nil {
		return nil, err
	}
	if respData != nil && len(body) > 0 {
		err = c.bind(r, body, respData)
	}
	return resp, err
}

func (c *Client) XmlPost(r *Req, reqBody interface{}, respData interface{}) (*http.Response, error) {
	var (
		err  error
		resp *http.Response
		body []byte
	)
	r.header.Set("Content-Type", "application/xml")
	r.method = http.MethodPost
	r.body = reqBody
	// Send request
	resp, body, err = c.do(r.ctx, r)
	if err != nil {
		return nil, err
	}
	if respData != nil && len(body) > 0 {
		err = c.bind(r, body, respData)
	}
	return resp, err
}

func (c *Client) CustomGet(r *Req) (*http.Response, error) {
	var (
		err  error
		resp *http.Response
	)
	r.method = http.MethodGet
	// Send request
	resp, _, err = c.do(r.ctx, r)
	if err != nil {
		return nil, err
	}
	return resp, err
}

func (c *Client) CustomPost(r *Req, reqBody interface{}) (*http.Response, error) {
	var (
		err  error
		resp *http.Response
	)
	r.method = http.MethodPost
	r.body = reqBody
	// Send request
	resp, _, err = c.do(r.ctx, r)
	if err != nil {
		return nil, err
	}
	return resp, err
}

func (c *Client) do(ctx context.Context, r *Req) (*http.Response, []byte, error) {
	var (
		resp *http.Response
		err  error
	)
	// Send request
	for att := 0; att <= c.retry; att++ {
		var req *http.Request
		req, err = r.makeRequest()
		if err != nil {
			continue
		}
		resp, err = doInterceptors(ctx, c, req)
		if err != nil {
			continue
		}
		break
	}
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	// Reset resp.Body so it can be use again
	resp.Body = io.NopCloser(bytes.NewBuffer(body))
	return resp, body, nil
}

func (c *Client) bind(req *Req, body []byte, v interface{}) error {
	accept := req.header.Get("Content-Type")
	if accept == "" {
		accept = "application/json"
	}
	if binder, has := _binders[accept]; has {
		err := binder.Unmarshal(body, v)
		if err != nil {
			return errors.WithMessage(err, fmt.Sprintf("| binder.Unmarshal:%v", string(body)))
		}
	}
	return nil
}
