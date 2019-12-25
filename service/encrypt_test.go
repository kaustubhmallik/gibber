package service

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGenerateHash(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "small string",
			input: "password",
		},
		{
			name:  "long string",
			input: "a long string to be hashed in a go test",
		},
	}
	for _, tc := range tests {
		hash, err := GenerateHash(tc.input)
		assert.NoError(t, err, "%s failed")
		assert.True(t, len(hash) > 0, "%s failed")
	}
}

func TestMatchHashAndPlainText(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "small string",
			input: "password",
		},
		{
			name:  "long string",
			input: "a long string to be hashed in a go test",
		},
	}
	for _, tc := range tests {
		hash, err := GenerateHash(tc.input)
		assert.NoError(t, err, "%s failed")
		assert.True(t, len(hash) > 0, "%s failed")
		assert.NoError(t, MatchHashAndPlainText(hash, tc.input), "%s failed")
	}
}
