package service

import (
	"context"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

func TestStartServer(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 3*time.Second)
	go func(f context.CancelFunc) {
		_ = StartServer("localhost", "34510", f)
	}(cancelFunc)
	<-ctx.Done()

	//connect to server
	conn, err := net.Dial("tcp", "localhost:34510")
	assert.NoError(t, err, "connection unsuccessful")
	assert.NotEmpty(t, conn, "connection unsuccessful")

	establishClientConnection(&conn)
}
