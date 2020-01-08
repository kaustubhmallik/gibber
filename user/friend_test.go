package user

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewFriend(t *testing.T) {
	assert.Equal(t, new(friends), new(friends), "initializing new friends")
}
