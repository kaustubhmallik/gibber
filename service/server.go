package service

import (
	"bufio"
	"context"
	"fmt"
	"gibber/log"
	"gibber/user"
	"io/ioutil"
	"net"
	"path/filepath"
	"runtime"
	"strings"
)

// client connection type
const connType = "tcp"

const (
	logoFilePath = "assets/logo.txt"
	repo         = "gibber/"
)

// StartServer starts the chat server and opens it given patterns of hosts and the given port
// The context is taken to signal that the server is initialized successfully
func StartServer(host, port string, complete context.CancelFunc) error {
	defer complete() // mark the context as completed/cancelled
	address := fmt.Sprintf("%s:%s", host, port)
	listener, err := net.Listen(connType, address)
	if err != nil {
		return fmt.Errorf("error in starting listener on host %s and port %s: %s", host, port, err)
	}
	log.Logger().Printf("started TCP listener on %s", address)
	_ = printLogo()
	complete() // as the next step is infinite loop
	for {
		conn, err := listener.Accept()
		if err != nil {
			if conn.RemoteAddr().String() != "" {
				log.Logger().Printf("client %s => connection establishment failed", conn.RemoteAddr().String())
			}
			continue // some error occurred
		}
		log.Logger().Printf("client %s => connection established successfully", conn.RemoteAddr().String())
		go establishClientConnection(&conn)
	}
}

// establishClientConnection setups the read and write streams to allow communication b/w server and client
// It passes the flow to the user back by showing landing page (dashboard)
func establishClientConnection(conn *net.Conn) {
	// when connection is closed from client, the resource need to be released
	defer closeClientConnection(conn)
	client := &client{}
	client.User = &user.User{}
	client.Connection = &Connection{}
	client.Conn = conn

	client.Reader = bufio.NewReader(*client.Conn)
	client.Writer = bufio.NewWriter(*client.Conn)

	client.showWelcomeMessage()
	if client.Err != nil {
		return
	}

	client.authenticate()
	defer client.logoutUser()
	if client.Err != nil {
		return
	}

	client.userDashboard()
}

// closeClientConnection safely closes the connection when the client exits or gets disconnected
// to avoid any memory leak
func closeClientConnection(conn *net.Conn) {
	log.Logger().Printf("client %s => closing connection from internal", (*conn).RemoteAddr().String())
	_ = (*conn).Close()
}

// printLogo prints the service logo (via logger)
func printLogo() (err error) {
	filePath := projectRootPath() + logoFilePath
	logoData, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Logger().Printf("reading logo file %s filed failed: %s", filePath, err)
		return
	}
	log.Logger().Println(string(logoData[:]))
	return
}

// projectRootPath gives the path to the project root
// Useful for navigating files inside the repo
func projectRootPath() (path string) {
	_, fileStr, _, _ := runtime.Caller(0)
	rootPath := strings.Split(filepath.Dir(fileStr), repo)
	return rootPath[0] + repo
}
