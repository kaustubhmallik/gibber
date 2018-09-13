package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"syscall"
)

const (
	Host = "127.0.0.1"
	Port = "7000"
)

func StartServer() error {
	address := fmt.Sprintf("%s:%s", Host, Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return WriteLogAndReturnError("error in starting listener on host %s and port %s: %s", Host, Port, err)
	}
	GetLogger().Printf("started TCP listener on %s", address)
	for {
		conn, err := listener.Accept()
		if err != nil {
			if conn.RemoteAddr().String() != "" {
				GetLogger().Printf("client %s => connection establishment failed", conn.RemoteAddr().String())
			}
			continue // some error occurred
		}
		GetLogger().Printf("client %s => connection established successfully", conn.RemoteAddr().String())
		go startNewChatSession(&conn)
	}
}

func startNewChatSession(conn *net.Conn) {
	// when connection is closed from client, the resource need to be released
	defer closeChatSession(conn)

	reader := bufio.NewReader(*conn)
	writer := bufio.NewWriter(*conn)
	var err error
	msg := "Welcome to Gibber. How is your mood today?"
	var clientMsg string
	for {
		// write msg to client
		_, err = writer.WriteString(msg + "\n")
		if err != nil {
			if err == syscall.EPIPE { // checking if broken pipe
				return // connection will be closed inside deferred function
			}
			//WriteLog("client %s => writing message failed: %s", conn.RemoteAddr().String(), err)
			//WriteLog("client => writing message failed: %s", err)
			GetLogger().Printf("client %s => writing message failed: %s", (*conn).RemoteAddr().String(), err)
		} else {
			//WriteLog("client %s => message read: %s", conn.RemoteAddr().String(), msg)
			//WriteLog("client => message read: %s", msg)
			GetLogger().Printf("client %s => message sent: %s", (*conn).RemoteAddr().String(), msg)
		}
		err = writer.Flush()
		if err != nil {
			GetLogger().Printf("client %s => write flush failed: %s", (*conn).RemoteAddr().String(), err)
		}

		// reading msg from client
		clientMsg, err = reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if len(clientMsg) > 0 {
					GetLogger().Printf("client %s => message partially read: %s", (*conn).RemoteAddr().String(),
						clientMsg)
				} else { // connection is closed, net.OpError occurred
					// NOTE: An intentional EOF (Ctrl-D is considered as closing the connection similar to bash)
					GetLogger().Printf("client %s => connection closed from client", (*conn).RemoteAddr().String())
					return
				}
			}
		} else if len(clientMsg) > 0 { // some non-empty message
			// TODO: I think len() > 0 check can be remove, an else will suffice
			GetLogger().Printf("client %s => message read: %s", (*conn).RemoteAddr().String(), clientMsg)
		}
	}
}

func closeChatSession(conn *net.Conn) {
	GetLogger().Printf("client %s => closing connection from server", (*conn).RemoteAddr().String())
	(*conn).Close()
}
