package client

import (
	"bufio"
	"fmt"
	"github.com/prometheus/common/log"
	"net"
)

const (
	Host = "127.0.0.1"
	Port = "7000"
)

const ConnectionType = "tcp"

var conn net.Conn
var reader *bufio.Reader
var writer *bufio.Writer
var err error

func StartClient() {
	// establishing connection
	address := fmt.Sprintf("%s:%s", Host, Port)
	conn, err = net.Dial(ConnectionType, address)
	if err != nil {
		log.Fatalf("error while establishing connection to the gibber server: %s", err)
	}

	// setting read and writer connection buffers
	reader = bufio.NewReader(conn)
	writer = bufio.NewWriter(conn)
}



