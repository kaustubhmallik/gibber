package service

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPrintLogo(t *testing.T) {
	assert.NoError(t, nil, PrintLogo(), "printing logo resulted in error")
}
