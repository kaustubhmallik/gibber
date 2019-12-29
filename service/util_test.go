package service

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"reflect"
	"strings"
	"testing"
)

func TestProjectRootPath(t *testing.T) {
	path := ProjectRootPath()
	assert.True(t, len(path) > 0, "project root path should be non-empty")
	assert.True(t, strings.HasSuffix(path, "gibber/"), "project root path should end with project name")
}

func TestGetMap(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{} // expected result
		err      error       // expected error, if any
	}{
		{
			name: "single field struct",
			input: struct {
				Foo string `json:"foo"`
			}{
				Foo: "bar",
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			err: nil,
		},
		{
			name: "double field struct",
			input: struct {
				Foo  string `json:"foo"`
				Foo2 string `json:"foo2"`
			}{
				Foo:  "bar",
				Foo2: "bar2",
			},
			expected: map[string]interface{}{
				"foo":  "bar",
				"foo2": "bar2",
			},
			err: nil,
		},
		{
			name: "different types of field struct",
			input: struct {
				Foo  string  `json:"foo"`
				Foo2 float64 `json:"foo2"`
				Foo3 int     `json:"foo3"`
				Foo4 bool    `json:"foo4"`
			}{
				Foo:  "bar",
				Foo2: 2.12,
				Foo3: 123,
				Foo4: true,
			},
			expected: map[string]interface{}{
				"foo":  "bar",
				"foo2": 2.12,
				"foo3": float64(123),
				"foo4": true,
			},
			err: nil,
		},
		{
			name:     "nil data",
			input:    nil,
			expected: *new(map[string]interface{}), // its an empty map[string]interface{} with value as nil (but type is already defined)
			err:      nil,
		},
		{
			name:     "nil data",
			input:    func() {},
			expected: *new(map[string]interface{}), // its an empty map[string]interface{} with value as nil (but type is already defined)
			err:      new(json.UnsupportedTypeError),
		},
	}
	for _, tc := range tests {
		actual, err := GetMap(tc.input)
		assert.Equal(t, reflect.TypeOf(tc.err), reflect.TypeOf(err), "test %s failed", tc.name)
		assert.Equal(t, tc.expected, actual, "test %s failed", tc.name)
	}
}

func TestRandomString(t *testing.T) {
	tests := []struct {
		name  string
		input int
	}{
		{
			name:  "empty random string",
			input: 0,
		},
		{
			name:  "single rune random string",
			input: 1,
		},
		{
			name:  "normal random string",
			input: 10,
		},
		{
			name:  "long random string",
			input: 25,
		},
		{
			name:  "very long random string",
			input: 100,
		},
	}
	for _, tc := range tests {
		o1, o2, o3 := RandomString(tc.input), RandomString(tc.input), RandomString(tc.input)
		assert.Equal(t, tc.input, len(o1), "string length is not as expected")
		assert.Equal(t, tc.input, len(o2), "string length is not as expected")
		if tc.input > 1 {
			assert.NotEqual(t, o1, o2, "random string are not equal on successive calls")
			assert.NotEqual(t, o1, o3, "random string are not equal on successive calls")
			assert.NotEqual(t, o2, o3, "random string are not equal on successive calls")
		}
	}
}
