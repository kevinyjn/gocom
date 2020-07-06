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

	"github.com/kevinyjn/gocom/definations"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/utils"
)

type httpClientOption struct {
	headers    map[string]string
	tlsOptions *definations.TLSOptions
	proxies    *definations.Proxies
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
		logger.Error.Printf("Parsing result queried from url:%s response:%s failed with error:%s", queryURL, string(resp), err.Error())
		return nil, err
	}

	return result, nil
}

// HTTPGetJSONList request get json value list
func HTTPGetJSONList(queryURL string, params *map[string]interface{}, options ...ClientOption) ([]byte, error) {
	if params != nil {
		v := url.Values{}
		for pk, pv := range *params {
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
		logger.Error.Printf("Parsing result queried from url:%s response:%s failed with error:%s", queryURL, string(resp), err.Error())
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
		logger.Error.Printf("Parsing result queried from url:%s response:%s failed with error:%s", queryURL, string(resp), err.Error())
		return err
	}

	return nil
}

// HTTPQuery request
func HTTPQuery(method string, queryURL string, body io.Reader, options ...ClientOption) ([]byte, error) {
	req, err := http.NewRequest(method, queryURL, body)
	if err != nil {
		logger.Error.Printf("Formatting query %s failed with error:%s", queryURL, err.Error())
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
			logger.Error.Printf("Load tls certificates:%s and %s failed with error:%s", opts.tlsOptions.CertFile, opts.tlsOptions.KeyFile, err.Error())
			return nil, err
		}

		// ca, err := x509.ParseCertificate(certs.Certificate[0])
		// if err != nil {
		// 	logger.Error.Printf("Parse certificate faield with error:%s", err.Error())
		// } else {
		// 	caPool.AddCert(ca)
		// }

		if opts.tlsOptions.CaFile != "" {
			caData, err := ioutil.ReadFile(opts.tlsOptions.CaFile)
			if err != nil {
				logger.Error.Printf("Load tls root CA:%s failed with error:%s", opts.tlsOptions.CaFile, err.Error())
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
		proxyUrl, _ := url.Parse(opts.proxies.FetchProxyURL(queryURL))
		tr.Proxy = http.ProxyURL(proxyUrl)
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Do(req)
	if err != nil {
		logger.Error.Printf("query %s failed with error:%s", queryURL, err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error.Printf("Read result by queried url:%s failed with error:%s", queryURL, err.Error())
		return nil, err
	}

	if resp.StatusCode != 200 {
		logger.Error.Printf("Error: query %s failed with error:%s body:%s", queryURL, resp.Status, string(respBody))
		return nil, errors.New(resp.Status)
	}

	return respBody, nil
}
