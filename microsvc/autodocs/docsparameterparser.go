package autodocs

import (
	"reflect"
	"strings"

	"github.com/kevinyjn/gocom/validator"
)

// ParseParameters as list of ParameterInfo
func ParseParameters(valueType reflect.Type, fieldTagName string) []ParameterInfo {
	if valueType.Kind() == reflect.Ptr {
		valueType = valueType.Elem()
	}
	params := []ParameterInfo{}
	numField := valueType.NumField()
	for i := 0; i < numField; i++ {
		ft := valueType.Field(i)
		tagValue := ft.Tag.Get(fieldTagName)
		if "" == tagValue {
			continue
		}
		tagValues := strings.Split(tagValue, ",")
		isArray := ft.Type.Kind() == reflect.Array
		fieldName := ""
		fieldLabel := ""
		fieldDescription := ""
		isRequired := false
		if "protobuf" == fieldTagName {
			for _, e := range tagValues {
				if "rep" == e {
					isArray = true
				}
				if "req" == e {
					isRequired = true
				}
				if strings.HasPrefix(e, "name=") {
					fieldName = e[5:]
				}
			}
		} else {
			fieldName = tagValues[0]
			if "-" == fieldName || "" == fieldName {
				continue
			}
		}
		if "" == fieldName {
			fieldName = ft.Name
		}
		fieldDescription = ft.Tag.Get("description")
		if "" == fieldDescription {
			fieldDescription = ft.Tag.Get("comment")
		}
		fieldLabel = ft.Tag.Get("label")
		if "" == fieldLabel {
			fieldLabel = fieldDescription
		}
		if "" == fieldDescription {
			fieldDescription = fieldLabel
		}

		validateLabel := ft.Tag.Get("validate")
		if "" != validateLabel {
			validateElements := validator.AnalyzeValidateElements(validateLabel)
			for _, ele := range validateElements {
				switch ele.Type {
				case validator.ValidateTypeRequired:
					isRequired = true
					break
				}
			}
		}

		fieldType, fieldFormat := determineFieldType(ft.Type, isArray)
		pi := ParameterInfo{
			Type:        fieldType,
			Name:        fieldName,
			Description: fieldDescription,
			In:          "body",
			Required:    isRequired,
			Format:      fieldFormat,
		}

		params = append(params, pi)
	}
	return params
}

// ParseParameters as list of ParameterInfo
func ParseResponseParameters(valueType reflect.Type, fieldTagName string) map[string]PropertyInfo {
	properties := map[string]PropertyInfo{}
	params := ParseParameters(valueType, fieldTagName)
	for _, param := range params {
		properties[param.Name] = PropertyInfo{
			Type:        param.Type,
			Format:      param.Format,
			Description: param.Description,
		}
	}
	return properties
}

func determineFieldType(vt reflect.Type, isArray bool) (string, string) {
	if isArray {
		return "array", ""
	}
	if vt.Kind() == reflect.Ptr {
		vt = vt.Elem()
	}
	ft := "string"
	format := ""
	switch vt.Kind() {
	case reflect.Array:
		ft = "array"
		break
	case reflect.Int:
		ft = "integer"
		format = "int"
		break
	case reflect.Int8:
		ft = "integer"
		format = "int8"
		break
	case reflect.Int16:
		ft = "integer"
		format = "int16"
		break
	case reflect.Int32:
		ft = "integer"
		format = "int32"
		break
	case reflect.Int64:
		ft = "integer"
		format = "int64"
		break
	case reflect.Uint:
		ft = "integer"
		format = "uint"
		break
	case reflect.Uint8:
		ft = "integer"
		format = "uint8"
		break
	case reflect.Uint16:
		ft = "integer"
		format = "uint16"
		break
	case reflect.Uint32:
		ft = "integer"
		format = "uint32"
		break
	case reflect.Uint64:
		ft = "integer"
		format = "uint64"
		break
	case reflect.Bool:
		ft = "boolean"
		break
	case reflect.String:
		ft = "string"
		break
	case reflect.Struct:
		ft = "object"
		break
	case reflect.Float32:
		ft = "float"
		break
	case reflect.Float64:
		ft = "double"
		break
	case reflect.Slice:
		ft = "bytes"
		break
	}
	return ft, format
}
