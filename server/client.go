package server

import (
	"fmt"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/pkg/errors"
)

// a client using the gibber app
type Client struct {
	*User
	*Connection
}

const WelcomeMsg = "Welcome to Gibber. Hope you have a lot to say today."
const EmailPrompt = "\nPlease enter your email to continue.\nEmail: "
const PasswordPrompt = "\nYou are already a registered user. Please enter password to continue.\nPassword: "
const NewUserMsg = "You are an unregistered user. Please register yourself by providing details.\n"
const FirstNamePrompt = "First Name: "
const LastNamePrompt = "Last Name: "
const SuccessfulLogin = "\nLogged In Successfully"
const SuccessfulRegistration = "\nRegistered Successfully"
const SetPasswordPrompt = "New Password: "
const ConfirmSetPasswordPrompt = "Confirm Password: "

func (c *Client) ShowWelcomeMessage() {
	c.SendMessage(WelcomeMsg, true)
	if c.Err != nil {
		GetLogger().Printf("writing welcome message to client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
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
	c.SendMessage(EmailPrompt, false)
	if c.Err != nil {
		GetLogger().Printf("writing email prompt message to client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
	}
	c.User.Email = c.ReadMessage()
	if c.Err != nil {
		GetLogger().Printf("reading user email from client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
	}
	return
}

// TODO: Add mongo client, and check from users collections whether the given email exists
func (c *Client) ExistingUser() (exists bool) {
	_, c.Err = GetUser(c.User.Email) // if user not exists, it will throw an error
	if c.Err == mongo.ErrNoDocuments {
		c.Err = nil // resetting the error
		return
	}
	if c.Err != nil { // some other error occurred
		reason := fmt.Sprintf("existing user check for client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
		GetLogger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	exists = true
	return
}

// TODO: Take user password and check with hashed stored
func (c *Client) LoginUser() {
	c.SendMessage(PasswordPrompt, false)
	if c.Err != nil {
		c.Err = fmt.Errorf("user password prompt failed: %s", c.Err)
		return
	}
	password := c.ReadMessage()
	if c.Err != nil {
		c.Err = fmt.Errorf("reading user password failed: %s", c.Err)
		return
	}
	c.Err = c.User.AuthenticateUser(password)
	if c.Err != nil {
		reason := fmt.Sprintf("user %s authentication failed: %s", c.User.Email, c.Err)
		GetLogger().Println(reason)
		c.Err = fmt.Errorf(reason)
	}
	GetLogger().Printf("user %s successfully logged in", c.User.Email)
	c.SendMessage(SuccessfulLogin, true)
	if c.Err != nil {
		reason := fmt.Sprintf("successful login msg failed to client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
		GetLogger().Println(reason)
		c.Err = errors.New(reason)
	}
}

// TODO: register user name and age
func (c *Client) RegisterUser() {
	c.SendMessage(NewUserMsg, true)
	if c.Err != nil {
		reason := fmt.Sprintf("new user message sending failed: %s", c.Err)
		GetLogger().Println(reason)
		c.Err = fmt.Errorf(reason)
		return
	}

	firstName := c.SendAndReceiveMsg(FirstNamePrompt, false)
	if c.Err != nil {
		reason := fmt.Sprintf("reading user password failed: %s", c.Err)
		GetLogger().Println(reason)
		c.Err = fmt.Errorf(reason)
		return
	}
	c.User.FirstName = firstName

	lastName := c.SendAndReceiveMsg(LastNamePrompt, false)
	if c.Err != nil {
		reason := fmt.Sprintf("reading user last name failed: %s", c.Err)
		GetLogger().Println(reason)
		c.Err = fmt.Errorf(reason)
		return
	}
	c.User.LastName = lastName

	password := c.SendAndReceiveMsg(SetPasswordPrompt, false)
	if c.Err != nil {
		reason := fmt.Sprintf("reading user new password failed: %s", c.Err)
		GetLogger().Println(reason)
		c.Err = fmt.Errorf(reason)
		return
	}

	confPassword := c.SendAndReceiveMsg(ConfirmSetPasswordPrompt, false)
	if c.Err != nil {
		reason := fmt.Sprintf("reading user confirm password failed: %s", c.Err)
		GetLogger().Println(reason)
		c.Err = errors.New(reason)
		return
	} else if password != confPassword {
		reason := "passwords not matched"
		GetLogger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	c.User.Password = password

	_, c.Err = CreateUser(c.User)
	if c.Err != nil {
		reason := fmt.Sprintf("user %s registration failed: %s", c.User.Email, c.Err)
		GetLogger().Println(reason)
		c.Err = fmt.Errorf(reason)
	}

	GetLogger().Printf("user %s successfully regsistered", c.User)
	c.SendMessage(SuccessfulRegistration, true)
	if c.Err != nil {
		reason := fmt.Sprintf("successful registration msg failed to client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
		GetLogger().Println(reason)
		c.Err = errors.New(reason)
	}
}

func (c *Client) SendAndReceiveMsg(msgToSend string, newline bool) (msgRecvd string) {
	c.SendMessage(msgToSend, newline)
	if c.Err != nil {
		return
	}
	msgRecvd = c.ReadMessage()
	if c.Err != nil {
		c.Err = fmt.Errorf("reading failed: %s", c.Err)
	}
	return
}

func (c *Client) ShowConnectedPeople() {

}
