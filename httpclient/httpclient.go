package httpclient

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/kevinyjn/gocom/definations"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/queues"
	"github.com/kevinyjn/gocom/utils"
)

// Constants
const (
	RetryDurationFactor = 5
)

type httpClientOption struct {
	headers     map[string]string
	tlsOptions  *definations.TLSOptions
	proxies     *definations.Proxies
	timeouts    time.Duration
	retries     int // retry times that already executed
	shouldRetry int // retry times that caller expectes
}

// ClientOption http client option
type ClientOption interface {
	apply(*httpClientOption)
}

type funcHTTPClientOption struct {
	f func(*httpClientOption)
}

func (fdo *funcHTTPClientOption) apply(do *httpClientOption) {
	fdo.f(do)
}

func newFuncHTTPClientOption(f func(*httpClientOption)) *funcHTTPClientOption {
	return &funcHTTPClientOption{
		f: f,
	}
}

func defaultHTTPClientOptions() httpClientOption {
	return httpClientOption{
		headers:    map[string]string{},
		tlsOptions: nil,
	}
}

func defaultHTTPClientJSONOptions() httpClientOption {
	return httpClientOption{
		headers: map[string]string{
			"Content-Type": "application/json",
		},
		tlsOptions: nil,
		timeouts:   time.Second * 30,
	}
}

// WithHTTPHeader options
func WithHTTPHeader(name, value string) ClientOption {
	return newFuncHTTPClientOption(func(o *httpClientOption) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		o.headers[name] = value
	})
}

// WithHTTPHeaders options
func WithHTTPHeaders(headers map[string]string) ClientOption {
	return newFuncHTTPClientOption(func(o *httpClientOption) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		for name, value := range headers {
			o.headers[name] = value
		}
	})
}

// WithHTTPHeadersEx options
func WithHTTPHeadersEx(headers map[string]interface{}) ClientOption {
	return newFuncHTTPClientOption(func(o *httpClientOption) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		for name, value := range headers {
			o.headers[name] = value.(string)
		}
	})
}

// WithHTTPTLSOptions options
func WithHTTPTLSOptions(tlsOptions *definations.TLSOptions) ClientOption {
	return newFuncHTTPClientOption(func(o *httpClientOption) {
		o.tlsOptions = tlsOptions
	})
}

// WithHTTPProxies options
func WithHTTPProxies(proxies *definations.Proxies) ClientOption {
	return newFuncHTTPClientOption(func(o *httpClientOption) {
		o.proxies = proxies
	})
}

// WithTimeout options
func WithTimeout(timeoutSeconds int) ClientOption {
	return newFuncHTTPClientOption(func(o *httpClientOption) {
		o.timeouts = time.Duration(timeoutSeconds) * time.Second
	})
}

// WithRetry options
func WithRetry(shouldRetryTimes int) ClientOption {
	return newFuncHTTPClientOption(func(o *httpClientOption) {
		o.shouldRetry = shouldRetryTimes
	})
}

// HTTPGet request
func HTTPGet(queryURL string, params *map[string]string, options ...ClientOption) ([]byte, error) {
	if params != nil {
		v := url.Values{}
		for pk, pv := range *params {
			v.Add(pk, pv)
		}
		urlParams := v.Encode()
		if urlParams != "" {
			sep := "?"
			if strings.Contains(queryURL, "?") {
				sep = "&"
			}
			queryURL = queryURL + sep + urlParams
		}
	}

	return HTTPQuery("GET", queryURL, nil, options...)
}

// HTTPGetJSON request and response as json
func HTTPGetJSON(queryURL string, params *map[string]string, options ...ClientOption) (map[string]interface{}, error) {
	options = append(options, WithHTTPHeader("Content-Type", "application/json"))

	resp, err := HTTPGet(queryURL, params, options...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		logger.Error.Printf("Parsing result queried from url:%s response:%s failed with error:%v", queryURL, string(resp), err)
		return nil, err
	}

	return result, nil
}

// HTTPGetJSONList request get json value list
func HTTPGetJSONList(queryURL string, params *map[string]interface{}, options ...ClientOption) ([]byte, error) {
	if params != nil {
		v := url.Values{}
		for pk, pv := range *params {
			if pk == "childRoute" {
				queryURL += fmt.Sprintf("/%v", pv)
				continue
			}
			if reflect.TypeOf(pv).Kind() == reflect.Map {
				for mk, mv := range pv.(map[string]interface{}) {
					vk := fmt.Sprintf(pk+"[%v]", mk)
					v.Add(vk, utils.ToString(mv))
				}
			} else {
				v.Add(pk, utils.ToString(pv))
			}
		}
		urlParams := v.Encode()
		if urlParams != "" {
			sep := "?"
			if strings.Contains(queryURL, "?") {
				sep = "&"
			}
			queryURL = queryURL + sep + urlParams
		}
	}
	logger.Trace.Printf("HTTPGetJSONList queryURL: %s", queryURL)
	return HTTPQuery("GET", queryURL, nil, options...)
}

// HTTPURLRequestWithoutBody URL parameter transfer without body
func HTTPURLRequestWithoutBody(method string, queryURL string, params *map[string]interface{}, options ...ClientOption) ([]byte, error) {
	if params != nil {
		v := url.Values{}
		for pk, pv := range *params {
			if nil == pv {
				continue
			}
			if reflect.TypeOf(pv).Kind() == reflect.Map {
				for mk, mv := range pv.(map[string]interface{}) {
					vk := fmt.Sprintf(pk+"[%v]", mk)
					v.Add(vk, utils.ToString(mv))
				}
			} else {
				v.Add(pk, utils.ToString(pv))
			}
		}
		urlParams := v.Encode()
		if urlParams != "" {
			sep := "?"
			if strings.Contains(queryURL, "?") {
				sep = "&"
			}
			queryURL = queryURL + sep + urlParams
		}
	}
	logger.Trace.Printf("HTTPURLRequestWithoutBody queryURL: %s", queryURL)
	return HTTPQuery(method, queryURL, nil, options...)
}

// HTTPPostJSON request and response as json
func HTTPPostJSON(queryURL string, params map[string]interface{}, options ...ClientOption) (map[string]interface{}, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	resp, err := HTTPQuery("POST", queryURL, bytes.NewReader(body), options...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		logger.Error.Printf("Parsing result queried from url:%s response:%s failed with error:%v", queryURL, string(resp), err)
		return nil, err
	}

	return result, nil
}

// HTTPPostJSONEx request and response as json
func HTTPPostJSONEx(queryURL string, params interface{}, result interface{}, options ...ClientOption) error {
	body, err := json.Marshal(params)
	if err != nil {
		return err
	}

	resp, err := HTTPQuery("POST", queryURL, bytes.NewReader(body), options...)
	if err != nil {
		return err
	}

	err = json.Unmarshal(resp, result)
	if err != nil {
		logger.Error.Printf("Parsing result queried from url:%s response:%s failed with error:%v", queryURL, string(resp), err)
		return err
	}

	return nil
}

// HTTPQuery request
func HTTPQuery(method string, queryURL string, body io.Reader, options ...ClientOption) ([]byte, error) {
	req, err := http.NewRequest(method, queryURL, body)
	if err != nil {
		logger.Error.Printf("Formatting query %s failed with error:%v", queryURL, err)
		return nil, err
	}
	opts := defaultHTTPClientJSONOptions()
	for _, opt := range options {
		opt.apply(&opts)
	}
	if opts.headers != nil {
		for hk, hv := range opts.headers {
			req.Header.Set(hk, hv)
		}
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	if opts.tlsOptions != nil && opts.tlsOptions.Enabled {
		certs, err := tls.LoadX509KeyPair(opts.tlsOptions.CertFile, opts.tlsOptions.KeyFile)
		if err != nil {
			logger.Error.Printf("Load tls certificates:%s and %s failed with error:%v", opts.tlsOptions.CertFile, opts.tlsOptions.KeyFile, err)
			return nil, err
		}

		// ca, err := x509.ParseCertificate(certs.Certificate[0])
		// if err != nil {
		// 	logger.Error.Printf("Parse certificate faield with error:%v", err)
		// } else {
		// 	caPool.AddCert(ca)
		// }

		if opts.tlsOptions.CaFile != "" {
			caData, err := ioutil.ReadFile(opts.tlsOptions.CaFile)
			if err != nil {
				logger.Error.Printf("Load tls root CA:%s failed with error:%v", opts.tlsOptions.CaFile, err)
				return nil, err
			}
			caPool := x509.NewCertPool()
			caPool.AppendCertsFromPEM(caData)
			tlsConfig.RootCAs = caPool
		}
		// tlsConfig.BuildNameToCertificate()

		tlsConfig.Certificates = []tls.Certificate{certs}
		tlsConfig.InsecureSkipVerify = opts.tlsOptions.SkipVerify
		// tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven

		// DEBUG for tls ca verify
		// tlsConfig.ServerName = "10.248.100.227"
		// req.Host = "10.248.100.227"
		// logger.Info.Printf("loaded tls certificates:%s and %s", opts.tlsOptions.CertFile, opts.tlsOptions.KeyFile)
	}
	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	if opts.proxies != nil && opts.proxies.Valid() {
		proxyURL, _ := url.Parse(opts.proxies.FetchProxyURL(queryURL))
		tr.Proxy = http.ProxyURL(proxyURL)
	}
	client := &http.Client{Transport: tr}
	if opts.timeouts > 0 {
		client.Timeout = opts.timeouts
	}

	// logger.Trace.Printf("querying %s...", queryURL)
	resp, err := client.Do(req)
	if err != nil {
		logger.Error.Printf("query %s failed with error:%v", queryURL, err)
		afterQueryFailed(-1, err, []byte(err.Error()), method, queryURL, body, &opts)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error.Printf("Read result by queried url:%s failed with error:%v", queryURL, err)
		afterQueryFailed(resp.StatusCode, err, []byte(err.Error()), method, queryURL, body, &opts)
		return nil, err
	}

	if resp.StatusCode != 200 {
		if resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound {
			newLocation := resp.Header.Get("location")
			logger.Info.Printf("query %s while got status:%d for location:%s", queryURL, resp.StatusCode, newLocation)
			if "" != newLocation {
				return HTTPQuery(method, newLocation, body, options...)
			}
		}
		err = errors.New(resp.Status)
		afterQueryFailed(resp.StatusCode, err, respBody, method, queryURL, body, &opts)
		return nil, err
	}

	if opts.retries > 0 {
		logger.Info.Printf("query %s with method:%s succeed with %d retries", queryURL, method, opts.retries)
	}

	return respBody, nil
}

func afterQueryFailed(respStatusCode int, err error, respBody []byte, method string, queryURL string, body io.Reader, opts *httpClientOption) {
	logger.Error.Output(2, fmt.Sprintf("Error: query %s failed with error(code:%d):%v body:%s", queryURL, respStatusCode, err, string(respBody)))
	if opts.shouldRetry > 0 {
		if opts.retries >= opts.shouldRetry {
			logger.Error.Printf("query %s failed with %d retries, skip retring", queryURL, opts.retries)
			return
		}
		var reqBody []byte
		if nil != body {
			reqBody, err = ioutil.ReadAll(body)
			if nil != err {
				logger.Error.Printf("format http retry request while get request body content failed with error:%v", err)
			}
		}
		retryDuration := time.Second * time.Duration(RetryDurationFactor) * time.Duration(opts.retries+1)
		now := time.Now()
		now.Add(retryDuration)
		re := &requestEntity{
			method:           method,
			url:              queryURL,
			body:             reqBody,
			options:          *opts,
			triggerTimestamp: now.Unix() + formatRetryDuration(opts.retries),
		}
		_pendingRequestsQueue.Push(re)
		if nil == _pendingRequestsTimer {
			go pendingRequestsTimer()
		}
	}
}

func formatRetryDuration(retries int) int64 {
	if retries < 3 {
		return RetryDurationFactor
	}
	return int64(RetryDurationFactor * retries)
}

func pendingRequestsTimer() {
	if nil != _pendingRequestsTimer {
		return
	}
	_pendingRequestsTimer = time.NewTicker(1 * time.Second)
	for nil != _pendingRequestsTimer {
		select {
		case tim := <-_pendingRequestsTimer.C:
			var ok = true
			var item interface{}
			now := tim.Unix()
			for ok {
				item, ok = _pendingRequestsQueue.Pop()
				if ok {
					ok = checkRetryEntity(item, now)
				}
			}
			break
		}
	}
}

func checkRetryEntity(item interface{}, tim int64) bool {
	re, ok := item.(*requestEntity)
	if false == ok {
		logger.Error.Printf("check retry http request entity while convert the element into request entity failed")
		return true
	}
	if re.triggerTimestamp <= tim {
		// do request
		opts := newFuncHTTPClientOption(func(o *httpClientOption) {
			o.headers = re.options.headers
			o.proxies = re.options.proxies
			o.retries = re.options.retries + 1
			o.shouldRetry = re.options.shouldRetry
			o.timeouts = re.options.timeouts
			o.tlsOptions = re.options.tlsOptions
		})
		logger.Info.Printf("retrying http request %s with method:%s ...", re.url, re.method)
		HTTPQuery(re.method, re.url, bytes.NewReader(re.body), opts)
		return true
	}
	_pendingRequestsQueue.Push(re)
	return false
}

type requestEntity struct {
	method           string
	url              string
	body             []byte
	options          httpClientOption
	triggerTimestamp int64
}

var (
	_pendingRequestsQueue              = queues.NewAscOrderingQueue()
	_pendingRequestsTimer *time.Ticker = nil
)

func (r *requestEntity) GetID() string {
	return r.url
}
func (r *requestEntity) GetName() string {
	return r.url
}
func (r *requestEntity) OrderingValue() int64 {
	return r.triggerTimestamp
}
func (r *requestEntity) DebugString() string {
	return r.url
}
