package schema_test

import (
	"testing"

	"github.com/rs/rest-layer/schema"
	"github.com/stretchr/testify/assert"
)

func TestObjectCompile(t *testing.T) {
	cases := []referenceCompilerTestCase{
		{
			Name:             "{}",
			Compiler:         &schema.Object{},
			ReferenceChecker: fakeReferenceChecker{},
			Error:            "no schema defined",
		},
		{
			Name:             "{Schema:{}}",
			Compiler:         &schema.Object{Schema: &schema.Schema{}},
			ReferenceChecker: fakeReferenceChecker{},
		},
		{
			Name: `{Schema:{"foo":String}}`,
			Compiler: &schema.Object{Schema: &schema.Schema{Fields: schema.Fields{
				"foo": {Validator: &schema.String{}},
			}}},
			ReferenceChecker: fakeReferenceChecker{},
		},
		{
			Name: `{Schema:{"foo":Reference{Path:valid}}}`,
			Compiler: &schema.Object{Schema: &schema.Schema{Fields: schema.Fields{
				"foo": {Validator: &schema.Reference{Path: "bar"}},
			}}},
			ReferenceChecker: fakeReferenceChecker{"bar": {SchemaValidator: &schema.Schema{}}},
		},
		{
			Name: `{Schema:{"foo":Reference{Path:invalid}}}`,
			Compiler: &schema.Object{Schema: &schema.Schema{Fields: schema.Fields{
				"foo": {Validator: &schema.Reference{Path: "foobar"}},
			}}},
			ReferenceChecker: fakeReferenceChecker{"bar": {SchemaValidator: &schema.Schema{}}},
			Error:            "foo: can't find resource 'foobar'",
		},
	}
	for i := range cases {
		cases[i].Run(t)
	}
}

func TestObjectValidate(t *testing.T) {
	cases := []fieldValidatorTestCase{
		{
			Name: `{Schema:{"foo":String}}.Validate(valid)`,
			Validator: &schema.Object{Schema: &schema.Schema{Fields: schema.Fields{
				"foo": {Validator: &schema.String{}},
			}}},
			Input:  map[string]interface{}{"foo": "hello"},
			Expect: map[string]interface{}{"foo": "hello"},
		},
		{
			Name: `{Schema:{"foo":String}}.Validate(invalid)`,
			Validator: &schema.Object{Schema: &schema.Schema{Fields: schema.Fields{
				"foo": {Validator: &schema.String{}},
			}}},
			Input: map[string]interface{}{"foo": 1},
			Error: "foo is [not a string]",
		},
		{
			Name: `{Schema:{"test":String,"count:Integer"}}.Validate(doubleError)`,
			Validator: &schema.Object{Schema: &schema.Schema{Fields: schema.Fields{
				"test":  {Validator: &schema.String{}},
				"count": {Validator: &schema.Integer{}},
			}}},
			Input: map[string]interface{}{"test": 1, "count": "hello"},
			Error: "count is [not an integer], test is [not a string]",
		},
		{
			Name: `{Schema:{"foo":Reference{Path:valid}}}.Validate(valid)`,
			Validator: &schema.Object{Schema: &schema.Schema{Fields: schema.Fields{
				"foo": {Validator: &schema.Reference{Path: "bar"}},
			}}},
			ReferenceChecker: fakeReferenceChecker{
				"bar": {IDs: []interface{}{"a", "b"}, Validator: &schema.String{}, SchemaValidator: &schema.Schema{}},
			},
			Input:  map[string]interface{}{"foo": "a"},
			Expect: map[string]interface{}{"foo": "a"},
		},
		{
			Name: `{Schema:{"foo":Reference{Path:valid}}}.Validate(invalid)`,
			Validator: &schema.Object{Schema: &schema.Schema{Fields: schema.Fields{
				"foo": {Validator: &schema.Reference{Path: "bar"}},
			}}},
			ReferenceChecker: fakeReferenceChecker{
				"bar": {IDs: []interface{}{"a", "b"}, Validator: &schema.String{}, SchemaValidator: &schema.Schema{}},
			},
			Input: map[string]interface{}{"foo": "c"},
			Error: "foo is [not found]",
		},
	}
	for i := range cases {
		cases[i].Run(t)
	}
}

func TestObjectValidatorErrorType(t *testing.T) {
	obj := map[string]interface{}{"foo": 1}
	v := &schema.Object{Schema: &schema.Schema{Fields: schema.Fields{
		"foo": {Validator: &schema.String{}},
	}}}
	_, err := v.Validate(obj)
	assert.IsType(t, schema.ErrorMap{}, err, "Unexpected error type")
}

func TestObjectSerialize(t *testing.T) {
	cases := []fieldSerializerTestCase{
		{
			Name: "null",
			Serializer: schema.Object{Schema: &schema.Schema{
				Fields: schema.Fields{
					"field1": {Validator: &schema.String{}},
					"field2": {Validator: &schema.IP{StoreBinary: true}},
				},
			}},
			Input:  nil,
			Expect: nil,
			Error:  "",
		},
		{
			Name: "empty",
			Serializer: schema.Object{Schema: &schema.Schema{
				Fields: schema.Fields{
					"field1": {Validator: &schema.String{}},
					"field2": {Validator: &schema.IP{StoreBinary: true}},
				},
			}},
			Input:  map[string]interface{}{},
			Expect: map[string]interface{}{},
			Error:  "",
		},
		{
			Name: "field with Serializer",
			Serializer: schema.Object{Schema: &schema.Schema{
				Fields: schema.Fields{
					"field1": {Validator: &schema.String{}},
					"field2": {Validator: &schema.IP{StoreBinary: true}},
				},
			}},
			Input: map[string]interface{}{
				"field1": "foo",
				"field2": []byte{1, 2, 3, 4},
			},
			Expect: map[string]interface{}{
				"field1": "foo",
				"field2": "1.2.3.4",
			},
			Error: "",
		},
		{
			Name: "field with Serializer error",
			Serializer: schema.Object{Schema: &schema.Schema{
				Fields: schema.Fields{
					"field1": {Validator: &schema.String{}},
					"field2": {Validator: &schema.IP{StoreBinary: true}},
				},
			}},
			Input: map[string]interface{}{
				"field1": "foo",
				"field2": 11,
			},
			Expect: nil,
			Error:  "invalid type",
		},
	}
	for i := range cases {
		cases[i].Run(t)
	}
}
