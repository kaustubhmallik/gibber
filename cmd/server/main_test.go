package main

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

func TestMainFunc(t *testing.T) {
	go main()
	time.Sleep(time.Second) // wait for the server to get started

	// establish a tcp client
	conn, err := net.Dial("tcp", "localhost:7000")
	assert.NoError(t, err, "connection to server fail: %s", err)
	assert.NotNil(t, conn, "connection is not established")
	assert.Equal(t, "tcp", conn.RemoteAddr().Network(), "connection scheme is incorrect: %s",
		conn.RemoteAddr().Network())
	assert.Equal(t, "[::1]:7000", conn.RemoteAddr().String(), "connection scheme is incorrect: %s",
		conn.RemoteAddr().Network())
}
