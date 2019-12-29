package service

import (
	"bufio"
	"context"
	"fmt"
	"net"
)

const ConnectionType = "tcp"

func StartServer(host, port string, complete context.CancelFunc) error {
	defer complete() // mark the context as completed/cancelled
	address := fmt.Sprintf("%s:%s", host, port)
	listener, err := net.Listen(ConnectionType, address)
	if err != nil {
		return WriteLogAndReturnError("error in starting listener on host %s and port %s: %s", host, port, err)
	}
	Logger().Printf("started TCP listener on %s", address)
	_ = PrintLogo()
	complete() // as the next step is infinite loop
	for {
		conn, err := listener.Accept()
		if err != nil {
			if conn.RemoteAddr().String() != "" {
				Logger().Printf("client %s => connection establishment failed", conn.RemoteAddr().String())
			}
			continue // some error occurred
		}
		Logger().Printf("client %s => connection established successfully", conn.RemoteAddr().String())
		go establishClientConnection(&conn)
	}
}

func establishClientConnection(conn *net.Conn) {
	// when connection is closed from client, the resource need to be released
	defer closeClientConnection(conn)
	client := &Client{}
	client.User = &User{}
	client.Connection = &Connection{}
	client.Conn = conn

	client.Reader = bufio.NewReader(*client.Conn)
	client.Writer = bufio.NewWriter(*client.Conn)

	client.ShowWelcomeMessage()
	if client.Err != nil {
		return
	}

	client.Authenticate()
	defer client.LogoutUser()
	if client.Err != nil {
		return
	}

	client.UserDashboard()
}

func closeClientConnection(conn *net.Conn) {
	Logger().Printf("client %s => closing connection from internal", (*conn).RemoteAddr().String())
	_ = (*conn).Close()
}
