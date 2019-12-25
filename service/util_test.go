package service

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

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
		},
		{
			name:     "nil data",
			input:    nil,
			expected: *new(map[string]interface{}), // its an empty map[string]interface{} with value as nil (but type is already defined)
		},
	}
	for _, tc := range tests {
		actual, err := GetMap(tc.input)
		assert.NoError(t, tc.err, err, "test %s failed", tc.name)
		assert.Equal(t, tc.expected, actual, "test %s failed", tc.name)
	}
}
