package schema

import (
	"errors"
)

// Object validates objects which are defined by Schemas.
type Object struct {
	Schema *Schema
}

// Compile implements the ReferenceCompiler interface.
func (v *Object) Compile(rc ReferenceChecker) error {
	if v.Schema == nil {
		return errors.New("no schema defined")
	}
	if err := compileDependencies(*v.Schema, v.Schema); err != nil {
		return err
	}
	return v.Schema.Compile(rc)
}

// Validate implements FieldValidator interface.
func (v Object) Validate(value interface{}) (interface{}, error) {
	obj, ok := value.(map[string]interface{})
	if !ok {
		return nil, errors.New("not an object")
	}
	dest, errs := v.Schema.Validate(nil, obj)
	if len(errs) > 0 {
		// Currently, tests expect FieldValidators to always return a nil value
		// on validation errors.
		return nil, ErrorMap(errs)
	}
	return dest, nil
}

// GetField implements the FieldGetter interface.
func (v Object) GetField(name string) *Field {
	return v.Schema.GetField(name)
}

func (v Object) Serialize(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	obj, ok := value.(map[string]interface{})
	if !ok {
		return nil, errors.New("not an object")
	}

	for name, val := range obj {
		field := v.Schema.GetField(name)
		if field == nil {
			continue
		}
		s, ok := field.Validator.(FieldSerializer)
		if ok {
			var err error
			obj[name], err = s.Serialize(val)
			if err != nil {
				return nil, err
			}
		}
	}
	return obj, nil
}
