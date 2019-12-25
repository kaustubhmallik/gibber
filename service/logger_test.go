package service

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLogger(t *testing.T) {
	lg := Logger()
	assert.NotNil(t, lg, "logger initialization failed")
	assert.NotNil(t, lg.Writer(), "logger writer initialization failed")
	assert.True(t, len(lg.Prefix()) > 0, "logger prefix initialization failed")
}

func TestWriteLog(t *testing.T) {
	WriteLog("testing logger %s", "again")
}

func TestWriteLogAndReturnError(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expErr error
	}{
		{
			name:   "empty error",
			input:  "",
			expErr: errors.New(""),
		},
		{
			name:   "empty error",
			input:  "test error",
			expErr: errors.New("test error"),
		},
		{
			name:   "empty error",
			input:  fmt.Sprintf("test error %d", 2),
			expErr: fmt.Errorf("test error %d", 2),
		},
	}
	for _, tc := range tests {
		err := WriteLogAndReturnError(tc.input)
		assert.Equal(t, tc.expErr, err, "%s failed, error mismatch")
	}
}
