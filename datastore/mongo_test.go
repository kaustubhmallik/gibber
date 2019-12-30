package datastore

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMongoConn(t *testing.T) {
	conn := MongoConn()
	assert.NotNil(t, conn, "mongo connection is nil")
}
