package httpnotifier

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"reflect"
	"strings"

	"github.com/kevinyjn/gocom/definations"
	"github.com/kevinyjn/gocom/httpclient"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/soapclient"
	"github.com/kevinyjn/gocom/utils/signature"
)

// Constants
const (
	WebServerTypeRestful = 1
	WebServerTypeSOAP    = 2

	SignaturePositionHeader = "header"
	SignaturePositionBody   = "body"

	SerializingTypeJSON = "json"
	SerializingTypeXML  = "xml"

	SignatureVersionMD5    = "1.0"
	SignatureVersionHMAC   = "2.0"
	SignatureVersionPUBKEY = "3.0"
)

// SignagureOption options
type SignagureOption struct {
	SignatureField    string
	SignaturePosition string
	SignatureVersion  string
	SignatureKey      string
	SigningFields     []string
	SortingFieldsType signature.SignFieldsSortType
	SkipEmptyField    bool
}

// HTTPNotifier notifier
type HTTPNotifier struct {
	Endpoint        string
	WebserverType   int
	TLSOptions      *definations.TLSOptions
	SignatureOption *SignagureOption
	Headers         map[string]string
	SerializingType string
	SoapPortName    string
	Proxies         *definations.Proxies
}

type customHTTPNotifierOption struct {
	tlsOptions      *definations.TLSOptions
	signatureOption *SignagureOption
	headers         map[string]string
	serializingType string
	proxies         *definations.Proxies
}

// CustomHTTPNotifierOption options
type CustomHTTPNotifierOption interface {
	apply(*customHTTPNotifierOption)
}

type funcCustomHTTPNotifierOption struct {
	f func(*customHTTPNotifierOption)
}

func (fdo *funcCustomHTTPNotifierOption) apply(do *customHTTPNotifierOption) {
	fdo.f(do)
}

func newCustomHTTPNotifierOption(f func(*customHTTPNotifierOption)) *funcCustomHTTPNotifierOption {
	return &funcCustomHTTPNotifierOption{
		f: f,
	}
}

func defaultCustomHTTPNotifierOptionJSON() customHTTPNotifierOption {
	return customHTTPNotifierOption{
		tlsOptions:      nil,
		signatureOption: nil,
		headers:         make(map[string]string),
		serializingType: SerializingTypeJSON,
		proxies:         nil,
	}
}

func defaultCustomHTTPNotifierOptionXML() customHTTPNotifierOption {
	return customHTTPNotifierOption{
		tlsOptions:      nil,
		signatureOption: nil,
		headers:         make(map[string]string),
		serializingType: SerializingTypeXML,
		proxies:         nil,
	}
}

// WithTLSConfig options
func WithTLSConfig(tlsOptions *definations.TLSOptions) CustomHTTPNotifierOption {
	return newCustomHTTPNotifierOption(func(o *customHTTPNotifierOption) {
		o.tlsOptions = tlsOptions
	})
}

// WithSignatureConfig options
func WithSignatureConfig(signatureOption *SignagureOption) CustomHTTPNotifierOption {
	return newCustomHTTPNotifierOption(func(o *customHTTPNotifierOption) {
		o.signatureOption = signatureOption
	})
}

// WithHTTPHeader options
func WithHTTPHeader(name, value string) CustomHTTPNotifierOption {
	return newCustomHTTPNotifierOption(func(o *customHTTPNotifierOption) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		o.headers[name] = value
	})
}

// WithHTTPHeaders options
func WithHTTPHeaders(headers map[string]string) CustomHTTPNotifierOption {
	return newCustomHTTPNotifierOption(func(o *customHTTPNotifierOption) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		for name, value := range headers {
			o.headers[name] = value
		}
	})
}

// WithSerializingType options
func WithSerializingType(serializingType string) CustomHTTPNotifierOption {
	return newCustomHTTPNotifierOption(func(o *customHTTPNotifierOption) {
		o.serializingType = serializingType
	})
}

// WithHTTPProxies options
func WithHTTPProxies(proxies *definations.Proxies) CustomHTTPNotifierOption {
	return newCustomHTTPNotifierOption(func(o *customHTTPNotifierOption) {
		o.proxies = proxies
	})
}

type customQueryOption struct {
	soapEvpHeaders []interface{}
}

// CustomQueryOption options
type CustomQueryOption interface {
	apply(*customQueryOption)
}

type funcCustomQueryOption struct {
	f func(*customQueryOption)
}

func (fdo *funcCustomQueryOption) apply(do *customQueryOption) {
	fdo.f(do)
}

func newHTTPNotifierCustomQueryOption(f func(*customQueryOption)) *funcCustomQueryOption {
	return &funcCustomQueryOption{
		f: f,
	}
}

func defaultCustomQueryOption() customQueryOption {
	return customQueryOption{
		soapEvpHeaders: []interface{}{},
	}
}

// WithSoapEvpHeaders options
func WithSoapEvpHeaders(soapEvpHeaders []interface{}) CustomQueryOption {
	return newHTTPNotifierCustomQueryOption(func(o *customQueryOption) {
		o.soapEvpHeaders = soapEvpHeaders
	})
}

// CreateRestHTTPNotifier creator
func CreateRestHTTPNotifier(endpoint string, options ...CustomHTTPNotifierOption) *HTTPNotifier {
	opts := defaultCustomHTTPNotifierOptionJSON()
	for _, opt := range options {
		opt.apply(&opts)
	}
	s := &HTTPNotifier{
		Endpoint:        endpoint,
		WebserverType:   WebServerTypeRestful,
		Headers:         opts.headers,
		SerializingType: opts.serializingType,
		SignatureOption: opts.signatureOption,
		TLSOptions:      opts.tlsOptions,
		Proxies:         opts.proxies,
	}
	return s
}

// CreateSOAPHTTPNotifier creator
func CreateSOAPHTTPNotifier(endpoint string, soapPortName string, options ...CustomHTTPNotifierOption) *HTTPNotifier {
	opts := defaultCustomHTTPNotifierOptionXML()
	for _, opt := range options {
		opt.apply(&opts)
	}
	s := &HTTPNotifier{
		Endpoint:        endpoint,
		WebserverType:   WebServerTypeSOAP,
		SoapPortName:    soapPortName,
		Headers:         opts.headers,
		SerializingType: opts.serializingType,
		SignatureOption: opts.signatureOption,
		TLSOptions:      opts.tlsOptions,
		Proxies:         opts.proxies,
	}
	return s
}

// DoQuery request
func (s *HTTPNotifier) DoQuery(action string, payload interface{}, response interface{}, options ...CustomQueryOption) error {
	queryURL := s.Endpoint
	headers := s.Headers
	var err error
	opts := defaultCustomQueryOption()
	for _, opt := range options {
		opt.apply(&opts)
	}
	if action != "" {
		if s.WebserverType != WebServerTypeSOAP {
			queryURL = queryURL + action
		}
	}
	if s.SignatureOption != nil {
		so := s.SignatureOption
		// generate signature
		sign := SignagurePayloadData(payload, so)
		if so.SignaturePosition == SignaturePositionHeader {
			headers[so.SignatureField] = sign
		} else {
			// set signature to body
			value := reflect.ValueOf(payload)
			if value.Type().Kind() == reflect.Ptr {
				value = value.Elem()
			}
			if value.Type().Kind() == reflect.Struct {
				for i := 0; i < value.NumField(); i++ {
					f := value.Field(i)
					ft := value.Type().Field(i)
					tag := ft.Tag
					tagName := ft.Tag.Get(s.SerializingType)
					if tag == "" {
						fieldName := ft.Name
						if fieldName[0] >= 'A' && fieldName[0] <= 'Z' {
							tagName = strings.ToLower(fieldName[:1]) + fieldName[1:]
						}
					}
					if tagName == "" {
						continue
					}
					if strings.Contains(tagName, ",") {
						tagName = strings.Split(tagName, ",")[0]
					}
					if tagName == so.SignatureField {
						f.SetString(sign)
						break
					}
				}
			} else if value.Type().Kind() == reflect.Map {
				value.SetMapIndex(reflect.ValueOf(so.SignatureField), reflect.ValueOf(sign))
			}
		}
	}

	if s.WebserverType == WebServerTypeRestful {
		var bs []byte
		switch s.SerializingType {
		case SerializingTypeJSON:
			bs, err = json.Marshal(payload)
			break
		case SerializingTypeXML:
			bs, err = xml.Marshal(payload)
			break
		default:
			err = fmt.Errorf("unsupported serializing type:%s", s.SerializingType)
		}
		if err != nil {
			logger.Error.Printf("serializing payload data as %s failed with error:%v", s.SerializingType, err)
			return err
		}

		resp, err := httpclient.HTTPQuery("POST", queryURL, bytes.NewReader(bs),
			httpclient.WithHTTPHeaders(headers),
			httpclient.WithHTTPTLSOptions(s.TLSOptions),
			httpclient.WithHTTPProxies(s.Proxies),
		)
		if err != nil {
			return err
		}
		switch s.SerializingType {
		case SerializingTypeJSON:
			err = json.Unmarshal(resp, response)
		case SerializingTypeXML:
			err = xml.Unmarshal(resp, response)
		default:
			err = fmt.Errorf("unsupported serializing type:%s", s.SerializingType)
		}
	} else if s.WebserverType == WebServerTypeSOAP {
		sc, err := soapclient.NewSOAPClient(s.Endpoint,
			soapclient.WithHTTPHeaders(headers),
			soapclient.WithHTTPTLSOptions(s.TLSOptions),
			soapclient.WithHTTPProxies(s.Proxies),
		)
		if err != nil {
			logger.Error.Printf("Open soap client by endpoint:%s failed with error:%v", s.Endpoint, err)
			return err
		}

		sp := sc.GetPort(s.SoapPortName)
		if sp == nil {
			logger.Error.Printf("Call soap operation:%s on port:%s failed while the port does not exists", action, s.SoapPortName)
			return fmt.Errorf("No soap port:%s", s.SoapPortName)
		}
		err = sp.Call(action, response, payload, opts.soapEvpHeaders)
		if err != nil {
			logger.Error.Printf("Call soap operation:%s on port:%s failed with error:%v", action, s.SoapPortName, err)
			return err
		}
	} else {
		return fmt.Errorf("Unsupported webserver type:%d", s.WebserverType)
	}

	return err
}

// SignagurePayloadData signature
func SignagurePayloadData(payload interface{}, so *SignagureOption) string {
	var sign string
	if so.SignatureVersion == SignatureVersionMD5 {
		sign = signature.GenerateSignatureDataMd5Ex(so.SignatureKey, payload,
			signature.WithSignaturingFields(so.SigningFields),
			signature.WithSignatureField(so.SignatureField),
			signature.WithSortedFields(so.SortingFieldsType),
			signature.WithSkipEmptyField(so.SkipEmptyField),
			signature.WithSkipSignaturingField(so.SignatureField),
		)
	} else if so.SignatureVersion == SignatureVersionHMAC {
		sign = signature.GenerateSignatureDataHmacSHA1Ex(so.SignatureKey, payload,
			signature.WithSignaturingFields(so.SigningFields),
			signature.WithSignatureField(so.SignatureField),
			signature.WithSortedFields(so.SortingFieldsType),
			signature.WithSkipEmptyField(so.SkipEmptyField),
			signature.WithSkipSignaturingField(so.SignatureField),
		)
	}
	return sign
}
