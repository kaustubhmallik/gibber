package service

import (
	"bufio"
	"github.com/stretchr/testify/assert"
	"log"
	"net"
	"testing"
	"time"
)

var conn *Connection

const (
	Host = "0.0.0.0"
	Port = "12197"
)

func init() {
	go func() {
		log.Fatal(StartServer(Host, Port)) // start tcp server
	}()
	time.Sleep(time.Second)

	co, err := net.Dial("tcp", Host+":"+Port) // connect as client
	if err != nil {
		log.Fatalf("error in connecting to server: %s", err)
	}

	conn = new(Connection)
	conn.Conn = &co
	conn.Reader = bufio.NewReader(co)
	conn.Writer = bufio.NewWriter(co)
}

func TestConnection_SendMessage(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		newline bool
	}{
		{
			name:    "normal msg without newline",
			msg:     "test message",
			newline: false,
		},
		{
			name:    "normal msg with newline",
			msg:     "test message",
			newline: true,
		},
		{
			name:    "empty msg without newline",
			msg:     "",
			newline: false,
		},
		{
			name:    "empty msg with newline",
			msg:     "",
			newline: true,
		},
	}
	for _, tc := range tests {
		conn.SendMessage(tc.msg, tc.newline)
		assert.NoError(t, conn.Err)
	}
}

func TestConnection_ReadMessage(t *testing.T) {
	content := conn.ReadMessage()
	assert.NoError(t, conn.Err)
	assert.NotEmpty(t, content, "non-empty message received")
}
