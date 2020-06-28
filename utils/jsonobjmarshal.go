package utils

import (
	"fmt"
	"reflect"

	// "github.com/kevinyjn/gocom/logger"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// JSONObjUnmarshal unmarshal object
func JSONObjUnmarshal(in *map[string]interface{}, out interface{}) error {
	rval := reflect.ValueOf(out)
	if rval.Kind() != reflect.Ptr {
		return fmt.Errorf("argument to Decode must be a pointer to a type, but got %v", rval)
	}
	st := rval.Type().Elem()
	rval = rval.Elem()
	for i := 0; i < st.NumField(); i++ {
		sf := st.Field(i)
		fieldName := sf.Tag.Get("bson")
		if fieldName == "" {
			fieldName = sf.Tag.Get("json")
		} else if (*in)[fieldName] == nil {
			fieldName = sf.Tag.Get("json")
		}
		if fieldName == "" {
			continue
		}
		fieldValue := (*in)[fieldName]
		if fieldValue == nil {
			continue
		}

		fts := sf.Type.String()
		field := rval.FieldByName(sf.Name)
		switch fts {
		case "primitive.ObjectID":
			oval, err := primitive.ObjectIDFromHex(ToString(fieldValue))
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(oval))
		case "string":
			field.SetString(ToString(fieldValue))
		case "int", "int64":
			field.SetInt(ToInt64(fieldValue))
		case "bool":
			field.SetBool(ToBoolean(fieldValue))
		case "float", "float64":
			field.SetFloat(ToDouble(fieldValue))
		case "uint64":
			field.SetUint(ToUint64(fieldValue))
		case "map[string]string":
			if reflect.ValueOf(fieldValue).Type().String() == "map[string]interface {}" {
				mval := map[string]string{}
				for fk, fv := range fieldValue.(map[string]interface{}) {
					mval[fk] = ToString(fv)
				}
				field.Set(reflect.ValueOf(mval))
			}
		// case "map[string]interface {}":
		// 	if reflect.ValueOf(fieldValue).Type().String() == "map[string]interface {}" {
		// 		mval := map[string]interface{}{}
		// 		for fk, fv := range fieldValue.(map[string]interface{}) {
		// 			mval[fk] = fv
		// 		}
		// 		field.Set(reflect.ValueOf(mval))
		// 	}
		default:
			field.Set(reflect.ValueOf(fieldValue))
			// logger.Info.Printf("Set field type:%s for field:%s", fts, sf.Name)
		}
	}
	return nil
}
