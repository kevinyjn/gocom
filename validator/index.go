package validator

import (
	"errors"
	"reflect"
	"strings"

	"github.com/kevinyjn/gocom/validator/validates"
)

// ValidateElement validate information
type ValidateElement struct {
	Type    string
	Content string
}

// Validate types
const (
	ValidateTypeRequired = "required"
	ValidateTypeRegex    = "regex"
	ValidateTypeObjectID = "objectId"
)

// Validate validator
func Validate(v interface{}) error {
	msgs := []string{}
	value := reflect.ValueOf(v)
	t := reflect.TypeOf(v)
	if value.IsValid() && value.Type().Kind() == reflect.Ptr {
		value = value.Elem()
		t = t.Elem()
	}
	if !value.IsValid() || value.Type().Kind() != reflect.Struct {
		return nil
	}

	var err error
	for i := 0; i < value.NumField(); i++ {
		f := value.Field(i)
		ft := t.Field(i)
		labelInfo := ft.Tag.Get("label")
		if "" == labelInfo {
			labelInfo = ft.Tag.Get("comment")
		}
		if "" == labelInfo {
			labelInfo = ft.Name
		}
		defaultInfo := ft.Tag.Get("default")
		if "" != defaultInfo && (ft.Name[0] >= 'A' && ft.Name[0] <= 'Z') {
			err = validates.ValidateDefault(f, defaultInfo, labelInfo)
			if nil != err {
				msgs = append(msgs, err.Error())
			}
		}
		validateInfo := ft.Tag.Get("validate")
		if "" == validateInfo {
			continue
		}

		err = ValidateFieldValue(f, validateInfo, labelInfo)
		if nil != err {
			msgs = append(msgs, err.Error())
		}

		if (reflect.Struct == f.Type().Kind() || reflect.Ptr == f.Type().Kind()) && f.CanInterface() {
			err = Validate(f.Interface())
			if nil != err {
				msgs = append(msgs, err.Error())
			}
		} else if reflect.Slice == f.Type().Kind() {
			l := f.Len()
			for i := 0; i < l; i++ {
				f2 := f.Index(i)
				if (reflect.Struct == f2.Type().Kind() || reflect.Ptr == f2.Type().Kind()) && f2.CanInterface() {
					err = Validate(f2.Interface())
					if nil != err {
						msgs = append(msgs, err.Error())
					}
				} else {
					break
				}
			}
		}
	}

	if len(msgs) == 0 {
		return nil
	}
	return errors.New(strings.Join(msgs, ";"))
}

// ValidateFieldValue validator
func ValidateFieldValue(f reflect.Value, validateInfo string, label string) error {
	validateElements := AnalyzeValidateElements(validateInfo)
	msgs := []string{}
	var err error
	for _, ele := range validateElements {
		err = nil
		switch ele.Type {
		case ValidateTypeRequired:
			err = validates.ValidateRequired(f, label)
			break
		case ValidateTypeRegex:
			err = validates.ValidateRegex(f, ele.Content, label)
			break
		case ValidateTypeObjectID:
			err = validates.ValidateObjectID(f, label)
			break
		}
		if err != nil {
			msgs = append(msgs, err.Error())
		}
	}
	if len(msgs) > 0 {
		return errors.New(strings.Join(msgs, ";"))
	}
	return nil
}

// AnalyzeValidateElements validator
func AnalyzeValidateElements(validateInfo string) []ValidateElement {
	validateSlices := splitValidateContent(validateInfo, ',', 0)
	elements := []ValidateElement{}
	for _, validateElement := range validateSlices {
		slices := splitValidateContent(validateElement, ':', 1)
		validateType := strings.TrimSpace(slices[0])
		ele := ValidateElement{
			Type: validateType,
		}
		switch validateType {
		case "regex":
			ele.Content = slices[1]
			break
		}
		elements = append(elements, ele)
	}
	return elements
}

func splitValidateContent(s string, sep byte, count int) []string {
	slices := []string{}
	tmp := []byte{}
	bs := []byte(s)
	bsl := len(bs)
	escape := false
	splitCount := 0
	for i := 0; i < bsl; i++ {
		c := bs[i]
		switch c {
		case '\\':
			if escape {
				tmp = append(tmp, c)
				escape = false
			} else {
				escape = true
			}
			break
		case sep:
			if escape {
				tmp = append(tmp, c)
				escape = false
			} else {
				slices = append(slices, string(tmp))
				splitCount++
				tmp = []byte{}
				if count > 0 {
					if splitCount >= count {
						for i++; i < bsl; i++ {
							tmp = append(tmp, bs[i])
						}
					}
				}
			}
			break
		default:
			tmp = append(tmp, c)
			escape = false
			break
		}
	}
	slices = append(slices, string(tmp))

	return slices
}
