package service

import (
	"errors"
	"fmt"
	"gibber/datastore"
	"gibber/log"
	"gibber/user"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
	"io"
	"strconv"
	"strings"
	"time"
)

// client captures all the necessary runtime information to enable a user to use gibber
type client struct {
	*user.User
	*Connection
}

// User response messages
const (
	welcomeMsg               = "Welcome to Gibber. Hope you have a lot to say today."
	emailPrompt              = "\nPlease enter your email to continue.\nEmail: "
	reenterEmailPrompt       = "Please re-enter your email.\nEmail: "
	passwordPrompt           = "\nYou are already a registered user. Please enter password to continue.\nPassword: "
	reenterPasswordPrompt    = "\nPlease re-enter your password.\nPassword: "
	newUserMsg               = "You are an unregistered user. Please register yourself by providing details.\n"
	firstNamePrompt          = "First Name: "
	lastNamePrompt           = "Last Name: "
	successfulLogin          = "\nLogged In Successfully. Last login: %s\n"
	failedLogin              = "Log In Failed"
	successfulRegistration   = "\nRegistered Successfully"
	failedRegistration       = "\nRegistration Failed"
	setPasswordPrompt        = "New Password: "
	confirmSetPasswordPrompt = "Confirm Password: "
	sendInvitationInfo       = "You can search other people uniquely by their email.\n"
	emailSearchPrompt        = "\nEmail(\"q\" to quit): "
	exitingMsg               = "exiting..."
	incomingMsgPollInterval  = 500 * time.Millisecond
	passwordMinLength        = 6
)

// Specific errors related to user flow
var (
	incorrectPassword          = errors.New("incorrect password")
	invalidEmail               = errors.New("invalid email")
	serverError                = errors.New("server processing error")
	emptyInput                 = errors.New("empty msg")
	shortPassword              = errors.New("password should be at 6 characters long")
	invalidInput               = errors.New("invalid msg")
	fetchReceivedInvitesFailed = errors.New("failed to fetch received invitations")
	fetchSentInvitesFailed     = errors.New("failed to fetch sent invitations")
	cancelInviteFailed         = errors.New("cancelling invite failed")
	fetchUserFailed            = errors.New("fetch user details failed")
	readEmailFailed            = errors.New("reading email failed")
	readPasswordFailed         = errors.New("reading password failed")
	passwordNotMatched         = errors.New("passwords not matched")
	internalError              = errors.New("internal error")
	logoutFailed               = errors.New("logout failed")
	fetchUserFriendsFailed     = errors.New("fetch user friends failed")
	insufficientLengthPassword = errors.New("password length is less than required")
	updateUserNameFailed       = errors.New("update user name failed")
	updateUserPasswordFailed   = errors.New("update user password failed")
)

// user menus for different scenarios
const (
	dashboardHeader = "********************** Welcome to Gibber ************************" +
		"\n\nPlease select one of the option from below."
	userMenu = "\n0 - Exit" +
		"\n1 - Start/Resume Chat" +
		"\n2 - See All Friends" +
		"\n3 - Send invitation" +
		"\n4 - See all invitations" +
		"\n5 - Change password" +
		"\n6 - Change Name" +
		"\n7 - See your profile" +
		"\n\nEnter a choice: "
	invitationMenu = "\n0 - Go back to previous menu" +
		"\n1 - Active Sent Invites" +
		"\n2 - Active Received Invites" +
		"\n3 - Inactive Sent Invites" +
		"\n4 - Inactive Received Invites" +
		"\n\nEnter a choice: "
)

// user options on the primary dashboard
const (
	exitChoice = iota
	startChatChoice
	seeAllFriends
	sendInvitationChoice
	seeInvitationChoice
	changePasswordChoice
	changeNameChoice
	seeProfileChoice
)

// user invites type (choices)
const (
	activeSentInvitesChoice = 1 + iota
	activeReceivedInvitesChoice
	inactiveSentInvitesChoice
	inactiveReceivedInvitesChoice
)

const chatPrompt = "Type message (press \"enter\" to send, \"q\" to quit): "

// showWelcomeMessage displays a welcome message to as user logs in
func (c *client) showWelcomeMessage() {
	c.sendMessage(welcomeMsg, true)
	if c.Err != nil {
		log.Logger().Printf("writing welcome message to client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
	}
}

// authenticate authenticate a client (user) by taking necessary details,
// email and password currently
func (c *client) authenticate() {
	c.promptForEmail()
	if c.Err != nil {
		return
	}
	exists := c.existingUser()
	if c.Err != nil {
		return
	}
	if exists {
		c.loginUser()
	} else {
		c.registerUser()
	}
}

// promptForEmail prompts for a newly connected client for email. It has 2 retries (total 3 times)
// before closing the client connection
func (c *client) promptForEmail() {
	for failureCount := 0; failureCount < 3; failureCount++ {
		if failureCount == 0 {
			c.Email = c.sendAndReceiveMsg(emailPrompt, false, false)
		} else {
			c.Email = c.sendAndReceiveMsg(reenterEmailPrompt, false, false)
		}
		if c.Err != nil {
			continue
		}
		c.Email = strings.ToLower(c.Email) // make email address case insensitive
		if !user.ValidUserEmail(c.Email) { // check for valid email - regex based
			log.Logger().Printf("invalid email %s", c.Email)
			c.sendMessage(invalidEmail.Error(), true)
			if c.Err != nil {
				log.Logger().Printf("sending invalid email msg to client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
				return
			}
			c.Err = invalidEmail
			continue
		}
		return // successfully read valid email from user
	}
	c.exitClient()
	c.Err = readEmailFailed
}

// existingUser checks whether the user already exists in the system, based on the entered email
func (c *client) existingUser() (exists bool) {
	_, c.Err = user.GetUserByEmail(c.Email) // if user not exists, it will throw an error
	if c.Err == mongo.ErrNoDocuments {
		c.Err = nil // resetting the error
		return
	}
	if c.Err != nil { // some other error occurred
		log.Logger().Printf("existing user check for client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
		return
	}
	exists = true
	return
}

// loginUser facilitate the user login. It has 3 retry attempts for incorrect credentials before exiting
func (c *client) loginUser() {
	for failureCount := 0; failureCount < 3; failureCount++ {
		if failureCount == 0 {
			c.sendMessage(passwordPrompt, false)
		} else {
			c.sendMessage(reenterPasswordPrompt, false)
		}
		if c.Err != nil {
			log.Logger().Printf("user password prompt failed: %s", c.Err)
			continue
		}
		password := c.readMessage()
		if c.Err != nil {
			log.Logger().Printf("reading user password failed: %s", c.Err)
			continue
		}
		var lastLogin string
		lastLogin, c.Err = c.User.LoginUser(password)
		if c.Err != nil {
			log.Logger().Printf("user %s authentication failed: %s", c.Email, c.Err)
			if c.Err == incorrectPassword {
				c.sendMessage(failedLogin+": "+incorrectPassword.Error(), true)
			} else {
				c.sendMessage(failedLogin+": "+serverError.Error(), true)
			}
			continue
		}
		log.Logger().Printf("user %s successfully logged in", c.Email)
		c.sendMessage(fmt.Sprintf(successfulLogin, lastLogin), true)
		if c.Err != nil {
			log.Logger().Printf("successful login msg failed to client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
		}
		c.sendMessage(dashboardHeader, true)
		if c.Err != nil {
			log.Logger().Printf("dashboard header msg failed to send to client %s: %s", (*c.Conn).RemoteAddr(),
				c.Err)
		}
		return
	}
	c.exitClient()
	c.Err = readPasswordFailed
}

// registerUser registers a new user when a new email is entered
func (c *client) registerUser() {
	c.sendMessage(newUserMsg, true)
	if c.Err != nil {
		log.Logger().Printf("new user message sending failed: %s", c.Err)
		return
	}

	firstName := c.sendAndReceiveMsg(firstNamePrompt, false, false)
	if c.Err != nil {
		log.Logger().Printf("reading user password failed: %s", c.Err)
		return
	}
	c.FirstName = firstName

	lastName := c.sendAndReceiveMsg(lastNamePrompt, false, false)
	if c.Err != nil {
		log.Logger().Printf("reading user last name failed: %s", c.Err)
		return
	}
	c.LastName = lastName

	password := c.sendAndReceiveMsg(setPasswordPrompt, false, false)
	if c.Err != nil {
		log.Logger().Printf("reading user new password failed: %s", c.Err)
		return
	}
	c.Err = validatePassword(password)
	if c.Err != nil {
		c.sendMessage(shortPassword.Error(), true)
		return
	}

	confPassword := c.sendAndReceiveMsg(confirmSetPasswordPrompt, false, false)
	if c.Err != nil {
		log.Logger().Printf("reading user confirm password failed: %s", c.Err)
		return
	} else if password != confPassword {
		log.Logger().Print(passwordNotMatched.Error())
		c.Err = passwordNotMatched
		return
	}
	c.Password = password

	_, c.Err = user.CreateUser(c.User)
	if c.Err != nil {
		log.Logger().Printf("user %s registration failed: %s", c.Email, c.Err)
		c.sendMessage(failedRegistration, true)
		return
	}

	log.Logger().Printf("user %s successfully regsistered", c.User)
	c.sendMessage(successfulRegistration, true)
	if c.Err != nil {
		log.Logger().Printf("successful registration msg failed to client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
	}
}

// sendAndReceiveMsg combines the sending and receiving of the message from user.
// Used for prompting user for something, and gets the entered input
func (c *client) sendAndReceiveMsg(msgToSend string, newline, emptyInputValid bool) (msgRecvd string) {
	c.sendMessage(msgToSend, newline)
	if c.Err != nil {
		return
	}
	msgRecvd = c.readMessage()
	if c.Err != nil {
		log.Logger().Printf("reading failed from client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
		return
	}
	if !emptyInputValid && msgRecvd == "" {
		log.Logger().Printf("empty string received from client %s: %s", (*c.Conn).RemoteAddr(), c.Err)
		c.sendMessage(emptyInput.Error(), true)
		if c.Err != nil {
			log.Logger().Printf("sending empty msg msg to client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
		}
	}
	return
}

// userDashboard shows the user dashboard, and facilitate the user interaction with the service
func (c *client) userDashboard() {
	exit := false
	var userInput string
	for !exit {
		userInput = c.showLandingPage()
		choice, err := strconv.Atoi(userInput)
		if c.Err == io.EOF { // connection is closed
			log.Logger().Printf("connection closed from %s", (*c.Conn).RemoteAddr())
			break
		} else if err != nil {
			c.sendMessage(invalidInput.Error(), true)
			continue
		}
		switch choice {
		case exitChoice:
			c.exitClient()
			exit = true
		case startChatChoice:
			c.seeOnlineFriends()
		case seeAllFriends:
			c.seeFriends()
		case sendInvitationChoice:
			c.sendInvitation()
		case seeInvitationChoice:
			c.seeInvitation()
		case changePasswordChoice:
			c.changePassword()
		case changeNameChoice:
			c.changeName()
		case seeProfileChoice:
			c.seePersonalProfile()
		default:
			c.sendMessage(invalidInput.Error(), true)
			continue
		}
	}
}

// showLandingPage displays the landing page for newly connected client
func (c *client) showLandingPage() string {
	return c.sendAndReceiveMsg(userMenu, false, false)
}

// startChat initiates/resumes a chat b/w the current user and the given user
func (c *client) starChat(friendID primitive.ObjectID) {
	content, timestamp := c.User.GetChat(friendID)
	c.sendMessage(content, true)
	var input string
	done := make(chan bool)
	go c.pollIncomingMessages(friendID, done, timestamp)
	for {
		c.sendMessage(chatPrompt, false)
		input = c.readMessage()
		if input == "" {
			c.sendMessage("Empty message can't be sent!!!", true)
			continue
		}
		if strings.ToLower(input) == "q" {
			done <- true // kill the incoming message listener
			break
		}
		err := user.SendMessage(c.User.ID, friendID, input, datastore.MongoConn().Collection(datastore.ChatCollection))
		if err != nil {
			log.Logger().Print(err)
		}
		c.sendMessage(fmt.Sprintf("\bYou: %s\n", input), true)
	}
}

// sendInvitation sends the invitation to a user
func (c *client) sendInvitation() {
	c.sendMessage(sendInvitationInfo, true)
	if c.Err != nil {
		log.Logger().Printf("error sending invitation prompt to user %s: %s", c.User.Email, c.Err)
		return
	}
	for {
		email := c.sendAndReceiveMsg(emailSearchPrompt, false, false)
		if c.Err != nil {
			continue
		}
		email = strings.ToLower(email)
		if email == "q" {
			break
		}
		user, err := c.seePublicProfile(email)
		if err == mongo.ErrNoDocuments { // user not found
			continue
		}
		c.sendMessage(fmt.Sprintf("Send invite to %s", email), false)
		confirm := c.sendAndReceiveMsg("Confirm? (Y/n): ", false, true)
		if c.Err != nil {
			log.Logger().Println(err)
			return
		}
		if strings.ToLower(confirm) == "y" || confirm == "" {
			err = c.User.SendInvitation(user)
			if err == nil {
				successMsg := fmt.Sprintf("\nInvitation sent successfully to %s %s (%s)", user.FirstName,
					user.LastName, user.Email)
				c.sendMessage(successMsg, true)
			}
		}
	}
}

// seeInvitation enables user to see all different kind of invitations (active/inactive, sent/received)
func (c *client) seeInvitation() {
	for {
		userInput := c.sendAndReceiveMsg(invitationMenu, false, false)
		if c.Err != nil {
			continue
		}
		choice, err := strconv.Atoi(userInput)
		if err != nil {
			c.sendMessage(invalidInput.Error(), true)
			continue
		}
		switch choice {
		case exitChoice:
			return
		case activeSentInvitesChoice:
			c.seeActiveSentInvitations()
		case activeReceivedInvitesChoice:
			c.seeActiveReceivedInvitations()
		case inactiveSentInvitesChoice:
			c.seeInactiveSentInvitations()
		case inactiveReceivedInvitesChoice:
			c.seeInactiveReceivedInvitations()
		default:
			c.sendMessage(invalidInput.Error(), true)
			continue
		}
	}

}

// seeActiveReceivedInvitations displays the invitations received from other users, yet to be acted upon
func (c *client) seeActiveReceivedInvitations() {
	invites, err := c.User.GetReceivedInvitations()
	if err != nil {
		log.Logger().Printf("error fetching active received invitations for user %s: %s", c.Email, err)
		c.Err = fetchReceivedInvitesFailed
		return
	}
	c.sendMessage("\n**** Active Received Invitations ****\n", true)
	for idx, invite := range invites {
		userProfile, _ := user.UserProfile(invite)
		c.sendMessage(fmt.Sprintf("%d - %s", idx+1, userProfile), true)
	}
	userInput := c.sendAndReceiveMsg("\nChoose one to accept or reject(\"b to go back\"): ", false,
		false)
	if c.Err != nil {
		log.Logger().Printf("error receiving user invitation msg from client %s: %s", (*c.Conn).RemoteAddr(), err)
		return
	}
	if strings.ToLower(userInput) == "b" {
		return
	}
	invitationIdx, err := strconv.Atoi(userInput)
	if err != nil || invitationIdx < 0 || invitationIdx > len(invites) {
		log.Logger().Printf("invitation index msg %s parsing failed from client %s: %s", userInput,
			(*c.Conn).RemoteAddr(), userInput)
		c.Err = invalidInput
		c.sendMessage(fmt.Sprintf("Invalid choice: %s", userInput), true)
		return
	}
	// The user sees 1-based indexing, so reducing one from it
	inviteeUser, err := user.GetUserByID(invites[invitationIdx-1]) // user who sent this invitation
	if err != nil {
		log.Logger().Printf("fetching invitee user %s details failed from client %s: %s", invites[invitationIdx],
			(*c.Conn).RemoteAddr(), userInput)
		c.Err = internalError
		c.sendMessage("Internal error. Try again", true)
		c.seeActiveReceivedInvitations()
		return
	}
	c.sendMessage("\n===== Invitation Details =====\n", true)
	c.sendMessage(fmt.Sprintf("Name: %s %s", inviteeUser.FirstName, inviteeUser.LastName), true)
	c.sendMessage(fmt.Sprintf("Email: %s", inviteeUser.Email), true)
	confirm := c.sendAndReceiveMsg("\nConfirm(Y/n): ", false, true)
	if c.Err != nil {
		return
	}
	if strings.ToLower(confirm) == "y" || confirm == "" {
		err = c.User.AddFriend(inviteeUser.ID)
		if err != nil {
			c.sendMessage(fmt.Sprintf("\nAdding %s as friend failed\n", inviteeUser.Email), true)
			log.Logger().Printf("adding %s as friend to %s failed: %s", c.User.Email, inviteeUser.Email, err)
			c.Err = internalError
		} else {
			successMsg := fmt.Sprintf("\nAdded %s as friend successfully\n",
				inviteeUser.FirstName+" "+inviteeUser.LastName)
			c.sendMessage(successMsg, true)
			log.Logger().Print(successMsg)
		}
	}
}

// seeActiveSentInvitations displays the invitations sent to users, yet to be acted upon
func (c *client) seeActiveSentInvitations() {
	invites, err := c.User.GetSentInvitations()
	if err != nil {
		log.Logger().Printf("error while fetching active sent invitations for user %s: %s", c.Email, err)
		c.Err = fetchSentInvitesFailed
		return
	}
	c.sendMessage("\n**** Active Sent Invitations ****\n", true)
	for idx, invite := range invites {
		userProfile, _ := user.UserProfile(invite)
		c.sendMessage(fmt.Sprintf("%d - %s", idx+1, userProfile), true)
	}
	userInput := c.sendAndReceiveMsg("\nChoose one to cancel(\"b to go back\"): ", false, false)
	if c.Err != nil {
		log.Logger().Printf("error while seeing active user invitation sent from client %s: %s", (*c.Conn).RemoteAddr(), err)
		return
	}
	if strings.ToLower(userInput) == "b" {
		return
	}
	invitationIdx, err := strconv.Atoi(userInput)
	if err != nil || invitationIdx < 0 || invitationIdx > len(invites) {
		log.Logger().Printf("invitation index msg %s parsing failed from client %s: %s", userInput,
			(*c.Conn).RemoteAddr(), userInput)
		c.Err = invalidInput
		c.sendMessage(fmt.Sprintf("Invalid choice: %s", userInput), true)
		return
	}

	// The user sees 1-based indexing, so reducing one from it
	inviteeUser, err := user.GetUserByID(invites[invitationIdx-1]) // user who sent this invitation
	if err != nil {
		log.Logger().Println(err)
		return
	}

	confirm := c.sendAndReceiveMsg("\nConfirm(Y/n): ", false, true)
	if c.Err != nil {
		log.Logger().Printf("canceling invitation failed: %s", c.Err)
		c.sendMessage(fmt.Sprintf("Invalid choice: %s", userInput), true)
		return
	}
	if strings.ToLower(confirm) == "y" || confirm == "" {
		err = c.User.CancelInvitation(inviteeUser)
		if err != nil {
			c.sendMessage(fmt.Sprintf("\nCancelling invitation to %s failed\n", inviteeUser.Email), true)
			log.Logger().Printf("cancelling invitation from %s to %s failed: %s", c.User.Email, inviteeUser.Email, err)
			c.Err = cancelInviteFailed
		} else {
			c.sendMessage(fmt.Sprintf("\nInvitation to %s successfully cancelled\n", inviteeUser.Email), true)
			log.Logger().Printf("cancelling invitation from %s to %s succeeded", c.User.Email, inviteeUser.Email)
		}
	}
}

// seeInactiveReceivedInvitations displays the list of inactive invitations received from other users
func (c *client) seeInactiveReceivedInvitations() {
	invites, err := c.User.GetSentInvitations()
	if err != nil {
		log.Logger().Printf("error while fetching active sent invitations for user %s: %s", c.Email, err)
		c.Err = fetchSentInvitesFailed
		return
	}
	c.sendMessage("\n**** Active Sent Invitations ****\n", true)
	for idx, invite := range invites {
		userProfile, _ := user.UserProfile(invite)
		c.sendMessage(fmt.Sprintf("%d - %s", idx+1, userProfile), true)
	}
	userInput := c.sendAndReceiveMsg("\nChoose one to cancel(\"b to go back\"): ", false, false)
	if c.Err != nil {
		log.Logger().Printf("error while seeing active user invitation sent from client %s: %s", (*c.Conn).RemoteAddr(), err)
		c.Err = internalError
		return
	}
	if strings.ToLower(userInput) == "b" {
		return
	}
	invitationIdx, err := strconv.Atoi(userInput)
	if err != nil || invitationIdx < 0 || invitationIdx > len(invites) {
		log.Logger().Printf("invitation index msg %s parsing failed from client %s: %s", userInput,
			(*c.Conn).RemoteAddr(), userInput)
		c.Err = invalidInput
		c.sendMessage(fmt.Sprintf("Invalid choice: %s", userInput), true)
		return
	}
	// The user sees 1-based indexing, so reducing one from it
	inviteeUser, err := user.GetUserByID(invites[invitationIdx-1]) // user who sent this invitation
	if err != nil {
		log.Logger().Printf("error fetching user %s details: %s", invites[invitationIdx-1], err)
		c.Err = fetchUserFailed
		return
	}

	confirm := c.sendAndReceiveMsg("\nConfirm(Y/n): ", false, true)
	if c.Err != nil {
		log.Logger().Printf("error getting confirmation: %s", c.Err)
		c.Err = invalidInput
		return
	}
	if strings.ToLower(confirm) == "y" || confirm == "" {
		err = c.User.CancelInvitation(inviteeUser)
		if err != nil {
			c.sendMessage(fmt.Sprintf("\nCancelling invitation to %s failed\n", inviteeUser.Email), true)
			log.Logger().Printf("cancelling invitation from %s to %s failed: %s", c.User.Email, inviteeUser.Email, err)
			c.Err = cancelInviteFailed
		} else {
			c.sendMessage(fmt.Sprintf("\nInvitation to %s successfully cancelled\n", inviteeUser.Email), true)
			log.Logger().Printf("cancelling invitation from %s to %s succeeded", c.User.Email, inviteeUser.Email)
		}
	}
}

// seeInactiveSentInvitations displays the list of inactive invitations sent to other users
func (c *client) seeInactiveSentInvitations() {

}

// changePassword enables user to change his/her password
func (c *client) changePassword() {
	var failureCount int
	for failureCount = 0; failureCount < 3; failureCount++ {
		currPassword := c.sendAndReceiveMsg("\nEnter your current password: ", false, false)
		if c.Err != nil {
			continue
		}
		if err := bcrypt.CompareHashAndPassword([]byte(c.Password), []byte(currPassword)); err != nil {
			log.Logger().Printf("user %s entered incorrect password: %s", c.Email, err)
			c.Err = incorrectPassword
			continue
		}
		break
	}
	if failureCount == 3 {
		return // user unable to enter current password
	}
	for failureCount = 0; failureCount < 3; failureCount++ {
		newPassword := c.sendAndReceiveMsg("\nEnter your new password: ", false, false)
		if c.Err != nil {
			continue
		}
		c.Err = validatePassword(newPassword)
		if c.Err != nil {
			c.sendMessage(shortPassword.Error(), true)
			continue
		}
		confirmNewPassword := c.sendAndReceiveMsg("\nConfirm your new password: ", false, false)
		if c.Err != nil {
			continue
		}
		if newPassword != confirmNewPassword {
			log.Logger().Print(passwordNotMatched)
			c.sendMessage(passwordNotMatched.Error(), true)
			c.Err = passwordNotMatched
			continue
		}
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Logger().Println(err)
			c.Err = internalError
			return
		}
		err = c.User.UpdatePassword(string(passwordHash))
		if err != nil {
			c.Err = updateUserPasswordFailed
			c.sendMessage("Password update failed. Please try again.\n", true)
			return
		}
		c.sendMessage("Password successfully updated\n", true)
		return
	}
}

// changeName enables current user to change his/her name
func (c *client) changeName() {
	newFirstName := c.sendAndReceiveMsg("\nEnter your new first name(enter blank for skip): ", false, true)
	if c.Err != nil {
		log.Logger().Printf("error getting entered first name: %s", c.Err)
		return
	}
	if newFirstName == "" {
		log.Logger().Printf("skipping first name change for user %s", c.Email)
	}

	newLastName := c.sendAndReceiveMsg("\nEnter your new last name(enter blank for skip): ", false, true)
	if c.Err != nil {
		log.Logger().Printf("error getting entered last name: %s", c.Err)
		return
	}
	if newLastName == "" {
		log.Logger().Printf("skipping first name change for user %s", c.Email)
	}

	err := c.User.UpdateName(newFirstName, newLastName)
	if err != nil {
		c.Err = updateUserNameFailed
		c.sendMessage("Name update failed. Please try again.\n", true)
		return
	}

	c.sendMessage("Name successfully updated\n", true)
}

// seePersonalProfile displays the profile for the current user
func (c *client) seePersonalProfile() {
	details := "\n************ Profile ************ \n"
	details += fmt.Sprintf("\nFirst Name: %s\n", c.User.FirstName)
	details += fmt.Sprintf("Last Name: %s\n", c.User.LastName)
	details += fmt.Sprintf("Email: %s\n", c.User.Email)
	details += fmt.Sprintf("Last Login: %s\n", c.User.LastLogin)
	c.sendMessage(details, true)
}

// exitClient displays the exiting message to client
func (c *client) exitClient() {
	c.sendMessage(exitingMsg, true)
	if c.Err != nil {
		log.Logger().Printf("sending exit msg to client %s failed: %s", (*c.Conn).RemoteAddr(), c.Err)
	}
}

// seePublicProfile allow a client to see other person's basic detail before sending invitation
func (c *client) seePublicProfile(email string) (usr *user.User, err error) {
	usr, err = user.GetUserByEmail(email)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.sendMessage(fmt.Sprintf("\nNo usr found with given email %s", email), true)
		}
		log.Logger().Printf("error while sending no usr found msg: %s", c.Err)
		return
	}
	c.sendMessage(fmt.Sprintf("\nusr found => First Name: %s, LastName: %s, Email: %s", usr.FirstName, usr.LastName,
		usr.Email), true)
	return
}

// seeOnlineFriends displays the list of friends (already connected)
// who are currently logged in the service
func (c *client) seeOnlineFriends() {
	friends, err := c.User.SeeFriends()
	if err != nil {
		log.Logger().Printf("error while fetching online friends for client %s: %s", (*c.Conn).RemoteAddr(), err)
		c.Err = fetchUserFriendsFailed
		return
	}
	c.sendMessage("\n****************** Online Friends List *****************\n", true)
	for idx, friend := range friends {
		userProfile, _ := user.UserProfile(friend)
		c.sendMessage(fmt.Sprintf("%d - %s", idx+1, userProfile), true)
	}
	userInput := c.sendAndReceiveMsg("Enter a friend's index to start chat: ", false, false)
	friendIdx, err := strconv.Atoi(userInput)
	if err != nil {
		log.Logger().Printf("error while parsing user msg %s to start chat: %s", userInput, err)
		c.Err = invalidInput
		return
	}
	c.starChat(friends[friendIdx-1])
}

// seeFriends displays the list of friends (already connected) to current user
func (c *client) seeFriends() {
	friends, err := c.User.SeeFriends()
	if err != nil {
		log.Logger().Printf("error while fetching friends for client %s: %s", (*c.Conn).RemoteAddr(), err)
		c.Err = fetchUserFriendsFailed
		return
	}
	c.sendMessage("\n****************** Friends List *****************\n", true)
	for idx, friend := range friends {
		userProfile, _ := user.UserProfile(friend)
		c.sendMessage(fmt.Sprintf("%d - %s", idx+1, userProfile), true)
	}
	for {
		userInput := c.sendAndReceiveMsg("\nEnter 'b' to go back: ", false, false)
		if userInput == "b" {
			break
		}
		c.sendMessage(fmt.Sprintf("Invalid msg: %s", userInput), true)
	}
}

// logoutUser cleanly logs out the current user and free the resource
func (c *client) logoutUser() {
	if c.User.Email != "" {
		err := c.User.Logout()
		if err != nil {
			log.Logger().Printf("error while logging out client %s: %s", c.User.Email, err)
			c.Err = logoutFailed
		}
	}
}

// pollIncomingMessages checks for any new incoming message from a particular user,
// received after the given timestamp
func (c *client) pollIncomingMessages(other primitive.ObjectID, done chan bool, processed time.Time) {
	otherUser, err := user.GetUserByID(other)
	if err != nil {
		log.Logger().Printf("error fetching user %s details: %s", c.User.Email, err)
		c.Err = fetchUserFailed
		return
	}
	pollTick := time.NewTicker(incomingMsgPollInterval)
	for {
		select {
		case <-done: // clean exit
			return
		case <-pollTick.C:
			incomingMessages, _ := user.FetchIncomingMessages(processed, c.User.ID, other)
			for _, msg := range incomingMessages {
				processed = msg.Timestamp
				c.sendMessage(fmt.Sprintf("\n\n%s (%s): %s\n", otherUser.FirstName, msg.Timestamp, msg.Text),
					true)
				c.sendMessage(chatPrompt, false)
			}
		}
	}
}

// validatePassword checks whether the passowrd is an acceptable password or not
func validatePassword(password string) (err error) {
	if len(password) < passwordMinLength {
		log.Logger().Printf("%s password: %s", shortPassword, password)
		err = insufficientLengthPassword
	}
	return
}
