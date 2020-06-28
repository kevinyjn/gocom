package validates

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
)

// ValidateRegex validator
func ValidateRegex(v reflect.Value, regtext string, label string) error {
	if v.Type().Kind() != reflect.String {
		return nil
	}

	reg, err := regexp.Compile(regtext)
	if err != nil {
		return err
	}
	if reg.MatchString(v.String()) {
		return nil
	}
	return errors.New(fmt.Sprintf("%s内容不合规范", label))
}
