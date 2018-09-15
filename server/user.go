package server

import (
	"context"
	"fmt"
	"github.com/fatih/structs"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/pkg/errors"
)

const UserCollection = "users"
const UserEmailField = "email"

const ValidEmailRegex = `^[\w\.=-]+@[\w\.-]+\.[\w]{2,3}$`

// User details
type User struct {
	FirstName, LastName string
	Email               string
	Password            string // hashed
}

func CreateUser(user *User) (id interface{}, err error) {
	var fetchUser *User
	if !validUserEmail(user.Email) { // check for valid email - regex based
		reason := fmt.Sprintf("invalid email %s", user.Email)
		GetLogger().Printf(reason)
		err = fmt.Errorf(reason)
		return
	}
	if user.ExistingUser() {
		reason := fmt.Sprintf("user %#v already exist with email %s", fetchUser, user.Email) // passed email id should be unique
		GetLogger().Printf(reason)
		err = fmt.Errorf(reason)
		return
	}
	userMap := MapLowercaseKeys(structs.Map(*user))
	collection := GetDBConn().Collection(UserCollection)
	res, err := collection.InsertOne(context.Background(), userMap)
	if err != nil {
		reason := fmt.Sprintf("error while creating new user %#v: %s", userMap, err)
		err = errors.New(reason)
		GetLogger().Printf(reason)
	} else {
		id = res.InsertedID // TODO: Check if it is mostly string (expected), change the id to string, and use reflection on InsertID
		GetLogger().Printf("user %#v successfully created with id: %v", userMap, res)
	}
	return
}

func GetUser(email string) (user *User, err error) {
	collection := GetDBConn().Collection(UserCollection)
	user = &User{}
	err = collection.FindOne(context.Background(), bson.NewDocument(bson.EC.String(UserEmailField, email))).Decode(user)
	if err == mongo.ErrNoDocuments {
		reason := fmt.Sprintf("no user found with email: %s", email)
		GetLogger().Println(reason)
		// no changes in error so that it can be used to verify unique email ID before insertion
	} else if err != nil {
		reason := fmt.Sprintf("decoding(unmarshal) user fetch result for email %s failed: %s", email, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	}
	return
}

// raises an error if authentication fails due to any reason, including password mismatch
func (user *User) AuthenticateUser(password string) (err error) {
	if err != nil {
		return
	}
	if user.Password != GetSHA512Encrypted(password) {
		err = fmt.Errorf("password mismatch") // TODO: May be we can create a new collection just to store credential and other auth related data
		return
	}
	return
}

func (u *User) ExistingUser() (exists bool) {
	_, err := GetUser(u.Email) // if user not exists, it will throw an error
	if err == mongo.ErrNoDocuments {
		return
	}
	if err != nil { // some other error occurred
		reason := fmt.Sprintf("user email unique check failed: %s", err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}
	exists = true
	return
}

func validUserEmail(email string) bool {
	return true
	//return regexp.MustCompile(ValidEmailRegex).MatchString(email)
}
