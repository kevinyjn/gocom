package validates

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/kevinyjn/gocom/utils"
)

// ValidateDefault validator
func ValidateDefault(v reflect.Value, defaultText string, label string) error {
	if false == v.IsValid() {
		return nil
	}
	switch v.Type().Kind() {
	case reflect.String:
		if v.String() == "" {
			v.SetString(defaultText)
		}
		break
	case reflect.Slice:
		if len(v.Bytes()) <= 0 {
			v.SetBytes([]byte(defaultText))
		}
		break
	case reflect.Array, reflect.Map:
		if v.IsNil() || v.Len() <= 0 {
			return fmt.Errorf("%s: array 或 map 类型目前尚不支持默认值", label)
		}
		break
	case reflect.Struct:
		if v.IsNil() {
			return fmt.Errorf("%s: struct 类型目前尚不支持默认值", label)
		}
		break
	case reflect.Chan, reflect.Func:
		if v.IsNil() {
			return fmt.Errorf("%s: chan 或函数类型目前尚不支持默认值", label)
		}
		break
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int8:
		if v.Int() == 0 {
			iv, err := strconv.ParseInt(defaultText, 10, 64)
			if nil != err {
				return fmt.Errorf("%s: 转换默认值:%s到整形失败:%v", label, defaultText, err)
			}
			v.SetInt(iv)
		}
		break
	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8:
		if v.Uint() == 0 {
			iv, err := strconv.ParseUint(defaultText, 10, 64)
			if nil != err {
				return fmt.Errorf("%s: 转换默认值:%s到无符号整形失败:%v", label, defaultText, err)
			}
			v.SetUint(iv)
		}
		break
	case reflect.Float32, reflect.Float64:
		if v.Float() == 0 {
			fv, err := strconv.ParseFloat(defaultText, 64)
			if nil != err {
				return fmt.Errorf("%s: 转换默认值:%s到浮点型失败:%v", label, defaultText, err)
			}
			v.SetFloat(fv)
		}
		break
	case reflect.Ptr:
		if false == v.IsNil() {
			return fmt.Errorf("%s: 指针类型目前尚不支持默认值", label)
		}
	case reflect.Bool:
		if v.Bool() == false {
			v.SetBool(utils.ToBoolean(defaultText))
		}
		break
	default:
		break
	}
	return nil
}
