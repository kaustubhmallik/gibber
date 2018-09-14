package common

import (
	"fmt"
	"gibber/server"
)

// a client using the gibber app
type Client struct {
	User
	Connection
}

const welcomeMsg = "Welcome to Gibber. Hope you have a lot to say today."
const eemailPrompt = "Please enter your email to continue.\nEmail: "

func (c *Client) ShowWelcomeMessage() {
	c.SendMessage(welcomeMsg, true)
	if c.Err != nil {
		server.GetLogger().Printf("writing welcome message to client %s failed: %s", c.Conn.RemoteAddr(), c.Err)
	}
}

func (c *Client) Authenticate() {
	c.PromptForEmail()
	exists := c.ExistingUser()
	if c.Err != nil {
		return
	}
	if exists {
		c.LoginUser()
	} else {
		c.RegisterUser()
	}
	return
}

func (c *Client) PromptForEmail() {
	c.SendMessage(eemailPrompt, false)
	if c.Err != nil {
		server.GetLogger().Printf("writing email prompt message to client %s failed: %s", c.Conn.RemoteAddr(), c.Err)
	}
	c.User.Email = c.ReadMessage()
	if c.Err != nil {
		server.GetLogger().Printf("reading user email from client %s failed: %s", c.Conn.RemoteAddr(), c.Err)
	}
	return
}

// TODO: Add mongo client, and check from users collections whether the given email exists
func (c *Client) ExistingUser() (exists bool) {
	c.Err = fmt.Errorf("existing user check for client %s failed: %s", c.Conn.RemoteAddr(), c.Err)
	return
}

// TODO: Take user password and check with hashed stored
func (c *Client) LoginUser() {
	c.Err = fmt.Errorf("user login for client %s failed: %s", c.Conn.RemoteAddr(), c.Err)
}

// TODO: register user name and age
func (c *Client) RegisterUser() {
	c.Err = fmt.Errorf("user registration for client %s failed: %s", c.Conn.RemoteAddr(), c.Err)
}
