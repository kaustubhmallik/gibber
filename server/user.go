package server

import (
	"context"
	"fmt"
	"github.com/fatih/structs"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/pkg/errors"
	"regexp"
	"time"
)

// user document collection name and fields
const (
	UserCollection              = "users"
	UserFirstNameField          = "firstname"
	UserLastNameField           = "lastname"
	UserEmailField              = "email"
	UserFriendsField            = "friends"
	UserPasswordField           = "password"
	UserActiveInvitesSentField  = "activeInvites.sent"
	UserActiveInvitesRecvdField = "activeInvites.received"
)

const ValidEmailRegex = `^[\w\.=-]+@[\w\.-]+\.[\w]{2,3}$`

type InvitesData struct {
	Sent     []string // user emails
	Received []string // user emails
}

// User details
type User struct {
	FirstName, LastName string
	Email               string
	Password            string // hashed
	LastLogin           time.Time
	ActiveInvites       InvitesData
	InActiveInvites     InvitesData
	ConnectedPeople     []string // user emails
}

func CreateUser(user *User) (id interface{}, err error) {
	var fetchUser *User
	if user.ExistingUser() {
		reason := fmt.Sprintf("user %#v already exist with email %s", fetchUser, user.Email) // passed email id should be unique
		GetLogger().Printf(reason)
		err = errors.New(reason)
		return
	}
	user.Password = GetSHA512Encrypted(user.Password)
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
	err = collection.FindOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(UserEmailField, email),
		),
	).Decode(user)
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
	fetchDBUser, err := GetUser(user.Email)
	if err != nil {
		reason := fmt.Sprintf("authenticate user failed: %s", err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}
	if fetchDBUser.Password != GetSHA512Encrypted(password) {
		reason := PasswordMismatch
		GetLogger().Println(reason)
		err = errors.New(reason) // TODO: May be we can create a new collection just to store credential and other auth related data
		return
	}
	user.Password = fetchDBUser.Password
	user.FirstName = fetchDBUser.FirstName
	user.LastName = fetchDBUser.LastName
	return
}

func (user *User) ExistingUser() (exists bool) {
	_, err := GetUser(user.Email) // if user not exists, it will throw an error
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

func (user *User) UpdatePassword(newEncryptedPassword string) (err error) {
	result, err := GetDBConn().Collection(UserCollection).UpdateOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(UserEmailField, user.Email),
		),
		bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoSetOperator,
				bson.EC.String(UserPasswordField, newEncryptedPassword),
			),
		),
	)
	if err != nil {
		reason := fmt.Sprintf("password update failed for user %s: %s", user.Email, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	} else {
		reason := fmt.Sprintf("password update successful for user %s: %+v", user.Email, result)
		GetLogger().Println(reason)
	}
	return
}

func (user *User) UpdateName(firstName, lastName string) (err error) {
	var updatedDoc *bson.Document
	if firstName != "" && lastName != "" {
		updatedDoc = bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoSetOperator,
				bson.EC.String(UserFirstNameField, firstName),
				bson.EC.String(UserLastNameField, lastName),
			),
		)
	} else if firstName != "" {
		updatedDoc = bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoSetOperator,
				bson.EC.String(UserFirstNameField, firstName),
			),
		)
	} else if lastName != "" {
		updatedDoc = bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoSetOperator,
				bson.EC.String(UserLastNameField, lastName),
			),
		)
	} else { // nothing to update
		reason := "nothing to update as both firstName and lastName are blank"
		GetLogger().Println(reason)
		return
	}
	currentDocFilter := bson.NewDocument(
		bson.EC.String(UserEmailField, user.Email),
	)
	result, err := GetDBConn().Collection(UserCollection).UpdateOne(
		context.Background(),
		currentDocFilter,
		updatedDoc,
	)
	if err != nil {
		reason := fmt.Sprintf("name update failed for user %s: %s", user.Email, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	} else {
		reason := fmt.Sprintf("name update successful for user %s: %+v", user.Email, result)
		GetLogger().Println(reason)
	}
	return
}

func (user *User) SendInvitation(invitedUser *User) (err error) {
	// check if they are not already connected

	GetDBConn().Collection(UserCollection).FindOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(UserEmailField, user.Email),
			bson.EC.String(UserFriendsField, invitedUser.Email),
		),
	)

	// add an invite to the recipient user's received array
	invitedUserFilter := bson.NewDocument(
		bson.EC.String(UserEmailField, user.Email),
	)
	inviteeUserData := bson.NewDocument(
		// TODO: Should first check if there is already an request from same invitee is made to the invited user
		// Using $addToSet for now, will change it to $push once we add check if the request can't be repeated
		bson.EC.SubDocumentFromElements(MongoAddToSetOperator,
			// just storing as name can be changed b/w sending request and seen by intended receiver
			// so will fetch other details when the receiver will see the invitation
			bson.EC.String(UserActiveInvitesSentField, invitedUser.Email)),
	)
	GetDBConn().Collection(UserCollection).UpdateOne(
		context.Background(),
		invitedUserFilter,
		inviteeUserData,
	)

	inviteeUserFilter := bson.NewDocument(
		bson.EC.String(UserEmailField, invitedUser.Email),
	)
	invitedUserData := bson.NewDocument(
		bson.EC.SubDocumentFromElements(MongoAddToSetOperator,
			bson.EC.String(UserActiveInvitesRecvdField, user.Email)),
	)
	GetDBConn().Collection(UserCollection).UpdateOne(
		context.Background(),
		inviteeUserFilter,
		invitedUserData,
	)
	return
}

// a user can see her active invitations and take an action on it i.e. accept/reject it
// once he/she takes an action, it is pushed to the inactive invitations with the added
// details of action taken
func (user *User) FetchActiveReceivedInvitations() (invites []string, err error) {
	fetchedUser := &User{}
	GetDBConn().Collection(UserCollection).FindOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(UserEmailField, user.Email),
		),
	).Decode(fetchedUser)
	invites = fetchedUser.ActiveInvites.Received
	return
}

func (user *User) FetchActiveSentInvitations() (invites []string, err error) {
	fetchedUser := &User{}
	GetDBConn().Collection(UserCollection).FindOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(UserEmailField, user.Email),
		),
	).Decode(fetchedUser)
	invites = fetchedUser.ActiveInvites.Sent
	return
}

func (user *User) FetchInactiveSentInvitations() (invites []string, err error) {
	fetchedUser := &User{}
	GetDBConn().Collection(UserCollection).FindOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(UserEmailField, user.Email),
		),
	).Decode(fetchedUser)
	invites = fetchedUser.InActiveInvites.Sent
	return
}

func (user *User) FetchInactiveReceivedInvitations() (invites []string, err error) {
	fetchedUser := &User{}
	GetDBConn().Collection(UserCollection).FindOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(UserEmailField, user.Email),
		),
	).Decode(fetchedUser)
	invites = fetchedUser.InActiveInvites.Received
	return
}

func ValidUserEmail(email string) bool {
	return regexp.MustCompile(ValidEmailRegex).MatchString(email)
}
