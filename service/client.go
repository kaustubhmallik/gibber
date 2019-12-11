package service

import (
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"strconv"
	"strings"
	"time"
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
	EmailSearchPrompt        = "\nEmail(\"q\" to quit): "
)

// specific errors
const (
	PasswordMismatch = "Incorrect password"
	InvalidEmail     = "Invalid email"
	ServerError      = "Server processing error"
	EmptyInput       = "Empty input\n"
	ShortPassword    = "Password should be at 6 characters long"
	ReadingError     = "Error while receiving data at internal"
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

func (c *Client) ShowWelcomeMessage() {
	c.SendMessage(WelcomeMsg, true)
	if c.Err != nil {
		Logger().Printf("writing welcome message to client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
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
			c.Email = c.SendAndReceiveMsg(EmailPrompt, false, false)
		} else {
			c.Email = c.SendAndReceiveMsg(ReenterEmailPrompt, false, false)
		}
		if c.Err != nil {
			//c.SendMessage(ReadingError, true)
			//Logger().Printf("reading user email from client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
			continue
		}
		if !ValidUserEmail(c.Email) { // check for valid email - regex based
			Logger().Printf("invalid email %s", c.Email)
			c.SendMessage(InvalidEmail, true)
			if c.Err != nil {
				Logger().Printf("sending invalud email msg to client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
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
	_, c.Err = GetUserByEmail(c.Email) // if user not exists, it will throw an error
	if c.Err == mongo.ErrNoDocuments {
		c.Err = nil // resetting the error
		return
	}
	if c.Err != nil { // some other error occurred
		reason := fmt.Sprintf("existing user check for client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
		Logger().Println(reason)
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
			Logger().Println(reason)
			if c.Err.Error() == PasswordMismatch {
				c.SendMessage(FailedLogin+": "+PasswordMismatch, true)
			} else {
				c.SendMessage(FailedLogin+": "+ServerError, true)
			}
			c.Err = errors.New(reason)
			continue
		}
		Logger().Printf("user %s successfully logged in", c.Email)
		c.SendMessage(SuccessfulLogin, true)
		if c.Err != nil {
			reason := fmt.Sprintf("successful login msg failed to client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
			Logger().Println(reason)
			c.Err = errors.New(reason)
		}
		c.SendMessage(DashboardHeader, true)
		if c.Err != nil {
			reason := fmt.Sprintf("dashboard header msg failed to send to client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
			Logger().Println(reason)
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
		Logger().Println(reason)
		c.Err = errors.New(reason)
		return
	}

	firstName := c.SendAndReceiveMsg(FirstNamePrompt, false, false)
	if c.Err != nil {
		reason := fmt.Sprintf("reading user password failed: %s", c.Err)
		Logger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	c.FirstName = firstName

	lastName := c.SendAndReceiveMsg(LastNamePrompt, false, false)
	if c.Err != nil {
		reason := fmt.Sprintf("reading user last name failed: %s", c.Err)
		Logger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	c.LastName = lastName

	password := c.SendAndReceiveMsg(SetPasswordPrompt, false, false)
	if c.Err != nil {
		reason := fmt.Sprintf("reading user new password failed: %s", c.Err)
		Logger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	c.Err = ValidatePassword(password)
	if c.Err != nil {
		c.SendMessage(ShortPassword, true)
		return
	}

	confPassword := c.SendAndReceiveMsg(ConfirmSetPasswordPrompt, false, false)
	if c.Err != nil {
		reason := fmt.Sprintf("reading user confirm password failed: %s", c.Err)
		Logger().Println(reason)
		c.Err = errors.New(reason)
		return
	} else if password != confPassword {
		reason := "passwords not matched"
		Logger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	c.Password = password

	_, c.Err = CreateUser(c.User)
	if c.Err != nil {
		reason := fmt.Sprintf("user %s registration failed: %s", c.Email, c.Err)
		Logger().Println(reason)
		c.SendMessage(FailedRegistration, true)
		c.Err = errors.New(reason)
		return
	}

	Logger().Printf("user %s successfully regsistered", c.User)
	c.SendMessage(SuccessfulRegistration, true)
	if c.Err != nil {
		reason := fmt.Sprintf("successful registration msg failed to client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
		Logger().Println(reason)
		c.Err = errors.New(reason)
	}
}

func (c *Client) SendAndReceiveMsg(msgToSend string, newline, emptyInputValid bool) (msgRecvd string) {
	c.SendMessage(msgToSend, newline)
	if c.Err != nil {
		return
	}
	msgRecvd = c.ReadMessage()
	if c.Err != nil {
		reason := fmt.Sprintf("reading failed from client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
		Logger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	if !emptyInputValid && msgRecvd == "" {
		reason := fmt.Sprintf("empty string received from client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
		Logger().Println(reason)
		c.SendMessage(EmptyInput, true)
		if c.Err != nil {
			reason := fmt.Sprintf("sending empty input msg to client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
			Logger().Println(reason)
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
	return c.SendAndReceiveMsg(UserMenu, false, false)
}

const ChatPrompt = "Type message (press \"enter\" to send, \"q\" to quit): "

func (c *Client) StarChat(friendID primitive.ObjectID) {
	content, timestamp := c.User.ShowChat(friendID)
	c.SendMessage(content, true)
	var input string
	done := make(chan bool)
	go c.PollIncomingMessages(friendID, done, timestamp)
	for {
		c.SendMessage(ChatPrompt, false)
		input = c.ReadMessage()
		if input == "" {
			c.SendMessage("Empty message can't be sent!!!", true)
			continue
		}
		if strings.ToLower(input) == "q" {
			done <- true // kill the incoming message listener
			break
		}
		err := SendMessage(c.User.ID, friendID, input)
		if err != nil {
			Logger().Print(err)
		}
		//c.SendMessage(fmt.Sprintf("You: %s\n", input), false)
		c.SendMessage(fmt.Sprintf("\bYou: %s\n", input), true)
		//c.SendMessage(fmt.Sprintf("\x0c\x0c\x0c\x0cYou: %s\n", input), true)
	}
}

func (c *Client) SendInvitation() {
	c.SendMessage(SendInvitationInfo, true)
	if c.Err != nil {
		// TODO: Handle error
	}
	for {
		email := c.SendAndReceiveMsg(EmailSearchPrompt, false, false)
		if c.Err != nil {
			continue
		}
		if strings.ToLower(email) == "q" {
			break
		}
		user, err := c.SeePublicProfile(email)
		if err == mongo.ErrNoDocuments { // user not found
			continue
		}
		c.SendMessage(fmt.Sprintf("Send invite to %s", email), false)
		confirm := c.SendAndReceiveMsg("Confirm? (Y/n): ", false, true)
		if c.Err != nil {
			// TODO: handle error
		}
		if strings.ToLower(confirm) == "y" || confirm == "" {
			err = c.User.SendInvitation(user)
			if err == nil {
				successMsg := fmt.Sprintf("\nInvitation sent successfully to %s %s (%s)", user.FirstName,
					user.LastName, user.Email)
				c.SendMessage(successMsg, true)
			}
		}
	}
}

func (c *Client) SeeInvitation() {
	for {
		userInput := c.SendAndReceiveMsg(InvitationMenu, false, false)
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
	invites, err := c.User.GetReceivedInvitations()
	if err != nil {
		reason := fmt.Sprintf("error while fetching active received invitations for user %s: %s", c.Email, err)
		Logger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	c.SendMessage("\n**** Active Received Invitations ****\n", true)
	for idx, invite := range invites {
		c.SendMessage(fmt.Sprintf("%d - %s", idx+1, UserProfile(invite)), true)
	}
	userInput := c.SendAndReceiveMsg("\nChoose one to accept or reject(\"b to go back\"): ", false,
		false)
	if c.Err != nil {
		reason := fmt.Sprintf("error while receiving user invitation input from client %s: %s", (*c.Conn).RemoteAddr(), err)
		Logger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	if strings.ToLower(userInput) == "b" {
		return
	}
	invitationIdx, err := strconv.Atoi(userInput)
	if err != nil || invitationIdx < 0 || invitationIdx > len(invites) {
		reason := fmt.Sprintf("invitation index input %s parsing failed from client %s: %s", userInput,
			(*c.Conn).RemoteAddr(), userInput)
		Logger().Println(reason)
		c.Err = errors.New(reason)
		c.SendMessage(fmt.Sprintf("Invalid choice: %s", userInput), true)
		return
	}
	// The user sees 1-based indexing, so reducing one from it
	inviteeUser, err := GetUserByID(invites[invitationIdx-1]) // user who sent this invitation
	if err != nil {
		reason := fmt.Sprintf("fetching invitee user %s details failed from client %s: %s", invites[invitationIdx],
			(*c.Conn).RemoteAddr(), userInput)
		Logger().Println(reason)
		c.Err = errors.New(reason)
		c.SendMessage("Internal error. Try again", true)
		c.SeeActiveReceivedInvitations()
		return
	}
	c.SendMessage("\n===== Invitation Details =====\n", true)
	c.SendMessage(fmt.Sprintf("Name: %s %s", inviteeUser.FirstName, inviteeUser.LastName), true)
	c.SendMessage(fmt.Sprintf("Email: %s", inviteeUser.Email), true)
	confirm := c.SendAndReceiveMsg("\nConfirm(Y/n): ", false, true)
	if c.Err != nil {
	}
	if strings.ToLower(confirm) == "y" || confirm == "" {
		err = c.User.AddFriend(inviteeUser.ID)
		if err != nil {
			c.SendMessage(fmt.Sprintf("\nAdding %s as friend failed\n", inviteeUser.Email), true)
			reason := fmt.Sprintf("adding %s as friend to %s failed: %s", c.User.Email, inviteeUser.Email, err)
			Logger().Println(reason)
			c.Err = errors.New(reason)
		} else {
			successMsg := fmt.Sprintf("\nAdded %s as friend successfully\n",
				inviteeUser.FirstName+" "+inviteeUser.LastName)
			c.SendMessage(successMsg, true)
			Logger().Print(successMsg)
		}
	}
}

func (c *Client) SeeActiveSentInvitations() {
	invites, err := c.User.GetSentInvitations()
	if err != nil {
		reason := fmt.Sprintf("error while fetching active sent invitations for user %s: %s", c.Email, err)
		Logger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	c.SendMessage("\n**** Active Sent Invitations ****\n", true)
	for idx, invite := range invites {
		c.SendMessage(fmt.Sprintf("%d - %s", idx+1, UserProfile(invite)), true)
	}
	userInput := c.SendAndReceiveMsg("\nChoose one to cancel(\"b to go back\"): ", false, false)
	if c.Err != nil {
		reason := fmt.Sprintf("error while seeing active user invitation sent from client %s: %s", (*c.Conn).RemoteAddr(), err)
		Logger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	if strings.ToLower(userInput) == "b" {
		return
	}
	invitationIdx, err := strconv.Atoi(userInput)
	if err != nil || invitationIdx < 0 || invitationIdx > len(invites) {
		reason := fmt.Sprintf("invitation index input %s parsing failed from client %s: %s", userInput,
			(*c.Conn).RemoteAddr(), userInput)
		Logger().Println(reason)
		c.Err = errors.New(reason)
		c.SendMessage(fmt.Sprintf("Invalid choice: %s", userInput), true)
		return
	}
	// The user sees 1-based indexing, so reducing one from it
	inviteeUser, err := GetUserByID(invites[invitationIdx-1]) // user who sent this invitation
	confirm := c.SendAndReceiveMsg("\nConfirm(Y/n): ", false, true)
	if c.Err != nil {
	}
	if strings.ToLower(confirm) == "y" || confirm == "" {
		err = c.User.CancelInvitation(inviteeUser)
		if err != nil {
			c.SendMessage(fmt.Sprintf("\nCancelling invitation to %s failed\n", inviteeUser.Email), true)
			reason := fmt.Sprintf("cancelling invitation from %s to %s failed: %s", c.User.Email, inviteeUser.Email, err)
			Logger().Println(reason)
			c.Err = errors.New(reason)
		} else {
			c.SendMessage(fmt.Sprintf("\nInvitation to %s successfully cancelled\n", inviteeUser.Email), true)
			reason := fmt.Sprintf("cancelling invitation from %s to %s succeeded", c.User.Email, inviteeUser.Email)
			Logger().Println(reason)
		}
	}
}

func (c *Client) SeeInactiveReceivedInvitations() {
	invites, err := c.User.GetSentInvitations()
	if err != nil {
		reason := fmt.Sprintf("error while fetching active sent invitations for user %s: %s", c.Email, err)
		Logger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	c.SendMessage("\n**** Active Sent Invitations ****\n", true)
	for idx, invite := range invites {
		c.SendMessage(fmt.Sprintf("%d - %s", idx+1, UserProfile(invite)), true)
	}
	userInput := c.SendAndReceiveMsg("\nChoose one to cancel(\"b to go back\"): ", false, false)
	if c.Err != nil {
		reason := fmt.Sprintf("error while seeing active user invitation sent from client %s: %s", (*c.Conn).RemoteAddr(), err)
		Logger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	if strings.ToLower(userInput) == "b" {
		return
	}
	invitationIdx, err := strconv.Atoi(userInput)
	if err != nil || invitationIdx < 0 || invitationIdx > len(invites) {
		reason := fmt.Sprintf("invitation index input %s parsing failed from client %s: %s", userInput,
			(*c.Conn).RemoteAddr(), userInput)
		Logger().Println(reason)
		c.Err = errors.New(reason)
		c.SendMessage(fmt.Sprintf("Invalid choice: %s", userInput), true)
		return
	}
	// The user sees 1-based indexing, so reducing one from it
	inviteeUser, err := GetUserByID(invites[invitationIdx-1]) // user who sent this invitation
	confirm := c.SendAndReceiveMsg("\nConfirm(Y/n): ", false, true)
	if c.Err != nil {
	}
	if strings.ToLower(confirm) == "y" || confirm == "" {
		err = c.User.CancelInvitation(inviteeUser)
		if err != nil {
			c.SendMessage(fmt.Sprintf("\nCancelling invitation to %s failed\n", inviteeUser.Email), true)
			reason := fmt.Sprintf("cancelling invitation from %s to %s failed: %s", c.User.Email, inviteeUser.Email, err)
			Logger().Println(reason)
			c.Err = errors.New(reason)
		} else {
			c.SendMessage(fmt.Sprintf("\nInvitation to %s successfully cancelled\n", inviteeUser.Email), true)
			reason := fmt.Sprintf("cancelling invitation from %s to %s succeeded", c.User.Email, inviteeUser.Email)
			Logger().Println(reason)
		}
	}
}

func (c *Client) SeeInactiveSentInvitations() {

}

func (c *Client) ChangePassword() {
	var failureCount int
	for failureCount = 0; failureCount < 3; failureCount++ {
		currPassword := c.SendAndReceiveMsg("\nEnter your current password: ", false, false)
		if c.Err != nil {
			continue
		}
		if err := MatchHashAndPlainText(c.Password, currPassword); err != nil {
			c.Err = fmt.Errorf("user %s entered incorrect password: %s", c.Email, err)
			Logger().Println(c.Err)
			//if c.Err.Error() == PasswordMismatch {
			//	c.SendMessage(PasswordMismatch, true)
			//} else {
			//	c.SendMessage(ServerError, true)
			//}
			//c.Err = errors.New(reason)
			continue
		}
		break
	}
	if failureCount == 3 {
		return // user unable to enter current password
	}
	for failureCount = 0; failureCount < 3; failureCount++ {
		newPassword := c.SendAndReceiveMsg("\nEnter your new password: ", false, false)
		if c.Err != nil {
			continue
		}
		c.Err = ValidatePassword(newPassword)
		if c.Err != nil {
			c.SendMessage(ShortPassword, true)
			continue
		}
		//encryptedPassword := GetSHA512Encrypted(newPassword)
		//if err MatchHashAndPlainText(c.Password, newPassword) {
		//	reason := "New Password is same as current. Please select a different one."
		//	Logger().Println(reason)
		//	c.SendMessage(reason, true)
		//	c.Err = errors.New(reason)
		//	continue
		//}
		confirmNewPassword := c.SendAndReceiveMsg("\nConfirm your new password: ", false, false)
		if c.Err != nil {
			continue
		}
		if newPassword != confirmNewPassword {
			reason := "Passwords didn't match"
			Logger().Println(reason)
			c.SendMessage(reason, true)
			c.Err = errors.New(reason)
			continue
		}
		passwordHash := GenerateHash(newPassword)
		err := c.User.UpdatePassword(passwordHash)
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
	newFirstName := c.SendAndReceiveMsg("\nEnter your new first name(enter blank for skip): ", false, true)
	if c.Err != nil {
		//continue
		// add retry
	}
	if newFirstName == "" {
		reason := fmt.Sprintf("skipping first name change for user %s", c.Email)
		Logger().Println(reason)
	}

	newLastName := c.SendAndReceiveMsg("\nEnter your new last name(enter blank for skip): ", false, true)
	if c.Err != nil {
		//continue
		// add retry
	}
	if newLastName == "" {
		reason := fmt.Sprintf("skipping first name change for user %s", c.Email)
		Logger().Println(reason)
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
		Logger().Println(reason)
	}
}

// enable client to see his/her own profile in detail
func (c *Client) SeeSelfProfile() {

}

// allow a client to see other person's basic detail before sending invitation
func (c *Client) SeePublicProfile(email string) (user *User, err error) {
	user, err = GetUserByEmail(email)
	if err == mongo.ErrNoDocuments {
		c.SendMessage(fmt.Sprintf("\nNo user found with given email %s", email), true)
		if c.Err != nil {
			reason := fmt.Sprintf("error while sending no user found msg: %s", c.Err)
			Logger().Println(reason)
			return
		}
	}
	c.SendMessage(fmt.Sprintf("\nUser found => First Name: %s, LastName: %s, Email: %s", user.FirstName, user.LastName,
		user.Email), true)
	return
}

func ValidatePassword(password string) (err error) {
	if len(password) < 6 {
		reason := fmt.Sprintf("%s password: %s", ShortPassword, password)
		Logger().Println(reason)
		err = errors.New(reason)
	}
	return
}

func (c *Client) SeeOnlineFriends() {
	friends, err := c.User.SeeFriends()
	if err != nil {
		reason := fmt.Sprintf("error while fetching online friends for client %s: %s", (*c.Conn).RemoteAddr(), err)
		Logger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	c.SendMessage("\n****************** Online Friends List *****************\n", true)
	for idx, friend := range friends {
		c.SendMessage(fmt.Sprintf("%d - %s", idx+1, UserProfile(friend)), true)
	}
	// TODO: Pagination and search
	userInput := c.SendAndReceiveMsg("Enter a friend's index to start chat: ", false, false)
	friendIdx, err := strconv.Atoi(userInput)
	if err != nil {
		reason := fmt.Sprintf("error while parsing user input %s to start chat: %s", userInput, err)
		Logger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	// TODO: Check if valid friendIndex
	c.StarChat(friends[friendIdx-1])
}

func (c *Client) SeeFriends() {
	friends, err := c.User.SeeFriends()
	if err != nil {
		reason := fmt.Sprintf("error while fetching friends for client %s: %s", (*c.Conn).RemoteAddr(), err)
		Logger().Println(reason)
		c.Err = errors.New(reason)
		return
	}
	c.SendMessage("\n****************** Friends List *****************\n", true)
	for idx, friend := range friends {
		c.SendMessage(fmt.Sprintf("%d - %s", idx+1, UserProfile(friend)), true)
	}
	// TODO: Pagination and search
	for {
		userInput := c.SendAndReceiveMsg("\nEnter 'b' to go back: ", false, false)
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
			Logger().Println(reason)
			c.Err = errors.New(reason)
		}
	}
}

func (c *Client) PollIncomingMessages(other primitive.ObjectID, done chan bool, processed time.Time) {
	otherUser, _ := GetUserByID(other)
	pollTick := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-done: // clean exit
			return
		case <-pollTick.C:
			incomingMessages, _ := fetchIncomingMessages(processed, c.User.ID, other)
			for _, msg := range incomingMessages {
				processed = msg.Timestamp
				c.SendMessage(fmt.Sprintf("\n\n%s (%s): %s\n", otherUser.FirstName, msg.Timestamp, msg.Text),
					true)
				c.SendMessage(ChatPrompt, false)
			}
		}
	}
}
