package service

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestStartServer(t *testing.T) {
	go func() {
		_ = StartServer("localhost", "34517")
	}()

	//connect to server
	conn, err := net.Dial("tcp", "localhost:34517")
	assert.NoError(t, err, "connection unsuccessful")
	assert.NotEmpty(t, conn, "connection unsuccessful")

	establishClientConnection(&conn)
}
