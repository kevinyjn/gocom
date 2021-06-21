package utils

import (
	"reflect"
	"strings"
)

// FormDataCopyFields copy form data fields into db model fields
func FormDataCopyFields(formData interface{}, dest interface{}, tagName string) int {
	valueFormData := reflect.ValueOf(formData)
	valueDest := reflect.ValueOf(dest)
	if valueFormData.Type().Kind() == reflect.Ptr {
		valueFormData = valueFormData.Elem()
	}
	if valueDest.Type().Kind() == reflect.Ptr {
		valueDest = valueDest.Elem()
	}
	fields := valueFormData.NumField()
	fieldValues := map[string]reflect.Value{}
	for i := 0; i < fields; i++ {
		fieldType := valueFormData.Type().Field(i)
		if "ID" == fieldType.Name {
			continue
		}
		fieldTagName := strings.SplitN(fieldType.Tag.Get(tagName), ",", 2)[0]
		if "" != fieldTagName {
			fieldValues[fieldTagName] = valueFormData.Field(i)
		} else if IsCapital(fieldType.Name) {
			fieldValues[fieldType.Name] = valueFormData.Field(i)
		}
	}
	return copyFields(fieldValues, valueDest, tagName)
}

func copyFields(fieldValues map[string]reflect.Value, dst reflect.Value, tagName string) int {
	affected := 0
	fields := dst.NumField()
	for i := 0; i < fields; i++ {
		fieldType := dst.Type().Field(i)
		fieldTagName := strings.SplitN(fieldType.Tag.Get(tagName), ",", 2)[0]
		if "" == fieldTagName {
			if IsCapital(fieldType.Name) {
				fieldTagName = fieldType.Name
			} else {
				continue
			}
		}
		srcValue, exists := fieldValues[fieldTagName]
		if !exists {
			continue
		}
		fieldValue := dst.Field(i)
		switch fieldValue.Kind() {
		case reflect.String:
			if fieldValue.String() != srcValue.String() {
				affected++
				fieldValue.Set(srcValue)
			}
			break
		case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int8:
			if fieldValue.Int() != srcValue.Int() && srcValue.Int() != 0 {
				affected++
				fieldValue.Set(srcValue)
			}
			break
		case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8:
			if fieldValue.Uint() != srcValue.Uint() {
				affected++
				fieldValue.Set(srcValue)
			}
			break
		case reflect.Float32, reflect.Float64:
			if fieldValue.Float() != srcValue.Float() {
				affected++
				fieldValue.Set(srcValue)
			}
			break
		case reflect.Bool:
			if fieldValue.Bool() != srcValue.Bool() {
				affected++
				fieldValue.Set(srcValue)
			}
			break
		default:
			break
		}
	}
	return affected
}
