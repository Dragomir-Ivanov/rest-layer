//go:build go1.7
// +build go1.7

package schema_test

import (
	"testing"

	"github.com/rs/rest-layer/schema"
)

func TestArrayValidatorCompile(t *testing.T) {
	testCases := []referenceCompilerTestCase{
		{
			Name:             "Values.Validator=&String{}",
			Compiler:         &schema.Array{Values: schema.Field{Validator: &schema.String{}}},
			ReferenceChecker: fakeReferenceChecker{},
		},
		{
			Name:             "Values.Validator=&String{Regexp:invalid}",
			Compiler:         &schema.Array{Values: schema.Field{Validator: &schema.String{Regexp: "[invalid re"}}},
			ReferenceChecker: fakeReferenceChecker{},
			Error:            ": invalid regexp: error parsing regexp: missing closing ]: `[invalid re`",
		},
		{
			Name:             "Values.Validator=String{}",
			Compiler:         &schema.Array{Values: schema.Field{Validator: schema.String{}}},
			ReferenceChecker: fakeReferenceChecker{},
			Error:            ": not a schema.Validator pointer",
		},
	}
	for i := range testCases {
		testCases[i].Run(t)
	}
}

func TestArrayValidator(t *testing.T) {
	testCases := []fieldValidatorTestCase{
		{
			Name:      `Values.Validator=nil,Validate([]interface{}{true,"value"})`,
			Validator: &schema.Array{},
			Input:     []interface{}{true, "value"},
			Expect:    []interface{}{true, "value"},
		},
		{
			Name:      `Values.Validator=&schema.Bool{},Validate([]interface{}{true,false})`,
			Validator: &schema.Array{Values: schema.Field{Validator: &schema.Bool{}}},
			Input:     []interface{}{true, false},
			Expect:    []interface{}{true, false},
		},
		{
			Name:      `Values.Validator=&schema.Bool{},Validate([]interface{}{})`,
			Validator: &schema.Array{Values: schema.Field{Validator: &schema.Bool{}}},
			Input:     []interface{}{},
			Expect:    []interface{}{},
		},
		{
			Name:      `Values.Validator=&schema.Bool{},Validate([]interface{}{true,"value"})`,
			Validator: &schema.Array{Values: schema.Field{Validator: &schema.Bool{}}},
			Input:     []interface{}{true, "value"},
			Error:     "invalid value at #2: not a Boolean",
		},
		{
			Name:      `Values.Validator=&String{},Validate("value")`,
			Validator: &schema.Array{Values: schema.Field{Validator: &schema.String{}}},
			Input:     "value",
			Error:     "not an array",
		},
		{
			Name:      `MinLen=2,Validate([]interface{}{true,false})`,
			Validator: &schema.Array{Values: schema.Field{Validator: &schema.Bool{}}, MinLen: 2},
			Input:     []interface{}{true, false},
			Expect:    []interface{}{true, false},
		},
		{
			Name:      `MinLen=3,Validate([]interface{}{true,false})`,
			Validator: &schema.Array{Values: schema.Field{Validator: &schema.Bool{}}, MinLen: 3},
			Input:     []interface{}{true, false},
			Error:     "has fewer items than 3",
		},
		{
			Name:      `MaxLen=2,Validate([]interface{}{true,false})`,
			Validator: &schema.Array{Values: schema.Field{Validator: &schema.Bool{}}, MaxLen: 2},
			Input:     []interface{}{true, false},
			Expect:    []interface{}{true, false},
		},
		{
			Name:      `MaxLen=1,Validate([]interface{}{true,false})`,
			Validator: &schema.Array{Values: schema.Field{Validator: &schema.Bool{}}, MaxLen: 1},
			Input:     []interface{}{true, false},
			Error:     "has more items than 1",
		},
	}
	for i := range testCases {
		testCases[i].Run(t)
	}
}

func TestArrayQueryValidator(t *testing.T) {
	testCases := []fieldQueryValidatorTestCase{
		{
			Name:      `Values.Validator=nil,ValidateQuery([]interface{}{true,"value"})`,
			Validator: &schema.Array{},
			Input:     []interface{}{true, "value"},
			Expect:    []interface{}{true, "value"},
		},
		{
			Name:      `Values.Validator=&schema.Bool{},ValidateQuery([]interface{}{true,false})`,
			Validator: &schema.Array{Values: schema.Field{Validator: &schema.Bool{}}},
			Input:     []interface{}{true, false},
			Expect:    []interface{}{true, false},
		},
		{
			Name:      `Values.Validator=&schema.Bool{},ValidateQuery([]interface{}{})`,
			Validator: &schema.Array{Values: schema.Field{Validator: &schema.Bool{}}},
			Input:     []interface{}{},
			Expect:    []interface{}{},
		},
		{
			Name:      `Values.Validator=&schema.Bool{},ValidateQuery([]interface{}{true,"value"})`,
			Validator: &schema.Array{Values: schema.Field{Validator: &schema.Bool{}}},
			Input:     []interface{}{true, "value"},
			Error:     "invalid value at #2: not a Boolean",
		},
		{
			Name:      `Values.Validator=&String{},ValidateQuery("value")`,
			Validator: &schema.Array{Values: schema.Field{Validator: &schema.String{}}},
			Input:     "value",
			Expect:    "value",
		},
		{
			Name:      `MinLen=2,ValidateQuery([]interface{}{true,false})`,
			Validator: &schema.Array{Values: schema.Field{Validator: &schema.Bool{}}, MinLen: 2},
			Input:     []interface{}{true, false},
			Expect:    []interface{}{true, false},
		},
		{
			Name:      `MinLen=3,ValidateQuery([]interface{}{true,false})`,
			Validator: &schema.Array{Values: schema.Field{Validator: &schema.Bool{}}, MinLen: 3},
			Input:     []interface{}{true, false},
			Expect:    []interface{}{true, false},
		},
		{
			Name:      `MaxLen=2,ValidateQuery([]interface{}{true,false})`,
			Validator: &schema.Array{Values: schema.Field{Validator: &schema.Bool{}}, MaxLen: 2},
			Input:     []interface{}{true, false},
			Expect:    []interface{}{true, false},
		},
		{
			Name:      `MaxLen=1,ValidateQuery([]interface{}{true,false})`,
			Validator: &schema.Array{Values: schema.Field{Validator: &schema.Bool{}}, MaxLen: 1},
			Input:     []interface{}{true, false},
			Expect:    []interface{}{true, false},
		},
	}
	for i := range testCases {
		testCases[i].Run(t)
	}
}

func TestArraySerialize(t *testing.T) {
	cases := []fieldSerializerTestCase{
		{
			Name:       "null",
			Serializer: schema.Array{Values: schema.Field{Validator: &schema.IP{StoreBinary: true}}},
			Input:      nil,
			Expect:     nil,
			Error:      "",
		},
		{
			Name:       "empty",
			Serializer: schema.Array{Values: schema.Field{Validator: &schema.IP{StoreBinary: true}}},
			Input:      []interface{}{},
			Expect:     []interface{}{},
			Error:      "",
		},
		{
			Name:       "field without Serializer",
			Serializer: schema.Array{Values: schema.Field{Validator: &schema.String{}}},
			Input: []interface{}{
				"foo",
				"bar",
			},
			Expect: []interface{}{
				"foo",
				"bar",
			},
			Error: "",
		},
		{
			Name:       "field with Serializer",
			Serializer: schema.Array{Values: schema.Field{Validator: &schema.IP{StoreBinary: true}}},
			Input: []interface{}{
				[]byte{1, 2, 3, 4},
				[]byte{5, 6, 7, 8},
			},
			Expect: []interface{}{
				"1.2.3.4",
				"5.6.7.8",
			},
			Error: "",
		},
		{
			Name:       "field with Serializer error",
			Serializer: schema.Array{Values: schema.Field{Validator: &schema.IP{StoreBinary: true}}},
			Input: []interface{}{
				11,
				[]byte{5, 6, 7, 8},
			},
			Expect: nil,
			Error:  "invalid type",
		},
	}
	for i := range cases {
		cases[i].Run(t)
	}
}
