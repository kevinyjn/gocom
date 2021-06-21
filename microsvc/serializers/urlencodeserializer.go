package serializers

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/kevinyjn/gocom/utils"
)

// URLEncodeParam serializable paremeter wrapper
type URLEncodeParam struct {
	DataPtr interface{}
}

// NewURLEncodeParam serializable parameter wrapper
func NewURLEncodeParam(data interface{}) *URLEncodeParam {
	return &URLEncodeParam{DataPtr: data}
}

// GetURLEncodeSerializer instance
func GetURLEncodeSerializer() Serializable {
	return _urlEncodeSerializer
}

// urlEncodeSerializer json serializing instance
type urlEncodeSerializer struct{}

var _urlEncodeSerializer = &urlEncodeSerializer{}

// ContentType serialization content type name
func (p *URLEncodeParam) ContentType() string {
	return "application/x-www-form-urlencoded"
}

// Serialize serialize the payload data as json content
func (p *URLEncodeParam) Serialize() ([]byte, error) {
	return marshalURLEncode(p.DataPtr)
}

// ParseFrom parse payload data from json content
func (p *URLEncodeParam) ParseFrom(buffer []byte) error {
	return unmarshalURLEncode(buffer, p.DataPtr)
}

// ContentType serialization content type name
func (s *urlEncodeSerializer) ContentType() string {
	return "application/x-www-form-urlencoded"
}

// Serialize serialize the payload data as json content
func (s *urlEncodeSerializer) Serialize(payloadObject interface{}) ([]byte, error) {
	return marshalURLEncode(payloadObject)
}

// ParseFrom parse payload data from json content
func (s *urlEncodeSerializer) ParseFrom(buffer []byte, payloadObject interface{}) error {
	return unmarshalURLEncode(buffer, payloadObject)
}

func marshalURLEncode(payload interface{}) ([]byte, error) {
	paramPieces := [][]byte{}

	v := reflect.ValueOf(payload)
	t := reflect.TypeOf(payload)
	if v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	n := t.NumField()
	for i := 0; i < n; i++ {
		ft := t.Field(i)
		fv := v.Field(i)
		tagValue := ft.Tag.Get("form")
		isOmitempty := false
		if "" == tagValue {
			tagValue = strings.ToLower(ft.Name[0:1]) + ft.Name[1:]
		} else if "-" == tagValue {
			continue
		} else {
			isOmitempty = strings.HasSuffix(tagValue, ",omitempty")
			tagValue = strings.SplitN(tagValue, ",", 2)[0]
		}
		val := ""
		switch ft.Type.Kind() {
		case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int8:
			val = utils.ToString(fv.Interface())
			break
		case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8:
			val = utils.ToString(fv.Interface())
			break
		case reflect.String:
			val = fv.String()
			break
		case reflect.Bool:
			val = utils.ToString(fv.Interface())
			break
		case reflect.Float64, reflect.Float32:
			val = utils.ToString(fv.Interface())
			break
		default:
			// pass
			break
		}
		if isOmitempty && "" == val {
			continue
		}
		paramPieces = append(paramPieces, []byte(tagValue+"="+utils.URLEncode(val)))
	}
	return bytes.Join(paramPieces, []byte{'&'}), nil
}

func unmarshalURLEncode(buffer []byte, payload interface{}) error {
	paramPieces := bytes.Split(buffer, []byte{'&'})
	values := map[string][]string{}
	for _, paramPiece := range paramPieces {
		parts := bytes.Split(paramPiece, []byte{'='})
		name := string(parts[0])
		var val string
		if len(parts) > 1 {
			val = utils.URLDecode(string(parts[1]))
		}
		if _, ok := values[name]; ok {
			values[name] = append(values[name], val)
		} else {
			values[name] = []string{val}
		}
	}

	v := reflect.ValueOf(payload)
	t := reflect.TypeOf(payload)
	if v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	n := t.NumField()
	for i := 0; i < n; i++ {
		ft := t.Field(i)
		fv := v.Field(i)
		tagValue := ft.Tag.Get("form")
		if "" == tagValue {
			tagValue = strings.ToLower(ft.Name[0:1]) + ft.Name[1:]
		} else {
			tagValue = strings.SplitN(tagValue, ",", 2)[0]
		}
		val, ok := values[tagValue]
		if ok {
			switch ft.Type.Kind() {
			case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int8:
				iv, err := strconv.ParseInt(val[0], 10, 64)
				if nil != err {
					return fmt.Errorf("Cannot unmarshal field %s value %s as integer type", tagValue, val[0])
				}
				fv.SetInt(iv)
				break
			case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8:
				iv, err := strconv.ParseUint(val[0], 10, 64)
				if nil != err {
					return fmt.Errorf("Cannot unmarshal field %s value %s as unsigned integer type", tagValue, val[0])
				}
				fv.SetUint(iv)
				break
			case reflect.String:
				fv.SetString(val[0])
				break
			case reflect.Bool:
				fv.SetBool(utils.ToBoolean(val[0]))
				break
			case reflect.Float64, reflect.Float32:
				iv, err := strconv.ParseFloat(val[0], 64)
				if nil != err {
					return fmt.Errorf("Cannot unmarshal field %s value %s as float type", tagValue, val[0])
				}
				fv.SetFloat(iv)
				break
			default:
				// pass
				break
			}
		}
	}
	return nil
}
