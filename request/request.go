package request

import (
	"bytes"
	"encoding"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"jiayou_backend_spider/config"
	"jiayou_backend_spider/utils"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/exp/maps"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

const defaultUserAgent = "go-cas/v2"
const defaultMaxRedirect = 15

type Method string

func (method Method) Method() (string, error) {
	for _, v := range []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodConnect,
		http.MethodPut,
		http.MethodPatch,
		http.MethodTrace,
		http.MethodOptions,
		http.MethodHead,
		http.MethodDelete,
	} {
		if strings.ToUpper(string(method)) == v {
			return v, nil
		}
	}
	return "", fmt.Errorf("request method[%s] unsupported", method)
}

type HeaderValue struct {
	key    string
	values []string
}

func (hv *HeaderValue) Key() string {
	return hv.key
}
func (hv *HeaderValue) Values() []string {
	return hv.values
}
func (hv *HeaderValue) First() string {
	if len(hv.values) > 0 {
		return hv.values[0]
	}
	return ""
}
func (hv *HeaderValue) Last() string {
	var size = len(hv.values)
	if size > 0 {
		return hv.values[size-1]
	}
	return ""
}

type ReqHeader struct {
	header *fasthttp.RequestHeader
}

func (req *ReqHeader) Set(key, value string) {
	req.header.Set(key, value)
}
func (req *ReqHeader) Add(key, value string) {
	req.header.Add(key, value)
}
func (req *ReqHeader) Get(key string) *HeaderValue {
	var values []string
	req.header.VisitAll(func(k, v []byte) {
		if key == string(k) {
			values = append(values, string(v))
		}
	})
	return NewHeaderValue(key, values)
}
func (req *ReqHeader) Keys() []string {
	var keys []string
	req.header.VisitAll(func(key, value []byte) {
		keys = append(keys, string(key))
	})
	return keys
}
func (req *ReqHeader) UniqueKeys() []string {
	var keys = req.Keys()
	var mapKeys = map[string]struct{}{}
	for _, key := range keys {
		mapKeys[key] = struct{}{}
	}
	return maps.Keys(mapKeys)
}
func (req *ReqHeader) Values() []string {
	var values []string
	req.header.VisitAll(func(key, value []byte) {
		values = append(values, string(value))
	})
	return values
}
func (req *ReqHeader) UniqueValues() []string {
	var values = req.Values()
	var mapKeys = map[string]struct{}{}
	for _, key := range values {
		mapKeys[key] = struct{}{}
	}
	return maps.Keys(mapKeys)
}
func (req *ReqHeader) DisableSpecialHeader() {
	req.header.DisableSpecialHeader()
}
func (req *ReqHeader) DisableNormalizing() {
	req.header.DisableNormalizing()
}
func (req *ReqHeader) UserAgent() string {
	return string(req.header.UserAgent())
}
func (req *ReqHeader) SetUserAgent(ua string) {
	req.header.SetUserAgent(ua)
}
func (req *ReqHeader) ContentType() string {
	return string(req.header.UserAgent())
}
func (req *ReqHeader) SetContentType(content string) {
	req.header.SetContentType(content)
}
func (req *ReqHeader) Referer() string {
	return string(req.header.Referer())
}
func (req *ReqHeader) SetReferer(referer string) {
	req.header.SetReferer(referer)
}
func (req *ReqHeader) ContentEncoding() string {
	return string(req.header.ContentEncoding())
}
func (req *ReqHeader) SetContentEncoding(encoding string) {
	req.header.SetContentEncoding(encoding)
}
func (req *ReqHeader) Host() string {
	return string(req.header.Host())
}
func (req *ReqHeader) SetHost(host string) {
	req.header.SetHost(host)
}
func (req *ReqHeader) Method() string {
	return string(req.header.Method())
}
func (req *ReqHeader) SetMethod(method string) {
	req.header.SetMethod(method)
}
func (req *ReqHeader) ContentLength() int {
	return req.header.ContentLength()
}
func (req *ReqHeader) SetContentLength(length int) {
	req.header.SetContentLength(length)
}
func (req *ReqHeader) SetByteRange(start, end int) {
	req.header.SetByteRange(start, end)
}
func (req *ReqHeader) Protocol() string {
	return string(req.header.Protocol())
}
func (req *ReqHeader) SetProtocol(protocol string) {
	req.header.SetProtocol(protocol)
}
func (req *ReqHeader) RequestURI() string {
	return string(req.header.RequestURI())
}
func (req *ReqHeader) SetRequestURI(uri string) {
	req.header.SetRequestURI(uri)
}
func (req *ReqHeader) SetCookie(key, value string) {
	req.header.SetCookie(key, value)
}
func (req *ReqHeader) SetCookieText(cookies string) {
	cookies = strings.TrimSpace(cookies)
	for _, kv := range strings.Split(cookies, ";") {
		var item = strings.Split(kv, "=")
		if len(item) > 1 {
			var key = strings.TrimSpace(item[0])
			var value = strings.TrimSpace(strings.Replace(kv, key+"=", "", 1))
			req.SetCookie(key, value)
		}
	}
}
func (req *ReqHeader) SetCookieMap(cookies map[string]string) {
	for k, v := range cookies {
		req.header.SetCookie(k, v)
	}
}
func (req *ReqHeader) SetCookieJson(cookies string) {
	var ck map[string]any
	if err := json.Unmarshal([]byte(cookies), &ck); err == nil {
		for k, v := range ck {
			req.header.SetCookie(k, fmt.Sprintf("%v", v))
		}
	}
}
func (req *ReqHeader) Cookie(key string) string {
	return string(req.header.Cookie(key))
}
func (req *ReqHeader) CookiesText() string {
	var cookies []string
	for k, v := range req.header.Cookies() {
		cookies = append(cookies, fmt.Sprintf("%s=%s", string(k), string(v)))
	}
	return strings.Join(cookies, "; ")
}
func (req *ReqHeader) Cookies() map[string]string {
	var cookies = make(map[string]string)
	for k, v := range req.header.Cookies() {
		cookies[string(k)] = string(v)
	}
	return cookies
}
func (req *ReqHeader) DelCookie(key string) {
	req.header.DelCookie(key)
}
func (req *ReqHeader) DelAllCookie() {
	req.header.DelAllCookies()
}
func (req *ReqHeader) Reset() {
	req.header.Reset()
}
func (req *ReqHeader) String() string {
	return req.header.String()
}
func (req *ReqHeader) Header() *fasthttp.RequestHeader {
	var copied = new(fasthttp.RequestHeader)
	req.header.CopyTo(copied)
	return copied
}
func (req *ReqHeader) Clone() *ReqHeader {
	var copied = new(ReqHeader)
	copied.header = req.Header()
	return copied
}

type RespHeader struct {
	header *fasthttp.ResponseHeader
}

func (resp *RespHeader) Protocol() string {
	return string(resp.header.Protocol())
}
func (resp *RespHeader) Keys() []string {
	var keys []string
	resp.header.VisitAll(func(key, value []byte) {
		keys = append(keys, string(key))
	})
	return keys
}
func (resp *RespHeader) UniqueKeys() []string {
	var keys = resp.Keys()
	var mapKeys = map[string]struct{}{}
	for _, key := range keys {
		mapKeys[key] = struct{}{}
	}
	return maps.Keys(mapKeys)
}
func (resp *RespHeader) Values() []string {
	var values []string
	resp.header.VisitAll(func(key, value []byte) {
		values = append(values, string(value))
	})
	return values
}
func (resp *RespHeader) UniqueValues() []string {
	var values = resp.Values()
	var mapKeys = map[string]struct{}{}
	for _, key := range values {
		mapKeys[key] = struct{}{}
	}
	return maps.Keys(mapKeys)
}
func (resp *RespHeader) UniqueHeaders() map[string]string {
	var headers = make(map[string]string)
	resp.header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})
	return headers
}
func (resp *RespHeader) ContentType() string {
	return string(resp.header.ContentType())
}
func (resp *RespHeader) ContentEncoding() string {
	return string(resp.header.ContentEncoding())
}
func (resp *RespHeader) ContentLength() int {
	return resp.header.ContentLength()
}
func (resp *RespHeader) StatusCode() int {
	return resp.header.StatusCode()
}
func (resp *RespHeader) ConnectionClose() bool {
	return resp.header.ConnectionClose()
}
func (resp *RespHeader) ConnectionUpgrade() bool {
	return resp.header.ConnectionUpgrade()
}
func (resp *RespHeader) Header(key string) *HeaderValue {
	var values []string
	resp.header.VisitAll(func(k, v []byte) {
		if key == string(k) {
			values = append(values, string(v))
		}
	})
	return NewHeaderValue(key, values)
}
func (resp *RespHeader) Headers() []*HeaderValue {
	var headers []*HeaderValue
	var ckMap = make(map[string][]string)
	resp.header.VisitAll(func(k, v []byte) {
		if _, ok := ckMap[string(k)]; !ok {
			ckMap[string(k)] = make([]string, 0)
		}
		ckMap[string(k)] = append(ckMap[string(k)], string(v))
	})
	for k, values := range ckMap {
		headers = append(headers, NewHeaderValue(k, values))
	}
	return headers
}
func (resp *RespHeader) Cookie(name string) *HeaderValue {
	for _, ck := range resp.Cookies() {
		if ck.Key() == name {
			return ck
		}
	}
	return nil
}
func (resp *RespHeader) Cookies() []*HeaderValue {
	var cookies []*HeaderValue
	var ckMap = make(map[string][]string)
	for _, v := range resp.Header("Set-Cookie").Values() {
		ck := strings.Split(v, ";")
		if len(ck) > 0 {
			var kv = strings.Split(ck[0], "=")
			if len(kv) > 1 {
				if _, ok := ckMap[kv[0]]; !ok {
					ckMap[kv[0]] = make([]string, 0)
				}
				ckMap[kv[0]] = append(ckMap[kv[0]], strings.Join(kv[1:], "="))
			}
		}
	}
	for k, values := range ckMap {
		cookies = append(cookies, NewHeaderValue(k, values))
	}
	return cookies
}
func (resp *RespHeader) CookiesText() string {
	var cookies = resp.Cookies()
	var sb = new(strings.Builder)
	for _, v := range cookies {
		sb.WriteString(fmt.Sprintf("%s=%s;", v.Key(), v.Last()))
	}
	return sb.String()
}
func (resp *RespHeader) CookiesMap() map[string]string {
	var cookies = resp.Cookies()
	var ckMap = make(map[string]string)
	for _, v := range cookies {
		ckMap[v.Key()] = v.Last()
	}
	return ckMap
}
func (resp *RespHeader) String() string {
	return resp.header.String()
}

type Options struct {
	Header       *ReqHeader
	Timeout      time.Duration `json:"timeout" validate:"gte=0" default:"15s"`
	ReadTimeout  time.Duration `json:"read_timeout" validate:"gte=0" default:"15s"`
	WriteTimeout time.Duration `json:"write_timeout" validate:"gte=0" default:"15s"`
	Proxy        string        `json:"proxy"`
	MaxRedirect  int           `json:"max_redirect" validate:"gte=0" default:"15"`
	BodyStream   bool          `json:"body_stream"`
	Uncompressed bool          `json:"uncompressed"`
}

func (opts *Options) Clone() *Options {
	var copied = new(Options)
	*copied = *opts
	copied.Header = opts.Header.Clone()
	return copied
}

type Response struct {
	response *fasthttp.Response
	opts     *Options
	buffer   *bytes.Buffer
}

func (resp *Response) readFromStream() error {
	if resp.response.BodyStream() != nil && resp.opts.BodyStream {
		raw, err := io.ReadAll(resp.response.BodyStream())
		if err != nil {
			return err
		}
		resp.buffer = bytes.NewBuffer(raw)
	} else {
		if resp.opts.Uncompressed {
			raw, err := resp.response.BodyUncompressed()
			if err == nil {
				resp.buffer = bytes.NewBuffer(raw)
			}
		} else {
			resp.buffer = bytes.NewBuffer(resp.response.Body())
		}

	}
	return nil
}
func (resp *Response) Options() *Options {
	return resp.opts
}
func (resp *Response) Status() int {
	if resp.response != nil {
		return resp.response.Header.StatusCode()
	}
	return 0
}
func (resp *Response) Success() bool {
	if resp.response != nil {
		return resp.response.StatusCode() >= 200 && resp.response.StatusCode() <= 299
	}
	return false
}
func (resp *Response) Close() {
	if resp.response != nil {
		fasthttp.ReleaseResponse(resp.response)
		resp.response = nil
	}
	if resp.buffer != nil {
		resp.buffer.Reset()
	}
}
func (resp *Response) Content() []byte {
	if resp.buffer != nil {
		return resp.buffer.Bytes()
	}
	resp.readFromStream()
	return resp.buffer.Bytes()
}
func (resp *Response) Text() string {
	if resp.buffer != nil {
		return resp.buffer.String()
	}
	resp.readFromStream()
	return resp.buffer.String()
}
func (resp *Response) Header() *RespHeader {
	if resp.response != nil {
		return &RespHeader{header: &resp.response.Header}
	}
	return nil
}
func (resp *Response) Cookies() map[string]string {
	if resp.response != nil {
		var cookies = make(map[string]string)
		resp.response.Header.VisitAllCookie(func(key, value []byte) {
			cookies[string(key)] = string(value)
		})
		return cookies
	}
	return nil
}
func (resp *Response) Json() map[string]any {
	text := resp.Text()
	if text != "" {
		var jsonData map[string]any
		json.Unmarshal([]byte(text), &jsonData)
		return jsonData
	}
	return nil
}
func defaultClient() *fasthttp.Client {
	return &fasthttp.Client{
		NoDefaultUserAgentHeader: true,
		DialTimeout: func(addr string, timeout time.Duration) (net.Conn, error) {
			return fasthttp.DialTimeout(addr, timeout)
		},
		ReadTimeout:  time.Second * 15,
		WriteTimeout: time.Second * 15,
	}
}
func reader(data any) (io.Reader, error) {
	if data == nil {
		return nil, nil
	}
	var reader io.Reader
	switch t := data.(type) {
	case string:
		reader = strings.NewReader(t)
	case []byte:
		reader = bytes.NewReader(t)
	case url.Values:
		reader = strings.NewReader(t.Encode())
	case map[string]interface{}:
		if raw, err := json.Marshal(t); err != nil {
			return nil, err
		} else {
			reader = bytes.NewReader(raw)
		}
	case io.Reader:
		reader = t
	case encoding.BinaryMarshaler:
		if raw, err := t.MarshalBinary(); err != nil {
			return nil, err
		} else {
			reader = bytes.NewReader(raw)
		}
	default:
		var buf = new(bytes.Buffer)
		encoder := gob.NewEncoder(buf)
		if err := encoder.Encode(data); err != nil {
			return nil, fmt.Errorf("can`t support[%T],try impl encoding.BinaryMarshaler", t)
		} else {
			reader = buf
		}
	}
	return reader, nil
}
func execute(_url, method string, body io.Reader, opts *Options) (_resp *Response, err error) {
	method, err = Method(method).Method()
	if err != nil {
		return nil, err
	}
	var client = defaultClient()
	var req = fasthttp.AcquireRequest()
	var resp = fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	if opts == nil {
		opts = DefaultRequestOptions()
	}
	if opts.MaxRedirect <= 0 {
		opts.MaxRedirect = defaultMaxRedirect
	}
	if opts.Timeout > 0 {
		req.SetTimeout(opts.Timeout)
	}
	if opts.ReadTimeout > 0 {
		client.ReadTimeout = opts.ReadTimeout
	}
	if opts.WriteTimeout > 0 {
		client.WriteTimeout = opts.WriteTimeout
	}
	if opts.Header != nil {
		req.Header = *opts.Header.Header()
	}
	req.Header.SetMethod(method)
	if opts.Proxy != "" {
		if strings.HasPrefix(opts.Proxy, "http") {
			client.DialTimeout = func(addr string, timeout time.Duration) (net.Conn, error) {
				return fasthttpproxy.FasthttpHTTPDialerTimeout(opts.Proxy, timeout)(addr)

			}
		} else if strings.HasPrefix(opts.Proxy, "socks5") {
			client.Dial = fasthttpproxy.FasthttpSocksDialer(opts.Proxy)
		} else {
			return nil, errors.New("proxy unsupported")
		}

	}
	req.SetRequestURI(_url)
	if body != nil {
		req.SetBodyStream(body, -1)
	}
	err = client.DoRedirects(req, resp, opts.MaxRedirect)
	if err != nil {
		return nil, err
	}
	return &Response{response: resp, opts: opts}, nil
}
func Get(url string, opt *Options) (resp *Response, err error) {
	return execute(url, "GET", nil, opt)
}
func Post(url string, body any, opt *Options) (resp *Response, err error) {
	reader, err := reader(body)
	if err != nil {
		return nil, err
	}
	return execute(url, http.MethodPost, reader, opt)
}
func PostJson(url string, body any, opts *Options) (resp *Response, err error) {
	if opts == nil {
		opts = DefaultRequestOptions()
	}
	opts.Header.SetContentType("application/json")
	return Post(url, body, opts)
}
func PostForm(url string, body any, opts *Options) (resp *Response, err error) {
	if opts == nil {
		opts = DefaultRequestOptions()
	}
	opts.Header.SetContentType("application/x-www-form-urlencoded")
	return Post(url, body, opts)
}
func Execute(url string, method string, body any, opt *Options) (resp *Response, err error) {
	reader, err := reader(body)
	if err != nil {
		return nil, err
	}
	return execute(url, method, reader, opt)
}

func NewHeaderValue(key string, values []string) *HeaderValue {
	return &HeaderValue{key: key, values: values}
}
func NewReqHeader() *ReqHeader {
	var header = &fasthttp.RequestHeader{}
	header.SetUserAgent(defaultUserAgent)
	header.SetMethod("GET")
	return &ReqHeader{header: header}
}
func Map2Headers(headers map[string]string) *ReqHeader {
	var header = NewReqHeader()
	for k, v := range headers {
		header.Set(k, v)
	}
	return header
}
func Map2Options(options map[string]any) *Options {
	var defaultOptions = DefaultRequestOptions()
	if options == nil {
		return defaultOptions
	}
	utils.MS(options, defaultOptions, nil)
	if v, ok := options["headers"]; ok {
		if h, ok := v.(map[string]string); ok {
			for k, v := range h {
				defaultOptions.Header.Set(k, v)
			}
		}
	}
	return defaultOptions

}
func DefaultRequestOptions() *Options {
	return config.TryValidate(&Options{Header: NewReqHeader()})
}
