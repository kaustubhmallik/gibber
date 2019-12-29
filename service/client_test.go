package service

// Testing the functionality as a client i.e. from client-end

import (
	"bufio"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net"
	"testing"
)

var scanner *bufio.Scanner
var writer *bufio.Writer

func init() {
	go func() {
		_ = StartServer("localhost", "44510")
	}()

	//connect to server
	conn, err := net.Dial("tcp", "localhost:44510")
	if err != nil || conn == nil {
		log.Fatal("unable to connect to tcp server")
	}

	scanner = bufio.NewScanner(conn)
	writer = bufio.NewWriter(conn)
}

func TestClient(t *testing.T) {
	password := "password"
	hashPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := &User{
		ID:        primitive.NewObjectID(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john" + RandomString(20) + "@doe.com",
		Password:  string(hashPassword),
	}

	userID, err := CreateUser(user)
	assert.NoError(t, err, "user creation failed")
	user.ID = userID.(primitive.ObjectID)

	for !scanner.Scan() {
		t.Log(scanner.Text())
	}
	// correct email
	_, err = writer.WriteString(user.Email + "\n")
	assert.NoError(t, err, "writing email failed")
	err = writer.Flush()
	assert.NoError(t, err, "writing email failed")

	for !scanner.Scan() {
		t.Log(scanner.Text())
	}
	// wrong password
	_, err = writer.WriteString("invalid password" + "\n")
	assert.NoError(t, err, "writing password failed")
	err = writer.Flush()
	assert.NoError(t, err, "writing password failed")

	for !scanner.Scan() {
		t.Log(scanner.Text())
	}
	// correct password
	_, err = writer.WriteString(password + "\n")
	assert.NoError(t, err, "writing password failed")
	err = writer.Flush()
	assert.NoError(t, err, "writing password failed")

	// login completed
	_, err = writer.WriteString("0" + "\n")
	assert.NoError(t, err, "writing password failed")

	// connection closed from server as exit option selected
}

func TestClient_PromptForEmail(t *testing.T) {
}

func TestClient_ExistingUser(t *testing.T) {

}

func TestClient_LoginUser(t *testing.T) {
}

func TestClient_RegisterUser(t *testing.T) {

}

func TestClient_SendAndReceiveMsg(t *testing.T) {

}

func TestClient_UserDashboard(t *testing.T) {

}

func TestClient_ShowLandingPage(t *testing.T) {

}

func TestClient_StarChat(t *testing.T) {

}

func TestClient_SendInvitation(t *testing.T) {

}

func TestClient_SeeInvitation(t *testing.T) {

}

func TestClient_SeeActiveReceivedInvitations(t *testing.T) {

}

func TestClient_SeeActiveSentInvitations(t *testing.T) {

}

func TestClient_SeeInactiveReceivedInvitations(t *testing.T) {

}

func TestClient_SeeInactiveSentInvitations(t *testing.T) {

}

func TestClient_ChangePassword(t *testing.T) {

}

func TestClient_ChangeName(t *testing.T) {

}

func TestClient_SeePersonalProfile(t *testing.T) {

}

func TestClient_SeeSelfProfile(t *testing.T) {

}

func TestClient_SeePublicProfile(t *testing.T) {

}

func TestValidatePassword(t *testing.T) {

}

func TestClient_SeeFriends(t *testing.T) {

}

func TestClient_SeeOnlineFriends(t *testing.T) {

}

func TestClient_LogoutUser(t *testing.T) {

}

func TestClient_PollIncomingMessages(t *testing.T) {

}
