package signature

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/kevinyjn/gocom/utils"
)

// SignFieldsSortType signature fields sorting type
type SignFieldsSortType int

// Consts
const (
	SignFieldsSortNone = SignFieldsSortType(0)
	SignFieldsSortAsc  = SignFieldsSortType(1)
	SignFieldsSortDesc = SignFieldsSortType(-1)

	SignFieldsFormattingModeDefault = 0
	DefaultSignatureField           = "sign"

	SerializingTypeJSON = "json"
	SerializingTypeXML  = "xml"
)

var (
	signatureGettingTags = []string{SerializingTypeJSON, SerializingTypeXML}
)

type signatureOption struct {
	SignaturingFields     []string
	SignatureField        string
	SortedFields          SignFieldsSortType
	SkipEmptyField        bool
	FormattingMode        int
	SkipSignaturingFields map[string]bool
	SerializingType       string
}

// SigningOption customizing options
type SigningOption interface {
	apply(*signatureOption)
}

type funcSignatureOption struct {
	f func(*signatureOption)
}

func (fdo *funcSignatureOption) apply(do *signatureOption) {
	fdo.f(do)
}

func newFuncSignagureOption(f func(*signatureOption)) *funcSignatureOption {
	return &funcSignatureOption{
		f: f,
	}
}

func defaultSignagureOptions() signatureOption {
	return signatureOption{
		SignaturingFields:     []string{},
		SignatureField:        DefaultSignatureField,
		SortedFields:          SignFieldsSortAsc,
		SkipEmptyField:        true,
		FormattingMode:        SignFieldsFormattingModeDefault,
		SkipSignaturingFields: map[string]bool{},
		SerializingType:       SerializingTypeJSON,
	}
}

// WithSignaturingFields option
func WithSignaturingFields(fields []string) SigningOption {
	return newFuncSignagureOption(func(o *signatureOption) {
		o.SignaturingFields = fields
	})
}

// WithSignatureField option
func WithSignatureField(field string) SigningOption {
	return newFuncSignagureOption(func(o *signatureOption) {
		o.SignatureField = field
	})
}

// WithSortedFields option
func WithSortedFields(sortType SignFieldsSortType) SigningOption {
	return newFuncSignagureOption(func(o *signatureOption) {
		o.SortedFields = sortType
	})
}

// WithSkipEmptyField option
func WithSkipEmptyField(skip bool) SigningOption {
	return newFuncSignagureOption(func(o *signatureOption) {
		o.SkipEmptyField = skip
	})
}

// WithSkipSignaturingField option
func WithSkipSignaturingField(key string) SigningOption {
	return newFuncSignagureOption(func(o *signatureOption) {
		o.SkipSignaturingFields[key] = true
	})
}

// FormatSignatureContent formatting content
func FormatSignatureContent(params map[string]interface{}, configures ...SigningOption) string {
	// options
	opts := defaultSignagureOptions()
	for _, opt := range configures {
		opt.apply(&opts)
	}

	var keySlices []string
	var signSlices []string
	if len(opts.SignaturingFields) == 0 {
		for key := range params {
			if key == opts.SignatureField {
				continue
			}
			if opts.SkipEmptyField && utils.ToString(params[key]) == "" {
				// fmt.Printf("Skip the empty field:%s\n", key)
				continue
			}
			if opts.SkipSignaturingFields[key] {
				continue
			}
			keySlices = append(keySlices, key)
		}
	} else {
		for _, key := range opts.SignaturingFields {
			if opts.SkipEmptyField && params[key] == nil {
				continue
			}
			keySlices = append(keySlices, key)
		}
	}

	if opts.SortedFields > 0 {
		sort.Strings(keySlices)
	} else if opts.SortedFields < 0 {
		sort.Sort(sort.Reverse(sort.StringSlice(keySlices)))
	}

	var signContent string
	if opts.FormattingMode == SignFieldsFormattingModeDefault {
		for _, v := range keySlices {
			signSlices = append(signSlices, fmt.Sprintf("%s=%s", v, utils.ToString(params[v])))
		}
		signContent = strings.Join(signSlices, "&")
	} else {
		for _, v := range keySlices {
			signSlices = append(signSlices, fmt.Sprintf("%s=%s", v, utils.ToString(params[v])))
		}
		signContent = strings.Join(signSlices, "&")
	}
	// fmt.Printf("slicesize:%d formatted: %s\n", len(signSlices), signContent)
	return signContent
}

// FormatSignatureContentEx formatting content
func FormatSignatureContentEx(payload interface{}, configures ...SigningOption) string {
	var signingContent string
	value := reflect.ValueOf(payload)
	if value.Type().Kind() == reflect.String {
		signingContent = value.String()
	} else {
		mappings := GeneratePayloadMappingData(value, "data")
		fieldValues := map[string]interface{}{}
		orgnizeSignaturingMappingData(mappings, fieldValues)
		signingContent = FormatSignatureContent(fieldValues, configures...)
	}
	return signingContent
}

func orgnizeSignaturingMappingData(v map[string]interface{}, result map[string]interface{}) {
	for k, v := range v {
		_, ok := result[k]
		if ok {
			continue
		}
		value := reflect.ValueOf(v)
		if value.Type().Kind() == reflect.Ptr {
			result[k] = utils.ToString(v)
		} else if value.Type().Kind() == reflect.Map {
			it := value.MapRange()
			r2 := map[string]interface{}{}
			for it.Next() {
				r2[it.Key().String()] = it.Value().Interface()
			}
			orgnizeSignaturingMappingData(r2, result)
		} else {
			result[k] = utils.ToString(v)
		}
	}
}

// GeneratePayloadMappingData generates payload data
func GeneratePayloadMappingData(value reflect.Value, parentTagName string) map[string]interface{} {
	result := map[string]interface{}{}
	if value.Type().Kind() == reflect.Ptr {
		value = value.Elem()
	}

	switch value.Type().Kind() {
	case reflect.Struct:
		{
			for i := 0; i < value.NumField(); i++ {
				f := value.Field(i)
				if f.Type().Kind() == reflect.Ptr {
					f = f.Elem()
					if !f.IsValid() {
						continue
					}
				}
				ft := value.Type().Field(i)
				tag := ft.Tag
				var tagName string
				if tag == "" {
					fieldName := ft.Name
					if fieldName[0] >= 'A' && fieldName[0] <= 'Z' {
						tagName = strings.ToLower(fieldName[:1]) + fieldName[1:]
					}
				} else {
					for _, tag2 := range signatureGettingTags {
						tagName = ft.Tag.Get(tag2)
						if tagName != "" {
							break
						}
					}
					if tagName == "-" {
						tagName = ""
					}
				}
				if tagName == "" {
					continue
				}

				if f.Type().Kind() == reflect.Struct {
					result[tagName] = GeneratePayloadMappingData(f, tagName)
				} else {
					result[tagName] = f.Interface()
				}
			}
			break
		}
	case reflect.Map:
		result[parentTagName] = value.Interface()
		break
	case reflect.Array:
		result[parentTagName] = value.Interface()
		break
	default:
		result[parentTagName] = value.Interface()
		break
	}
	return result
}
