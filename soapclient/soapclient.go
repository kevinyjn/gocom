package soapclient

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/xml"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/kevinyjn/gocom/definations"
	"github.com/kevinyjn/gocom/logger"
)

// 因开源soap库无法正确访问业务，这里封装个原生点的代用.

// DebugSoapResponse debugger flag
var DebugSoapResponse = false

type soapClientOption struct {
	headers        map[string]string
	tlsOptions     *definations.TLSOptions
	proxies        *definations.Proxies
	timeoutSeconds int
}

// ClientOption options
type ClientOption interface {
	apply(*soapClientOption)
}

type funcSOAPClientOption struct {
	f func(*soapClientOption)
}

func (fdo *funcSOAPClientOption) apply(do *soapClientOption) {
	fdo.f(do)
}

func newFuncSOAPClientOption(f func(*soapClientOption)) *funcSOAPClientOption {
	return &funcSOAPClientOption{
		f: f,
	}
}

func defaultSOAPClientOption() soapClientOption {
	return soapClientOption{
		headers: map[string]string{
			"Content-Type": "text/xml; charset=\"utf-8\"",
		},
		tlsOptions:     nil,
		timeoutSeconds: 30,
	}
}

// WithHTTPHeader options
func WithHTTPHeader(name, value string) ClientOption {
	return newFuncSOAPClientOption(func(o *soapClientOption) {
		o.headers[name] = value
	})
}

// WithHTTPHeaders options
func WithHTTPHeaders(headers map[string]string) ClientOption {
	return newFuncSOAPClientOption(func(o *soapClientOption) {
		for name, value := range headers {
			o.headers[name] = value
		}
	})
}

// WithHTTPTLSOptions options
func WithHTTPTLSOptions(tlsOptions *definations.TLSOptions) ClientOption {
	return newFuncSOAPClientOption(func(o *soapClientOption) {
		o.tlsOptions = tlsOptions
	})
}

// WithHTTPProxies options
func WithHTTPProxies(proxies *definations.Proxies) ClientOption {
	return newFuncSOAPClientOption(func(o *soapClientOption) {
		o.proxies = proxies
	})
}

// WithTimeoutSeconds options
func WithTimeoutSeconds(timeoutSeconds int) ClientOption {
	return newFuncSOAPClientOption(func(o *soapClientOption) {
		o.timeoutSeconds = timeoutSeconds
	})
}

// SOAPClient client
type SOAPClient struct {
	client     *http.Client
	endpoint   string
	TTL        int
	createdAt  int64
	services   map[string]*SOAPService
	defService *SOAPService
}

// SOAPService service
type SOAPService struct {
	name  string
	ports map[string]*SOAPPort
}

// SOAPPort port
type SOAPPort struct {
	endpoint   string
	name       string
	binding    string
	client     *http.Client
	operations map[string]*soapOperation
}

type soapOperation struct {
	name       string
	soapAction string
}

var soapClients = make(map[string]*SOAPClient)

// NewSOAPClient new client
func NewSOAPClient(endpoint string, options ...ClientOption) (*SOAPClient, error) {
	c, ok := soapClients[endpoint]
	now := time.Now().Unix()
	if ok && c.ActiveState(now) {
		return c, nil
	}
	delete(soapClients, endpoint)

	opts := defaultSOAPClientOption()
	for _, opt := range options {
		opt.apply(&opts)
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	if opts.tlsOptions != nil && opts.tlsOptions.Enabled {
		certs, err := tls.LoadX509KeyPair(opts.tlsOptions.CertFile, opts.tlsOptions.KeyFile)
		if err != nil {
			logger.Error.Printf("Load tls certificates:%s and %s failed with error:%s", opts.tlsOptions.CertFile, opts.tlsOptions.KeyFile, err.Error())
			return nil, err
		}

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

		tlsConfig.Certificates = []tls.Certificate{certs}
		tlsConfig.InsecureSkipVerify = opts.tlsOptions.SkipVerify
	}
	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	if opts.proxies != nil && opts.proxies.Valid() {
		proxyURL, _ := url.Parse(opts.proxies.FetchProxyURL(endpoint))
		tr.Proxy = http.ProxyURL(proxyURL)
	}
	client := &http.Client{Transport: tr, Timeout: time.Second * time.Duration(opts.timeoutSeconds)}

	c = &SOAPClient{
		client:    client,
		endpoint:  endpoint,
		services:  make(map[string]*SOAPService),
		TTL:       600,
		createdAt: now,
	}
	if err := c.Init(); err != nil {
		return nil, err
	}

	soapClients[endpoint] = c
	return c, nil
}

// Init initializer
func (c *SOAPClient) Init() error {
	wsdlResp, err := c.client.Get(c.endpoint)
	if err != nil {
		logger.Error.Printf("Get wsdl by endpoint:%s failed with error:%s", c.endpoint, err.Error())
		return err
	}
	defer wsdlResp.Body.Close()

	wsdlBody, err := ioutil.ReadAll(wsdlResp.Body)
	if err != nil {
		logger.Error.Printf("Read wsdl by endpoint:%s body failed with error:%s", c.endpoint, err.Error())
		return err
	}

	// logger.Trace.Printf("Body of endpoint:%s", string(wsdlBody))
	wsdl := wsdlObject{}

	err = xml.Unmarshal(wsdlBody, &wsdl)
	if err != nil {
		logger.Error.Printf("Parse wsdl by endpoint:%s body failed with error:%s", c.endpoint, err.Error())
		return err
	}

	c.defService = nil
	for _, _ws := range wsdl.Services {
		s := &SOAPService{
			ports: make(map[string]*SOAPPort),
			name:  _ws.Name,
		}
		for _, _wp := range _ws.Ports {
			p := &SOAPPort{
				name:       _wp.Name,
				binding:    _wp.Binding,
				endpoint:   _wp.Address.Location,
				client:     c.client,
				operations: make(map[string]*soapOperation),
			}

			s.ports[p.name] = p
		}

		if c.defService == nil {
			c.defService = s
		}
		c.services[s.name] = s
	}

	for _, _wb := range wsdl.Bindings {
		ops := []*soapOperation{}
		for _, _wo := range _wb.Operations {
			ops = append(ops, &soapOperation{
				name:       _wo.Name,
				soapAction: _wo.Operation.SoapAction,
			})
		}
		for _, s := range c.services {
			for _, p := range s.ports {
				if p.binding == "tns:"+_wb.Name {
					for _, _wo := range ops {
						p.operations[_wo.name] = _wo
						// logger.Trace.Printf("add operation:%s soap_action:%s to port:%s with binding:%s", _wo.name, _wo.soapAction, p.name, p.binding)
					}
				}
			}
		}
	}

	if c.defService != nil {
		logger.Trace.Printf("Fetch service:%s as default soap service", c.defService.name)
	} else {
		logger.Error.Printf("Parse wsdl by endpoint:%s body while not found any servcie", c.endpoint)
		return errors.New("Not found any service")
	}

	return nil
}

// ActiveState state
func (c *SOAPClient) ActiveState(now int64) bool {
	if now-c.createdAt < int64(c.TTL) {
		return true
	}
	return false
}

// GetService getter
func (c *SOAPClient) GetService(name string) *SOAPService {
	if name == "" {
		return c.defService
	}
	s := c.services[name]
	return s
}

// GetPort getter
func (c *SOAPClient) GetPort(name string) *SOAPPort {
	if c.defService == nil {
		logger.Error.Printf("Get port:%s by default service while the default service is empty", name)
		return nil
	}
	return c.defService.GetPort(name)
}

// GetPort getter
func (s *SOAPService) GetPort(name string) *SOAPPort {
	p := s.ports[name]
	return p
}

// Call caller
func (p *SOAPPort) Call(operation string, response interface{}, soapBody interface{}, soapHeaders []interface{}) error {
	op, ok := p.operations[operation]
	if !ok {
		logger.Error.Printf("Call %s on port:%s while there is no such operation on the port.", operation, p.name)
		return errors.New("No such operation")
	}
	logger.Trace.Printf("calling operation:%s by soapAction:%s...", operation, op.soapAction)

	evp := NewSOAPEnvelope(soapBody, soapHeaders)
	bodyBuf, err := xml.Marshal(evp)
	if err != nil {
		logger.Error.Printf("Serialize operation:%s soap body failed with error:%s", operation, err.Error())
		return err
	}
	httpBody := "<?xml version='1.0' encoding='utf-8'?>\n" + string(bodyBuf)
	if DebugSoapResponse {
		logger.Trace.Printf("formatted soap body:\n%s", string(httpBody))
	}

	req, err := http.NewRequest("POST", p.endpoint, bytes.NewReader([]byte(httpBody)))
	if err != nil {
		logger.Error.Printf("Formatting query %s failed with error:%s", p.endpoint, err.Error())
		return err
	}
	req.Header.Set("Content-Type", "text/xml; charset=\"utf-8\"")
	req.Header.Set("SOAPAction", op.soapAction)
	req.Header.Set("User-Agent", "gowsdl/0.1")
	resp, err := p.client.Do(req)
	if err != nil {
		logger.Error.Printf("Query soap operation:%s failed with error:%s", operation, err.Error())
		return err
	}
	defer resp.Body.Close()

	mtomBoundary, err := getMtomHeader(resp.Header.Get("Content-Type"))
	if err != nil {
		logger.Error.Printf("Query soap operation:%s while decode response Content-Type failed with error:%s", operation, err.Error())
		return err
	}

	var iobuffer io.Reader
	if DebugSoapResponse {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Error.Printf("Read result by queried url:%s failed with error:%s", p.endpoint, err.Error())
			return err
		}
		logger.Trace.Printf("Query soap operation:%s soap_action:%s status:%d message:%s body:\n%s", operation, op.soapAction, resp.StatusCode, resp.Status, string(respBody))
		iobuffer = bytes.NewReader(respBody)
	} else {
		iobuffer = resp.Body
	}

	respEnvelope := new(SOAPEnvelopeResponse)
	respEnvelope.Body = SOAPBodyResponse{Content: response}

	var dec SOAPDecoder
	if mtomBoundary != "" {
		dec = newMtomDecoder(iobuffer, mtomBoundary)
	} else {
		dec = xml.NewDecoder(iobuffer)
	}

	if err := dec.Decode(respEnvelope); err != nil {
		logger.Error.Printf("Query soap operation:%s while decode response body failed with error:%s", operation, err.Error())
		return err
	}

	fault := respEnvelope.Body.Fault
	if fault != nil {
		logger.Error.Printf("Query soap operation:%s while response got fault:%s", operation, fault.Error())
		return fault
	}

	return nil
}
