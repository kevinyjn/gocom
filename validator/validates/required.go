package validates

import (
	"fmt"
	"reflect"
)

// ValidateRequired validator
func ValidateRequired(v reflect.Value, label string) error {
	switch v.Type().Kind() {
	case reflect.String:
		if v.String() != "" {
			return nil
		}
		break
	case reflect.Array, reflect.Map, reflect.Struct, reflect.Chan, reflect.Func:
		if !v.IsNil() {
			return nil
		}
		break
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int8:
		if v.Int() != 0 {
			return nil
		}
		break
	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8:
		if v.Uint() != 0 {
			return nil
		}
		break
	case reflect.Float32, reflect.Float64:
		if v.Float() != 0 {
			return nil
		}
		break
	default:
		break
	}
	return fmt.Errorf("%s不能为空", label)
}
