package service

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestStartServer(t *testing.T) {
	go func() {
		_ = StartServer("127.0.0.1", "13090")
	}()

	// connect to server
	conn, err := net.Dial("tcp", "127.0.0.1:13090")
	assert.NoError(t, err, "connection unsuccessful")
	assert.NotEmpty(t, conn, "connection unsuccessful")
	assert.Equal(t, "127.0.0.1:13090", conn.RemoteAddr().String(), "not connected to server above")

	establishClientConnection(&conn)
}
