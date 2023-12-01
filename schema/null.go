package schema

import (
	"errors"
	"reflect"
)

// Null validates that the value is null.
type Null []FieldValidator

// Validate ensures that value is null.
func (v Null) Validate(value interface{}) (interface{}, error) {
	if value == nil || (reflect.TypeOf(value).Kind() == reflect.Ptr && reflect.ValueOf(value).IsNil()) {
		return value, nil
	}

	return value, errors.New("not null")
}
