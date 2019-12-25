package service

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewFriend(t *testing.T) {
	assert.Equal(t, new(Friends), NewFriends(), "initializing new friends")
}
