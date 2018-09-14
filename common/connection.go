package common

import (
	"bufio"
	"fmt"
	"net"
)

// connection details
type Connection struct {
	Conn   net.Conn
	Reader *bufio.Reader
	Writer *bufio.Writer
	Err    error
}

// writes a single line on writer by appending the newline to the passed string
func (c *Connection) SendMessage(msg string, newline bool) {
	_, c.Err = c.Writer.WriteString(msg)
	if c.Err != nil {
		c.Err = fmt.Errorf("error while writing to %s: %s", c.Conn.RemoteAddr(), c.Err)
		return
	}
	if newline {
		c.Writer.WriteString(msg)
		if c.Err != nil {
			c.Err = fmt.Errorf("error while writing to %s: %s", c.Conn.RemoteAddr(), c.Err)
			return
		}
	}
	c.Err = c.Writer.Flush()
	if c.Err != nil {
		c.Err = fmt.Errorf("error while flushing write to %s: %s", c.Conn.RemoteAddr(), c.Err)
	}
}

// reads a single line from reader
func (c *Connection) ReadMessage() (content string) {
	content, c.Err = c.Reader.ReadString('\n')
	if c.Err != nil {
		c.Err = fmt.Errorf("error while writing to %s: %s", c.Conn.RemoteAddr(), c.Err)
	}
	return
}
