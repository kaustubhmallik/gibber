package server

import (
	"fmt"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/pkg/errors"
	"strconv"
)

// a client using the gibber app
type Client struct {
	*User
	*Connection
}

// User Response Msgs
const (
	WelcomeMsg               = "Welcome to Gibber. Hope you have a lot to say today."
	EmailPrompt              = "\nPlease enter your email to continue.\nEmail: "
	ReenterEmailPrompt       = "Please re-enter your email.\nEmail: "
	PasswordPrompt           = "\nYou are already a registered user. Please enter password to continue.\nPassword: "
	ReenterPasswordPrompt    = "\nPlease re-enter your password.\nPassword: "
	NewUserMsg               = "You are an unregistered user. Please register yourself by providing details.\n"
	FirstNamePrompt          = "First Name: "
	LastNamePrompt           = "Last Name: "
	SuccessfulLogin          = "\nLogged In Successfully\n"
	FailedLogin              = "Log In Failed"
	SuccessfulRegistration   = "\nRegistered Successfully"
	FailedRegistration       = "\nRegistration Failed"
	SetPasswordPrompt        = "New Password: "
	ConfirmSetPasswordPrompt = "Confirm Password: "
)

// specific errors
const (
	PasswordMismatch = "Incorrect password"
	InvalidEmail     = "Invalid email"
	ServerError      = "Server processing error"
	EmptyInput       = "Empty input\n"
	ShortPassword    = "Password should be at 6 characters long"
	ReadingError     = "Error while receiving data at server"
	ExitingMsg       = "Exiting..."
	InvalidInput     = "Invalid input\n"
)

const (
	DashboardHeader = "********************** Welcome to Gibber ************************\n\nPlease select one of " +
		"the option from below.\n"
	UserMenu = "\n0 - Exit\n1 - Start/Resume Chat\n2 - Add new connection\n3 - See new inviations\n4 - Change password\n" +
		"5 - Change Name\n\nEnter a choice: "
)

const (
	ExitChoice = iota
	StartChatChoice
	SendInvitationChoice
	SeeInvitationChoice
	ChangePasswordChoice
	ChangeNameChoice
)

const EmptyString = ""

func (c *Client) ShowWelcomeMessage() {
	c.SendMessage(WelcomeMsg, true)
	if c.Err != nil {
		GetLogger().Printf("writing welcome message to client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
	}
}

func (c *Client) Authenticate() {
	c.PromptForEmail()
	if c.Err != nil {
		return
	}
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
	for failureCount := 0; failureCount < 3; failureCount++ {
		if failureCount == 0 {
			c.User.Email = c.SendAndReceiveMsg(EmailPrompt, false)
		} else {
			c.User.Email = c.SendAndReceiveMsg(ReenterEmailPrompt, false)
		}
		if c.Err != nil {
			//c.SendMessage(ReadingError, true)
			//GetLogger().Printf("reading user email from client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
			continue
		}
		if !ValidUserEmail(c.User.Email) { // check for valid email - regex based
			GetLogger().Printf("invalid email %s", c.User.Email)
			c.SendMessage(InvalidEmail, true)
			if c.Err != nil {
				GetLogger().Printf("sending invalud email msg to client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
				return
			}
			c.Err = errors.New(InvalidEmail)
			continue
		}
		return // successfully read valid email from user
	}
	c.ExitClient()
	c.Err = errors.New("reading email failed")
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
	for failureCount := 0; failureCount < 3; failureCount++ {
		if failureCount == 0 {
			c.SendMessage(PasswordPrompt, false)
		} else {
			c.SendMessage(ReenterPasswordPrompt, false)
		}
		if c.Err != nil {
			c.Err = fmt.Errorf("user password prompt failed: %s", c.Err)
			continue
		}
		password := c.ReadMessage()
		if c.Err != nil {
			c.Err = fmt.Errorf("reading user password failed: %s", c.Err)
			continue
		}
		c.Err = c.User.AuthenticateUser(password)
		if c.Err != nil {
			reason := fmt.Sprintf("user %s authentication failed: %s", c.User.Email, c.Err)
			GetLogger().Println(reason)
			if c.Err.Error() == PasswordMismatch {
				c.SendMessage(FailedLogin+": "+PasswordMismatch, true)
			} else {
				c.SendMessage(FailedLogin+": "+ServerError, true)
			}
			c.Err = errors.New(reason)
			continue
		}
		GetLogger().Printf("user %s successfully logged in", c.User.Email)
		c.SendMessage(SuccessfulLogin, true)
		if c.Err != nil {
			reason := fmt.Sprintf("successful login msg failed to client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
			GetLogger().Println(reason)
			c.Err = errors.New(reason)
		}
		c.SendMessage(DashboardHeader, true)
		if c.Err != nil {
			reason := fmt.Sprintf("dashboard header msg failed to send to client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
			GetLogger().Println(reason)
			c.Err = errors.New(reason)
		}
		return
	}
	c.ExitClient()
	c.Err = errors.New("reading password failed")
}

// TODO: register user name and age
func (c *Client) RegisterUser() {
	c.SendMessage(NewUserMsg, true)
	if c.Err != nil {
		reason := fmt.Sprintf("new user message sending failed: %s", c.Err)
		GetLogger().Println(reason)
		c.Err = errors.New(reason)
		return
	}

	firstName := c.SendAndReceiveMsg(FirstNamePrompt, false)
	if c.Err != nil {
		reason := fmt.Sprintf("reading user password failed: %s", c.Err)
		GetLogger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	c.User.FirstName = firstName

	lastName := c.SendAndReceiveMsg(LastNamePrompt, false)
	if c.Err != nil {
		reason := fmt.Sprintf("reading user last name failed: %s", c.Err)
		GetLogger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	c.User.LastName = lastName

	password := c.SendAndReceiveMsg(SetPasswordPrompt, false)
	if c.Err != nil {
		reason := fmt.Sprintf("reading user new password failed: %s", c.Err)
		GetLogger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	c.Err = ValidatePassword(password)
	if c.Err != nil {
		c.SendMessage(ShortPassword, true)
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
		c.SendMessage(FailedRegistration, true)
		c.Err = errors.New(reason)
		return
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
		reason := fmt.Sprintf("reading failed from client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
		GetLogger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	if msgRecvd == EmptyString {
		reason := fmt.Sprintf("empty string received from client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
		GetLogger().Println(reason)
		c.SendMessage(EmptyInput, true)
		if c.Err != nil {
			reason := fmt.Sprintf("sending empty input msg to client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
			GetLogger().Println(reason)
		}
		c.Err = errors.New(reason)
	}
	return
}

func (c *Client) UserDashboard() {
	exit := false
	var userInput string
	for !exit {
		userInput = c.ShowMenu()
		if c.Err != nil {
			continue
		}
		choice, err := strconv.Atoi(userInput)
		if err != nil {
			c.SendMessage(InvalidInput, true)
			continue
		}
		switch choice {
		case ExitChoice:
			c.ExitClient()
			exit = true
		case StartChatChoice:
			c.StarChat()
		case SendInvitationChoice:
			c.SendInvitation()
		case SeeInvitationChoice:
			c.SeeInvitation()
		case ChangePasswordChoice:
			c.ChangePassword()
		case ChangeNameChoice:
			c.ChangeName()
		default:
			c.SendMessage(InvalidInput, true)
			continue
		}
	}
}

func (c *Client) ShowMenu() string {
	return c.SendAndReceiveMsg(UserMenu, false)
}

func (c *Client) StarChat() {

}

func (c *Client) SendInvitation() {

}

func (c *Client) SeeInvitation() {

}

func (c *Client) ChangePassword() {
	var failureCount int
	for failureCount = 0; failureCount < 3; failureCount++ {
		currPassword := c.SendAndReceiveMsg("\nEnter your current password: ", false)
		if c.Err != nil {
			continue
		}
		if GetSHA512Encrypted(currPassword) != c.User.Password {
			reason := fmt.Sprintf("user %s entered incorrect password: %s", c.User.Email, c.Err)
			GetLogger().Println(reason)
			if c.Err.Error() == PasswordMismatch {
				c.SendMessage(PasswordMismatch, true)
			} else {
				c.SendMessage(ServerError, true)
			}
			c.Err = errors.New(reason)
			continue
		}
		break
	}
	if failureCount == 3 {
		return // user unable to enter current password
	}
	for failureCount = 0; failureCount < 3; failureCount++ {
		newPassword := c.SendAndReceiveMsg("\nEnter your new password: ", false)
		if c.Err != nil {
			continue
		}
		c.Err = ValidatePassword(newPassword)
		if c.Err != nil {
			c.SendMessage(ShortPassword, true)
			continue
		}
		encryptedPassword := GetSHA512Encrypted(newPassword)
		if encryptedPassword == c.User.Password {
			reason := "New Password is same as current. Please select a different one."
			GetLogger().Println(reason)
			c.SendMessage(reason, true)
			c.Err = errors.New(reason)
			continue
		}
		confirmNewPassword := c.SendAndReceiveMsg("\nConfirm your new password: ", false)
		if c.Err != nil {
			continue
		}
		if newPassword != confirmNewPassword {
			reason := "Passwords didn't match"
			GetLogger().Println(reason)
			c.SendMessage(reason, true)
			c.Err = errors.New(reason)
			continue
		}
		c.User.UpdatePassword(encryptedPassword)
		c.SendMessage("Password successfully updated\n", true)
		return
	}
}

func (c *Client) ChangeName() {

}

func (c *Client) ExitClient() {
	c.SendMessage(ExitingMsg, true)
	if c.Err != nil {
		reason := fmt.Sprintf("sending exit msg to client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
		GetLogger().Println(reason)
	}
}

func ValidatePassword(password string) (err error) {
	if len(password) < 6 {
		reason := fmt.Sprintf("%s password: %s", ShortPassword, password)
		GetLogger().Println(reason)
		err = errors.New(reason)
	}
	return
}