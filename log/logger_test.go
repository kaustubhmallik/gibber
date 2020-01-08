package log

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLogger(t *testing.T) {
	lg := Logger()
	assert.NotNil(t, lg, "logger initialization failed")
	assert.NotNil(t, lg.Writer(), "logger writer initialization failed")
	assert.True(t, len(lg.Prefix()) > 0, "logger prefix initialization failed")
}