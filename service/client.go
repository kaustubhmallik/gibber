package service

import (
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"io"
	"strconv"
	"strings"
	"time"
)

// a client using the gibber app
type Client struct {
	*User
	*Connection
}

// User Response Messages
const (
	WelcomeMsg               = "Welcome to Gibber. Hope you have a lot to say today."
	EmailPrompt              = "\nPlease enter your email to continue.\nEmail: "
	ReenterEmailPrompt       = "Please re-enter your email.\nEmail: "
	PasswordPrompt           = "\nYou are already a registered user. Please enter password to continue.\nPassword: "
	ReenterPasswordPrompt    = "\nPlease re-enter your password.\nPassword: "
	NewUserMsg               = "You are an unregistered user. Please register yourself by providing details.\n"
	FirstNamePrompt          = "First Name: "
	LastNamePrompt           = "Last Name: "
	SuccessfulLogin          = "\nLogged In Successfully. Last login: %s\n"
	FailedLogin              = "Log In Failed"
	SuccessfulRegistration   = "\nRegistered Successfully"
	FailedRegistration       = "\nRegistration Failed"
	SetPasswordPrompt        = "New Password: "
	ConfirmSetPasswordPrompt = "Confirm Password: "
	SendInvitationInfo       = "You can search other people uniquely by their email.\n"
	EmailSearchPrompt        = "\nEmail(\"q\" to quit): "
	ExitingMsg               = "exiting..."
	IncomingMsgPollInterval  = 500 * time.Millisecond
	PasswordMinLength        = 6
)

// specific errors
var (
	IncorrectPassword          = errors.New("incorrect password")
	InvalidEmail               = errors.New("invalid email")
	ServerError                = errors.New("server processing error")
	EmptyInput                 = errors.New("empty msg")
	ShortPassword              = errors.New("password should be at 6 characters long")
	InvalidInput               = errors.New("invalid msg")
	FetchReceivedInvitesFailed = errors.New("failed to fetch received invitations")
	FetchSentInvitesFailed     = errors.New("failed to fetch sent invitations")
	CancelInviteFailed         = errors.New("cancelling invite failed")
	FetchUserFailed            = errors.New("fetch user details failed")
	ReadEmailFailed            = errors.New("reading email failed")
	ReadPasswordFailed         = errors.New("reading password failed")
	PasswordNotMatched         = errors.New("passwords not matched")
	InternalError              = errors.New("internal error")
	LogoutFailed               = errors.New("logout failed")
	FetchUserFriendsFailed     = errors.New("fetch user friends failed")
	InsufficientLengthPassword = errors.New("password length is less than required")
	UpdateUserNameFailed       = errors.New("update user name failed")
	UpdateUserPasswordFailed   = errors.New("update user password failed")
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
}

func (c *Client) PromptForEmail() {
	for failureCount := 0; failureCount < 3; failureCount++ {
		if failureCount == 0 {
			c.Email = c.SendAndReceiveMsg(EmailPrompt, false, false)
		} else {
			c.Email = c.SendAndReceiveMsg(ReenterEmailPrompt, false, false)
		}
		if c.Err != nil {
			continue
		}
		c.Email = strings.ToLower(c.Email) // make email address case insensitive
		if !ValidUserEmail(c.Email) {      // check for valid email - regex based
			Logger().Printf("invalid email %s", c.Email)
			c.SendMessage(InvalidEmail.Error(), true)
			if c.Err != nil {
				Logger().Printf("sending invalid email msg to client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
				return
			}
			c.Err = InvalidEmail
			continue
		}
		return // successfully read valid email from user
	}
	c.ExitClient()
	c.Err = ReadEmailFailed
}

func (c *Client) ExistingUser() (exists bool) {
	_, c.Err = GetUserByEmail(c.Email) // if user not exists, it will throw an error
	if c.Err == mongo.ErrNoDocuments {
		c.Err = nil // resetting the error
		return
	}
	if c.Err != nil { // some other error occurred
		Logger().Printf("existing user check for client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
		return
	}
	exists = true
	return
}

func (c *Client) LoginUser() {
	for failureCount := 0; failureCount < 3; failureCount++ {
		if failureCount == 0 {
			c.SendMessage(PasswordPrompt, false)
		} else {
			c.SendMessage(ReenterPasswordPrompt, false)
		}
		if c.Err != nil {
			Logger().Printf("user password prompt failed: %s", c.Err)
			continue
		}
		password := c.ReadMessage()
		if c.Err != nil {
			Logger().Printf("reading user password failed: %s", c.Err)
			continue
		}
		var lastLogin string
		lastLogin, c.Err = c.User.LoginUser(password)
		if c.Err != nil {
			Logger().Printf("user %s authentication failed: %s", c.Email, c.Err)
			if c.Err == IncorrectPassword {
				c.SendMessage(FailedLogin+": "+IncorrectPassword.Error(), true)
			} else {
				c.SendMessage(FailedLogin+": "+ServerError.Error(), true)
			}
			continue
		}
		Logger().Printf("user %s successfully logged in", c.Email)
		c.SendMessage(fmt.Sprintf(SuccessfulLogin, lastLogin), true)
		if c.Err != nil {
			Logger().Printf("successful login msg failed to client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
		}
		c.SendMessage(DashboardHeader, true)
		if c.Err != nil {
			Logger().Printf("dashboard header msg failed to send to client %s: %s", (*c.Conn).RemoteAddr(),
				c.Err)
		}
		return
	}
	c.ExitClient()
	c.Err = ReadPasswordFailed
}

func (c *Client) RegisterUser() {
	c.SendMessage(NewUserMsg, true)
	if c.Err != nil {
		Logger().Printf("new user message sending failed: %s", c.Err)
		return
	}

	firstName := c.SendAndReceiveMsg(FirstNamePrompt, false, false)
	if c.Err != nil {
		Logger().Printf("reading user password failed: %s", c.Err)
		return
	}
	c.FirstName = firstName

	lastName := c.SendAndReceiveMsg(LastNamePrompt, false, false)
	if c.Err != nil {
		Logger().Printf("reading user last name failed: %s", c.Err)
		return
	}
	c.LastName = lastName

	password := c.SendAndReceiveMsg(SetPasswordPrompt, false, false)
	if c.Err != nil {
		Logger().Printf("reading user new password failed: %s", c.Err)
		return
	}
	c.Err = ValidatePassword(password)
	if c.Err != nil {
		c.SendMessage(ShortPassword.Error(), true)
		return
	}

	confPassword := c.SendAndReceiveMsg(ConfirmSetPasswordPrompt, false, false)
	if c.Err != nil {
		Logger().Printf("reading user confirm password failed: %s", c.Err)
		return
	} else if password != confPassword {
		Logger().Print(PasswordNotMatched.Error())
		c.Err = PasswordNotMatched
		return
	}
	c.Password = password

	_, c.Err = CreateUser(c.User)
	if c.Err != nil {
		Logger().Printf("user %s registration failed: %s", c.Email, c.Err)
		c.SendMessage(FailedRegistration, true)
		return
	}

	Logger().Printf("user %s successfully regsistered", c.User)
	c.SendMessage(SuccessfulRegistration, true)
	if c.Err != nil {
		Logger().Printf("successful registration msg failed to client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
	}
}

func (c *Client) SendAndReceiveMsg(msgToSend string, newline, emptyInputValid bool) (msgRecvd string) {
	c.SendMessage(msgToSend, newline)
	if c.Err != nil {
		return
	}
	msgRecvd = c.ReadMessage()
	if c.Err != nil {
		Logger().Printf("reading failed from client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
		return
	}
	if !emptyInputValid && msgRecvd == "" {
		Logger().Printf("empty string received from client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
		c.SendMessage(EmptyInput.Error(), true)
		if c.Err != nil {
			Logger().Printf("sending empty msg msg to client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
		}
	}
	return
}

func (c *Client) UserDashboard() {
	exit := false
	var userInput string
	for !exit {
		userInput = c.ShowLandingPage()
		choice, err := strconv.Atoi(userInput)
		if c.Err == io.EOF { // connection is closed
			Logger().Printf("connection closed from %s", (*c.Conn).RemoteAddr())
			break
		} else if err != nil {
			c.SendMessage(InvalidInput.Error(), true)
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
			c.SeePersonalProfile()
		default:
			c.SendMessage(InvalidInput.Error(), true)
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
		c.SendMessage(fmt.Sprintf("\bYou: %s\n", input), true)
	}
}

func (c *Client) SendInvitation() {
	c.SendMessage(SendInvitationInfo, true)
	if c.Err != nil {
		Logger().Printf("error sending invitation prompt to user %s: %s", c.User.Email, c.Err)
		return
	}
	for {
		email := c.SendAndReceiveMsg(EmailSearchPrompt, false, false)
		if c.Err != nil {
			continue
		}
		email = strings.ToLower(email)
		if email == "q" {
			break
		}
		user, err := c.SeePublicProfile(email)
		if err == mongo.ErrNoDocuments { // user not found
			continue
		}
		c.SendMessage(fmt.Sprintf("Send invite to %s", email), false)
		confirm := c.SendAndReceiveMsg("Confirm? (Y/n): ", false, true)
		if c.Err != nil {
			Logger().Println(err)
			return
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
			c.SendMessage(InvalidInput.Error(), true)
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
			c.SendMessage(InvalidInput.Error(), true)
			continue
		}
	}

}

func (c *Client) SeeActiveReceivedInvitations() {
	invites, err := c.User.GetReceivedInvitations()
	if err != nil {
		Logger().Printf("error fetching active received invitations for user %s: %s", c.Email, err)
		c.Err = FetchReceivedInvitesFailed
		return
	}
	c.SendMessage("\n**** Active Received Invitations ****\n", true)
	for idx, invite := range invites {
		userProfile, _ := UserProfile(invite)
		c.SendMessage(fmt.Sprintf("%d - %s", idx+1, userProfile), true)
	}
	userInput := c.SendAndReceiveMsg("\nChoose one to accept or reject(\"b to go back\"): ", false,
		false)
	if c.Err != nil {
		Logger().Printf("error receiving user invitation msg from client %s: %s", (*c.Conn).RemoteAddr(), err)
		return
	}
	if strings.ToLower(userInput) == "b" {
		return
	}
	invitationIdx, err := strconv.Atoi(userInput)
	if err != nil || invitationIdx < 0 || invitationIdx > len(invites) {
		Logger().Printf("invitation index msg %s parsing failed from client %s: %s", userInput,
			(*c.Conn).RemoteAddr(), userInput)
		c.Err = InvalidInput
		c.SendMessage(fmt.Sprintf("Invalid choice: %s", userInput), true)
		return
	}
	// The user sees 1-based indexing, so reducing one from it
	inviteeUser, err := GetUserByID(invites[invitationIdx-1]) // user who sent this invitation
	if err != nil {
		Logger().Printf("fetching invitee user %s details failed from client %s: %s", invites[invitationIdx],
			(*c.Conn).RemoteAddr(), userInput)
		c.Err = InternalError
		c.SendMessage("Internal error. Try again", true)
		c.SeeActiveReceivedInvitations()
		return
	}
	c.SendMessage("\n===== Invitation Details =====\n", true)
	c.SendMessage(fmt.Sprintf("Name: %s %s", inviteeUser.FirstName, inviteeUser.LastName), true)
	c.SendMessage(fmt.Sprintf("Email: %s", inviteeUser.Email), true)
	confirm := c.SendAndReceiveMsg("\nConfirm(Y/n): ", false, true)
	if c.Err != nil {
		return
	}
	if strings.ToLower(confirm) == "y" || confirm == "" {
		err = c.User.AddFriend(inviteeUser.ID)
		if err != nil {
			c.SendMessage(fmt.Sprintf("\nAdding %s as friend failed\n", inviteeUser.Email), true)
			Logger().Printf("adding %s as friend to %s failed: %s", c.User.Email, inviteeUser.Email, err)
			c.Err = InternalError
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
		Logger().Printf("error while fetching active sent invitations for user %s: %s", c.Email, err)
		c.Err = FetchSentInvitesFailed
		return
	}
	c.SendMessage("\n**** Active Sent Invitations ****\n", true)
	for idx, invite := range invites {
		userProfile, _ := UserProfile(invite)
		c.SendMessage(fmt.Sprintf("%d - %s", idx+1, userProfile), true)
	}
	userInput := c.SendAndReceiveMsg("\nChoose one to cancel(\"b to go back\"): ", false, false)
	if c.Err != nil {
		Logger().Printf("error while seeing active user invitation sent from client %s: %s", (*c.Conn).RemoteAddr(), err)
		return
	}
	if strings.ToLower(userInput) == "b" {
		return
	}
	invitationIdx, err := strconv.Atoi(userInput)
	if err != nil || invitationIdx < 0 || invitationIdx > len(invites) {
		Logger().Printf("invitation index msg %s parsing failed from client %s: %s", userInput,
			(*c.Conn).RemoteAddr(), userInput)
		c.Err = InvalidInput
		c.SendMessage(fmt.Sprintf("Invalid choice: %s", userInput), true)
		return
	}

	// The user sees 1-based indexing, so reducing one from it
	inviteeUser, err := GetUserByID(invites[invitationIdx-1]) // user who sent this invitation
	if err != nil {
		Logger().Println(err)
		return
	}

	confirm := c.SendAndReceiveMsg("\nConfirm(Y/n): ", false, true)
	if c.Err != nil {
		Logger().Printf("canceling invitation failed: %s", c.Err)
		c.SendMessage(fmt.Sprintf("Invalid choice: %s", userInput), true)
		return
	}
	if strings.ToLower(confirm) == "y" || confirm == "" {
		err = c.User.CancelInvitation(inviteeUser)
		if err != nil {
			c.SendMessage(fmt.Sprintf("\nCancelling invitation to %s failed\n", inviteeUser.Email), true)
			Logger().Printf("cancelling invitation from %s to %s failed: %s", c.User.Email, inviteeUser.Email, err)
			c.Err = CancelInviteFailed
		} else {
			c.SendMessage(fmt.Sprintf("\nInvitation to %s successfully cancelled\n", inviteeUser.Email), true)
			Logger().Printf("cancelling invitation from %s to %s succeeded", c.User.Email, inviteeUser.Email)
		}
	}
}

func (c *Client) SeeInactiveReceivedInvitations() {
	invites, err := c.User.GetSentInvitations()
	if err != nil {
		Logger().Printf("error while fetching active sent invitations for user %s: %s", c.Email, err)
		c.Err = FetchSentInvitesFailed
		return
	}
	c.SendMessage("\n**** Active Sent Invitations ****\n", true)
	for idx, invite := range invites {
		userProfile, _ := UserProfile(invite)
		c.SendMessage(fmt.Sprintf("%d - %s", idx+1, userProfile), true)
	}
	userInput := c.SendAndReceiveMsg("\nChoose one to cancel(\"b to go back\"): ", false, false)
	if c.Err != nil {
		Logger().Printf("error while seeing active user invitation sent from client %s: %s", (*c.Conn).RemoteAddr(), err)
		c.Err = InternalError
		return
	}
	if strings.ToLower(userInput) == "b" {
		return
	}
	invitationIdx, err := strconv.Atoi(userInput)
	if err != nil || invitationIdx < 0 || invitationIdx > len(invites) {
		Logger().Printf("invitation index msg %s parsing failed from client %s: %s", userInput,
			(*c.Conn).RemoteAddr(), userInput)
		c.Err = InvalidInput
		c.SendMessage(fmt.Sprintf("Invalid choice: %s", userInput), true)
		return
	}
	// The user sees 1-based indexing, so reducing one from it
	inviteeUser, err := GetUserByID(invites[invitationIdx-1]) // user who sent this invitation
	if err != nil {
		Logger().Printf("error fetching user %s details: %s", invites[invitationIdx-1], err)
		c.Err = FetchUserFailed
		return
	}

	confirm := c.SendAndReceiveMsg("\nConfirm(Y/n): ", false, true)
	if c.Err != nil {
		Logger().Printf("error getting confirmation: %s", c.Err)
		c.Err = InvalidInput
		return
	}
	if strings.ToLower(confirm) == "y" || confirm == "" {
		err = c.User.CancelInvitation(inviteeUser)
		if err != nil {
			c.SendMessage(fmt.Sprintf("\nCancelling invitation to %s failed\n", inviteeUser.Email), true)
			Logger().Printf("cancelling invitation from %s to %s failed: %s", c.User.Email, inviteeUser.Email, err)
			c.Err = CancelInviteFailed
		} else {
			c.SendMessage(fmt.Sprintf("\nInvitation to %s successfully cancelled\n", inviteeUser.Email), true)
			Logger().Printf("cancelling invitation from %s to %s succeeded", c.User.Email, inviteeUser.Email)
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
			Logger().Printf("user %s entered incorrect password: %s", c.Email, err)
			c.Err = IncorrectPassword
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
			c.SendMessage(ShortPassword.Error(), true)
			continue
		}
		confirmNewPassword := c.SendAndReceiveMsg("\nConfirm your new password: ", false, false)
		if c.Err != nil {
			continue
		}
		if newPassword != confirmNewPassword {
			Logger().Print(PasswordNotMatched)
			c.SendMessage(PasswordNotMatched.Error(), true)
			c.Err = PasswordNotMatched
			continue
		}
		passwordHash, err := GenerateHash(newPassword)
		if err != nil {
			Logger().Println(err)
			c.Err = InternalError
			return
		}
		err = c.User.UpdatePassword(passwordHash)
		if err != nil {
			c.Err = UpdateUserPasswordFailed
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
		Logger().Printf("error getting entered first name: %s", c.Err)
		return
	}
	if newFirstName == "" {
		Logger().Printf("skipping first name change for user %s", c.Email)
	}

	newLastName := c.SendAndReceiveMsg("\nEnter your new last name(enter blank for skip): ", false, true)
	if c.Err != nil {
		Logger().Printf("error getting entered last name: %s", c.Err)
		return
	}
	if newLastName == "" {
		Logger().Printf("skipping first name change for user %s", c.Email)
	}

	err := c.User.UpdateName(newFirstName, newLastName)
	if err != nil {
		c.Err = UpdateUserNameFailed
		c.SendMessage("Name update failed. Please try again.\n", true)
		return
	}

	c.SendMessage("Name successfully updated\n", true)
}

func (c *Client) SeePersonalProfile() {
	details := "\n************ Profile ************ \n"
	details += fmt.Sprintf("\nFirst Name: %s\n", c.User.FirstName)
	details += fmt.Sprintf("Last Name: %s\n", c.User.LastName)
	details += fmt.Sprintf("Email: %s\n", c.User.Email)
	details += fmt.Sprintf("Last Login: %s\n", c.User.LastLogin)
	c.SendMessage(details, true)
}

func (c *Client) ExitClient() {
	c.SendMessage(ExitingMsg, true)
	if c.Err != nil {
		Logger().Printf("sending exit msg to client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
	}
}

// enable client to see his/her own profile in detail
func (c *Client) SeeSelfProfile() {

}

// allow a client to see other person's basic detail before sending invitation
func (c *Client) SeePublicProfile(email string) (user *User, err error) {
	user, err = GetUserByEmail(email)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.SendMessage(fmt.Sprintf("\nNo user found with given email %s", email), true)
		}
		Logger().Printf("error while sending no user found msg: %s", c.Err)
		return
	}
	c.SendMessage(fmt.Sprintf("\nuser found => First Name: %s, LastName: %s, Email: %s", user.FirstName, user.LastName,
		user.Email), true)
	return
}

func ValidatePassword(password string) (err error) {
	if len(password) < PasswordMinLength {
		Logger().Printf("%s password: %s", ShortPassword, password)
		err = InsufficientLengthPassword
	}
	return
}

func (c *Client) SeeOnlineFriends() {
	friends, err := c.User.SeeFriends()
	if err != nil {
		Logger().Printf("error while fetching online friends for client %s: %s", (*c.Conn).RemoteAddr(), err)
		c.Err = FetchUserFriendsFailed
		return
	}
	c.SendMessage("\n****************** Online Friends List *****************\n", true)
	for idx, friend := range friends {
		userProfile, _ := UserProfile(friend)
		c.SendMessage(fmt.Sprintf("%d - %s", idx+1, userProfile), true)
	}
	userInput := c.SendAndReceiveMsg("Enter a friend's index to start chat: ", false, false)
	friendIdx, err := strconv.Atoi(userInput)
	if err != nil {
		Logger().Printf("error while parsing user msg %s to start chat: %s", userInput, err)
		c.Err = InvalidInput
		return
	}
	c.StarChat(friends[friendIdx-1])
}

func (c *Client) SeeFriends() {
	friends, err := c.User.SeeFriends()
	if err != nil {
		Logger().Printf("error while fetching friends for client %s: %s", (*c.Conn).RemoteAddr(), err)
		c.Err = FetchUserFriendsFailed
		return
	}
	c.SendMessage("\n****************** Friends List *****************\n", true)
	for idx, friend := range friends {
		userProfile, _ := UserProfile(friend)
		c.SendMessage(fmt.Sprintf("%d - %s", idx+1, userProfile), true)
	}
	for {
		userInput := c.SendAndReceiveMsg("\nEnter 'b' to go back: ", false, false)
		if userInput == "b" {
			break
		}
		c.SendMessage(fmt.Sprintf("Invalid msg: %s", userInput), true)
	}
}

func (c *Client) LogoutUser() {
	if c.User.Email != "" {
		err := c.User.Logout()
		if err != nil {
			Logger().Printf("error while logging out client %s: %s", c.User.Email, err)
			c.Err = LogoutFailed
		}
	}
}

func (c *Client) PollIncomingMessages(other primitive.ObjectID, done chan bool, processed time.Time) {
	otherUser, err := GetUserByID(other)
	if err != nil {
		Logger().Printf("error fetching user %s details: %s", c.User.Email, err)
		c.Err = FetchUserFailed
		return
	}
	pollTick := time.NewTicker(IncomingMsgPollInterval)
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
