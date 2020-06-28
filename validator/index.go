package validator

import (
	"errors"
	"reflect"
	"strings"

	"github.com/kevinyjn/gocom/validator/validates"
)

// Validate validator
func Validate(v interface{}) error {
	msgs := []string{}
	value := reflect.ValueOf(v)
	t := reflect.TypeOf(v)
	if value.Type().Kind() == reflect.Ptr {
		value = value.Elem()
		t = t.Elem()
	}
	if value.Type().Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < value.NumField(); i++ {
		f := value.Field(i)
		ft := t.Field(i)
		validateInfo := ft.Tag.Get("validate")
		if validateInfo == "" {
			continue
		}
		commentInfo := ft.Tag.Get("comment")
		if commentInfo == "" {
			commentInfo = ft.Name
		}

		err := ValidateFieldValue(f, validateInfo, commentInfo)
		if err != nil {
			msgs = append(msgs, err.Error())
		}
	}

	if len(msgs) == 0 {
		return nil
	}
	return errors.New(strings.Join(msgs, ";"))
}

// ValidateFieldValue validator
func ValidateFieldValue(f reflect.Value, validateInfo string, label string) error {
	validateSlices := splitValidateContent(validateInfo, ',', 0)
	msgs := []string{}
	var err error
	for _, validateElement := range validateSlices {
		slices := splitValidateContent(validateElement, ':', 1)
		validateType := strings.TrimSpace(slices[0])
		err = nil
		switch validateType {
		case "required":
			err = validates.ValidateRequired(f, label)
			break
		case "regex":
			regtext := slices[1]
			err = validates.ValidateRegex(f, regtext, label)
			break
		case "objectId":
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
