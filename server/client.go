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
	SendInvitationInfo       = "You can search other people uniquely by their email.\n"
	EmailSearchPrompt        = "Email(\"q\" to quit): "
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

// user menus
const (
	DashboardHeader = "********************** Welcome to Gibber ************************" +
		"\n\nPlease select one of the option from below."
	UserMenu = "\n0 - Exit" +
		"\n1 - Start/Resume Chat" +
		"\n2 - See All Friends" +
		"\n3 - Send invitation" +
		"\n4 - See all invitations" +
		"\n5 - Change password" +
		"\n6 - Change Name" +
		"\n7 - See your profile" +
		"\n\nEnter a choice: "
	InvitationMenu = "\n0 - Go back to previous menu" +
		"\n1 - Active Sent Invites" +
		"\n2 - Active Received Invites" +
		"\n3 - Inactive Sent Invites" +
		"\n4 - Inactive Received Invites" +
		"\n\nEnter a choice: "
)

const (
	ExitChoice = iota
	StartChatChoice
	SeeAllFriends
	SendInvitationChoice
	SeeInvitationChoice
	ChangePasswordChoice
	ChangeNameChoice
	SeeProfileChoice
)

const (
	ActiveSentInvitesChoice = 1 + iota
	ActiveReceivedInvitesChoice
	InactiveSentInvitesChoice
	InactiveReceivedInvitesChoice
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
			c.Email = c.SendAndReceiveMsg(EmailPrompt, false)
		} else {
			c.Email = c.SendAndReceiveMsg(ReenterEmailPrompt, false)
		}
		if c.Err != nil {
			//c.SendMessage(ReadingError, true)
			//GetLogger().Printf("reading user email from client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
			continue
		}
		if !ValidUserEmail(c.Email) { // check for valid email - regex based
			GetLogger().Printf("invalid email %s", c.Email)
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
	_, c.Err = GetUser(c.Email) // if user not exists, it will throw an error
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
		c.Err = c.User.LoginUser(password)
		if c.Err != nil {
			reason := fmt.Sprintf("user %s authentication failed: %s", c.Email, c.Err)
			GetLogger().Println(reason)
			if c.Err.Error() == PasswordMismatch {
				c.SendMessage(FailedLogin+": "+PasswordMismatch, true)
			} else {
				c.SendMessage(FailedLogin+": "+ServerError, true)
			}
			c.Err = errors.New(reason)
			continue
		}
		GetLogger().Printf("user %s successfully logged in", c.Email)
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
	c.FirstName = firstName

	lastName := c.SendAndReceiveMsg(LastNamePrompt, false)
	if c.Err != nil {
		reason := fmt.Sprintf("reading user last name failed: %s", c.Err)
		GetLogger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	c.LastName = lastName

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
	c.Password = password

	_, c.Err = CreateUser(c.User)
	if c.Err != nil {
		reason := fmt.Sprintf("user %s registration failed: %s", c.Email, c.Err)
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
		userInput = c.ShowLandingPage()
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
			c.SeeOnlineFriends()
		case SeeAllFriends:
			c.SeeFriends()
		case SendInvitationChoice:
			c.SendInvitation()
		case SeeInvitationChoice:
			c.SeeInvitation()
		case ChangePasswordChoice:
			c.ChangePassword()
		case ChangeNameChoice:
			c.ChangeName()
		case SeeProfileChoice:
			c.ChangeName()
		default:
			c.SendMessage(InvalidInput, true)
			continue
		}
	}
}

func (c *Client) ShowLandingPage() string {
	return c.SendAndReceiveMsg(UserMenu, false)
}

func (c *Client) StarChat(friendsDetail string) {

}

func (c *Client) SendInvitation() {
	c.SendMessage(SendInvitationInfo, true)
	if c.Err != nil {

	}
	for {
		email := c.SendAndReceiveMsg(EmailSearchPrompt, false)
		if c.Err != nil {
			continue
		}
		if email == "q" {
			break
		}
		user, err := c.SeePublicProfile(email)
		if err == mongo.ErrNoDocuments { // user not found
			continue
		}
		c.SendMessage(fmt.Sprintf("Send invite to %s", email), false)
		confirm := c.SendAndReceiveMsg("Confirm? (Y/n): ", false)
		if c.Err != nil {
		}
		if confirm == "Y" || confirm == "y" || confirm == "" {
			err = c.User.SendInvitation(user)
		}
	}
}

func (c *Client) SeeInvitation() {
	for {
		userInput := c.SendAndReceiveMsg(InvitationMenu, false)
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
			return
		case ActiveSentInvitesChoice:
			c.SeeActiveSentInvitations()
		case ActiveReceivedInvitesChoice:
			c.SeeActiveReceivedInvitations()
		case InactiveSentInvitesChoice:
			c.SeeInactiveSentInvitations()
		case InactiveReceivedInvitesChoice:
			c.SeeInactiveReceivedInvitations()
		default:
			c.SendMessage(InvalidInput, true)
			continue
		}
	}

}

func (c *Client) SeeActiveReceivedInvitations() {
	invites, err := c.User.FetchActiveReceivedInvitations()
	if err != nil {
		reason := fmt.Sprintf("error while fetching active received invitations for user %s: %s", c.Email, err)
		GetLogger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	c.SendMessage("\n**** Active Received Invitations ****\n", true)
	for idx, invite := range invites {
		c.SendMessage(fmt.Sprintf("%d - %s", idx+1, invite), true)
	}
	userInput := c.SendAndReceiveMsg("\nChoose one to accept or reject(\"b to go back\"): ", false)
	if c.Err != nil {
		reason := fmt.Sprintf("error while receiving user invitation input from client %s: %s", (*c.Conn).RemoteAddr(), err)
		GetLogger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	if userInput == "b" || userInput == "B" {
		return
	}
	invitationIdx, err := strconv.Atoi(userInput)
	if err != nil || invitationIdx < 0 || invitationIdx > len(invites) {
		reason := fmt.Sprintf("invitation index input %s parsing failed from client %s: %s", userInput,
			(*c.Conn).RemoteAddr(), userInput)
		GetLogger().Println(reason)
		c.Err = errors.New(reason)
		c.SendMessage(fmt.Sprintf("Invalid choice: %s", userInput), true)
		return
	}
	// The user sees 1-based indexing, so reducing one from it
	inviteeUser, err := GetUser(invites[invitationIdx-1]) // user who sent this invitation
	if err != nil {
		reason := fmt.Sprintf("fetching invitee user %s details failed from client %s: %s", invites[invitationIdx],
			(*c.Conn).RemoteAddr(), userInput)
		GetLogger().Println(reason)
		c.Err = errors.New(reason)
		c.SendMessage("Internal error. Try again", true)
		c.SeeActiveReceivedInvitations()
	}
	c.SendMessage("\n===== Invitation Details =====\n", true)
	c.SendMessage(fmt.Sprintf("Name: %s %s", inviteeUser.FirstName, inviteeUser.LastName), true)
	c.SendMessage(fmt.Sprintf("Email: %s", inviteeUser.Email), true)
	confirm := c.SendAndReceiveMsg("\nConfirm(Y/n): ", false)
	if c.Err != nil {
	}
	if confirm == "Y" || confirm == "y" || confirm == "" {
		err = c.User.AddFriend(inviteeUser)
		if err != nil {
			c.SendMessage(fmt.Sprintf("\nAdding %s as friend failed\n", inviteeUser.Email), true)
			reason := fmt.Sprintf("adding %s as friend to %s failed: %s", c.User.Email, inviteeUser.Email, err)
			GetLogger().Println(reason)
			c.Err = errors.New(reason)
		} else {
			c.SendMessage("\nAdded %s as friend successfully\n", true)
			reason := fmt.Sprintf("adding %s as friend to %s failed: %s", c.User.Email, inviteeUser.Email, err)
			GetLogger().Println(reason)
		}
	}
}

func (c *Client) SeeActiveSentInvitations() {

}

func (c *Client) SeeInactiveReceivedInvitations() {

}

func (c *Client) SeeInactiveSentInvitations() {

}

func (c *Client) ChangePassword() {
	var failureCount int
	for failureCount = 0; failureCount < 3; failureCount++ {
		currPassword := c.SendAndReceiveMsg("\nEnter your current password: ", false)
		if c.Err != nil {
			continue
		}
		if GetSHA512Encrypted(currPassword) != c.Password {
			reason := fmt.Sprintf("user %s entered incorrect password: %s", c.Email, c.Err)
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
		if encryptedPassword == c.Password {
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
		err := c.User.UpdatePassword(encryptedPassword)
		if err != nil {
			c.Err = err
			c.SendMessage("Password update failed. Please try again.\n", true)
			return
		}
		c.SendMessage("Password successfully updated\n", true)
		return
	}
}

func (c *Client) ChangeName() {
	newFirstName := c.SendAndReceiveMsg("\nEnter your new first name(enter blank for skip): ", false)
	if c.Err != nil {
		//continue
		// add retry
	}
	if newFirstName == "" {
		reason := fmt.Sprintf("skipping first name change for user %s", c.Email)
		GetLogger().Println(reason)
	}

	newLastName := c.SendAndReceiveMsg("\nEnter your new last name(enter blank for skip): ", false)
	if c.Err != nil {
		//continue
		// add retry
	}
	if newLastName == "" {
		reason := fmt.Sprintf("skipping first name change for user %s", c.Email)
		GetLogger().Println(reason)
	}

	err := c.User.UpdateName(newFirstName, newLastName)
	if err != nil {
		c.Err = err
		c.SendMessage("Name update failed. Please try again.\n", true)
		return
	}

	c.SendMessage("Name successfully updated\n", true)
	return
}

func (c *Client) ExitClient() {
	c.SendMessage(ExitingMsg, true)
	if c.Err != nil {
		reason := fmt.Sprintf("sending exit msg to client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
		GetLogger().Println(reason)
	}
}

// enable client to see his/her own profile in detail
func (c *Client) SeeSelfProfile() {

}

// allow a client to see other person's basic detail before sending invitation
func (c *Client) SeePublicProfile(email string) (user *User, err error) {
	user, err = GetUser(email)
	if err == mongo.ErrNoDocuments {
		c.SendMessage(fmt.Sprintf("No user found with given email %s", email), true)
		if c.Err != nil {
			reason := fmt.Sprintf("error while sending no user found msg: %s", c.Err)
			GetLogger().Println(reason)
			return
		}
	}
	c.SendMessage(fmt.Sprintf("User found => First Name: %s, LastName: %s, Email: %s", user.FirstName, user.LastName,
		user.Email), true)
	return
}

func ValidatePassword(password string) (err error) {
	if len(password) < 6 {
		reason := fmt.Sprintf("%s password: %s", ShortPassword, password)
		GetLogger().Println(reason)
		err = errors.New(reason)
	}
	return
}

func (c *Client) SeeOnlineFriends() {
	friends, err := c.User.SeeFriends()
	if err != nil {
		reason := fmt.Sprintf("error while fetching online friends for client %s: %s", (*c.Conn).RemoteAddr(), err)
		GetLogger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	c.SendMessage("\n****************** Online Friends List *****************\n", true)
	for idx, friend := range friends {
		c.SendMessage(fmt.Sprintf("%d - %s", idx+1, friend), true)
	}
	// TODO: Pagination and search
	userInput := c.SendAndReceiveMsg("Enter a friend's index to start chat: ", false)
	friendIdx, err := strconv.Atoi(userInput)
	if err != nil {
		reason := fmt.Sprintf("error while parsing user input %s to start chat: %s", userInput, err)
		GetLogger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	// TODO: Check if valid friendIndex
	c.StarChat(friends[friendIdx])
}

func (c *Client) SeeFriends() {
	friends, err := c.User.SeeFriends()
	if err != nil {
		reason := fmt.Sprintf("error while fetching friends for client %s: %s", (*c.Conn).RemoteAddr(), err)
		GetLogger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	c.SendMessage("\n****************** Friends List *****************\n", true)
	for idx, friend := range friends {
		c.SendMessage(fmt.Sprintf("%d - %s", idx+1, friend), true)
	}
	// TODO: Pagination and search
	for {
		userInput := c.SendAndReceiveMsg("\nEnter 'b' key to go back\n", false)
		if userInput == "b" {
			break
		}
		c.SendMessage(fmt.Sprintf("Invalid input: %s", userInput), true)
	}
}

func (c *Client) LogoutUser() {
	if c.User.Email != "" {
		err := c.User.Logout()
		if err != nil {
			reason := fmt.Sprintf("error while logging out client %s: %s", err)
			GetLogger().Println(reason)
			c.Err = errors.New(reason)
		}
	}
}
