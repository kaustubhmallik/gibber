package service

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

func TestStartServer(t *testing.T) {
	go func() {
		_ = StartServer("localhost", "39597")
	}()
	time.Sleep(2 * time.Second)

	//connect to server
	conn, err := net.Dial("tcp", "localhost:39597")
	assert.NoError(t, err, "connection unsuccessful")
	assert.NotEmpty(t, conn, "connection unsuccessful")

	establishClientConnection(&conn)
}
