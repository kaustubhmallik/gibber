package server

import (
	"bufio"
	"fmt"
	"github.com/pkg/errors"
	"net"
	"strings"
)

// connection details
type Connection struct {
	Conn   *net.Conn
	Reader *bufio.Reader
	Writer *bufio.Writer
	Err    error
}

// writes a single line on writer by appending the newline to the passed string
func (c *Connection) SendMessage(msg string, newline bool) {
	if newline {
		msg += "\n"
	}
	_, c.Err = c.Writer.WriteString(msg)
	if c.Err != nil {
		reason := fmt.Sprintf("error while writing to %s: %s", (*c.Conn).RemoteAddr(), c.Err)
		c.Err = errors.New(reason)
		GetLogger().Printf(reason)
		return
	}
	c.Err = c.Writer.Flush()
	if c.Err != nil {
		reason := fmt.Sprintf("error while flushing data to %s: %s", (*c.Conn).RemoteAddr(), c.Err)
		c.Err = errors.New(reason)
		GetLogger().Printf(reason)
		return
	}
	GetLogger().Println("write successful")
}

// reads a single line from reader
func (c *Connection) ReadMessage() (content string) {
	content, c.Err = c.Reader.ReadString('\n')
	content = strings.TrimRight(content, "\n")
	if c.Err != nil {
		reason := fmt.Sprintf("error while writing to %s: %s", (*c.Conn).RemoteAddr(), c.Err)
		c.Err = errors.New(reason)
		GetLogger().Printf(reason)
	}
	return
}