package service

// Testing the functionality as a client i.e. from client-end

import (
	"bufio"
	"context"
	user2 "gibber/user"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"log"
	"math/rand"
	"net"
	"testing"
	"time"
)

var scanner *bufio.Scanner
var writer *bufio.Writer

func init() {
	ctx, f := context.WithTimeout(context.TODO(), 3*time.Second)
	go func() {
		_ = StartServer("localhost", "44517", f)
	}()
	<-ctx.Done() // wait till the server initialization is completed

	//connect to server
	conn, err := net.Dial("tcp", "localhost:44517")
	if err != nil {
		log.Fatalf("unable to connect to tcp server at localhost:44517 : %s", err)
	}

	scanner = bufio.NewScanner(conn)
	writer = bufio.NewWriter(conn)
}

func TestClient(t *testing.T) {
	password := "password"
	hashPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := &user2.User{
		ID:        primitive.NewObjectID(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john" + randomString(20) + "@doe.com",
		Password:  string(hashPassword),
	}

	userID, err := user2.CreateUser(user)
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

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
