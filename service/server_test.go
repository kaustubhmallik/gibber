package service

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

func TestStartServer(t *testing.T) {
	go func() {
		_ = StartServer("localhost", "34510")
	}()
	time.Sleep(time.Second * 3)

	//connect to server
	conn, err := net.Dial("tcp", "localhost:34510")
	assert.NoError(t, err, "connection unsuccessful")
	assert.NotEmpty(t, conn, "connection unsuccessful")

	establishClientConnection(&conn)
}
