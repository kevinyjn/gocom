package validates

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
)

// ValidateObjectID validator
func ValidateObjectID(v reflect.Value, label string) error {
	if v.Type().Kind() != reflect.String {
		return nil
	}

	regtext := "^(?:[a-f0-9]{24})?$"
	reg, err := regexp.Compile(regtext)
	if err != nil {
		return err
	}
	if reg.MatchString(v.String()) {
		return nil
	}
	return errors.New(fmt.Sprintf("%s内容不合规范", label))
}
