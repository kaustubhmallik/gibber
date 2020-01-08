package service

import (
	"bufio"
	"gibber/log"
	"net"
	"strings"
)

// Connection details of the TCP connection b/w client and service
type Connection struct {
	Conn   *net.Conn
	Reader *bufio.Reader
	Writer *bufio.Writer
	Err    error
}

// sendMessage sends a given message to the client using underlying connection write buffer
func (c *Connection) sendMessage(msg string, newline bool) {
	if newline {
		msg += "\n"
	}
	_, c.Err = c.Writer.WriteString(msg)
	if c.Err != nil {
		log.Logger().Printf("error while writing to %s: %s", (*c.Conn).RemoteAddr(), c.Err)
		return
	}
	c.Err = c.Writer.Flush()
	if c.Err != nil {
		log.Logger().Printf("error while flushing data to %s: %s", (*c.Conn).RemoteAddr(), c.Err)
		return
	}
}

// readMessage reads a single line (until end-of-line) of user input from connection read stream
func (c *Connection) readMessage() (content string) {
	content, c.Err = c.Reader.ReadString('\n')
	content = strings.TrimRight(content, "\n")
	if c.Err != nil {
		log.Logger().Printf("error while reading from %s: %s", (*c.Conn).RemoteAddr(), c.Err)
	}
	return
}
