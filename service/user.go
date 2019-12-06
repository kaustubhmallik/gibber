package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/fatih/structs"
	"go.mongodb.org/mongo-driver/mongo"
	"regexp"
	"time"
)

// user document collection name and fields
const (
	UserCollection     = "users"
	UserFirstNameField = "firstname"
	UserLastNameField  = "lastname"
	UserEmailField     = "email"
	UserLoggedIn       = "loggedin"
	UserPasswordField  = "password"
	InvitesDataField   = "invitesdataid"
)

const ValidEmailRegex = `^[\w\.=-]+@[\w\.-]+\.[\w]{2,3}$`

// User details
type User struct {
	Id                  interface{}
	FirstName, LastName string
	Email               string
	Password            string // hashed
	LastLogin           time.Time
	LoggedIn            bool   // depicts if the user is currently logged in
	InvitesDataId       string // object ID of invitesData
}

func CreateUser(user *User) (userId string, err error) {
	var fetchUser *User
	if user.ExistingUser() {
		reason := fmt.Sprintf("user %#v already exist with email %s", fetchUser, user.Email) // passed email userId should be unique
		GetLogger().Printf(reason)
		err = errors.New(reason)
		return
	}
	user.Password = GetSHA512Encrypted(user.Password)
	user.LoggedIn = true // as user is just created, he becomes online, until he quits the session
	userMap := MapLowercaseKeys(structs.Map(*user))
	collection := GetDBConn().Collection(UserCollection)
	res, err := collection.InsertOne(context.Background(), userMap)
	if err != nil {
		reason := fmt.Sprintf("error while creating new user %#v: %s", userMap, err)
		err = errors.New(reason)
		GetLogger().Printf(reason)
	} else {
		userId = res.InsertedID.(string) // TODO: Check if it is mostly string (expected), change the userId to string, and use reflection on InsertID
		GetLogger().Printf("user %#v successfully created with userId: %v", userMap, res)
	}
	//var invitesDataId string
	//invitesDataId, err = CreateUserInvitesData(userId)
	if err != nil {
		//var delRes *DeleteResult
		//delRes, err = GetDBConn().Collection(UserCollection).DeleteOne(
		//	context.Background(),
		//	bson.NewDocument(
		//		bson.EC.String(ObjectID, userId),
		//	),
		//)
		//if err == nil || delRes.DeletedCount == 0 {
		//	reason := fmt.Sprintf("error while rollbacking deleting user %s: %s", userId, err)
		//	GetLogger().Println(reason)
		//	return
		//}
	}
	//updateRes, err := GetDBConn().Collection(UserCollection).UpdateOne(
	//context.Background(),
	//bson.NewDocument(
	//	bson.EC.String(ObjectID, userId),
	//),
	//bson.NewDocument(
	//	bson.EC.String(InvitesDataField, invitesDataId),
	//),
	//)
	//if err != nil || updateRes.ModifiedCount != 1 {
	//	reason := fmt.Sprintf("error while setting up invites data for user%s: %s", userId, err)
	//	GetLogger().Println(reason)
	//	err = errors.New(reason)
	//}
	return
}

func GetUser(email string) (user *User, err error) {
	//collection := GetDBConn().Collection(UserCollection)
	user = &User{}
	//err = collection.FindOne(
	//	context.Background(),
	//	bson.NewDocument(
	//		bson.EC.String(UserEmailField, email),
	//	),
	//).Decode(user)
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
func (user *User) LoginUser(password string) (err error) {
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

	//userFilter := bson.NewDocument(
	//	bson.EC.String(UserEmailField, user.Email),
	//)
	//userData := bson.NewDocument(
	//	bson.EC.SubDocumentFromElements(MongoSetOperator,
	//		bson.EC.Boolean(UserLoggedIn, true)),
	//)
	//result, err := GetDBConn().Collection(UserCollection).UpdateOne(
	//	context.Background(),
	//	userFilter,
	//	userData,
	//)
	//if err != nil {
	//	reason := fmt.Sprintf("error while logging out user %s: %s", user.Email, err)
	//	GetLogger().Println(reason)
	//	err = errors.New(reason)
	//	return
	//} else if result.ModifiedCount != 1 {
	//	reason := fmt.Sprintf("error while logging out user %s: %s", user.Email, err)
	//	GetLogger().Println(reason)
	//	err = errors.New(reason)
	//	return
	//}

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
	//result, err := GetDBConn().Collection(UserCollection).UpdateOne(
	//	context.Background(),
	//	bson.NewDocument(
	//		bson.EC.String(UserEmailField, user.Email),
	//	),
	//	bson.NewDocument(
	//		bson.EC.SubDocumentFromElements(MongoSetOperator,
	//			bson.EC.String(UserPasswordField, newEncryptedPassword),
	//		),
	//	),
	//)
	//if err != nil {
	//	reason := fmt.Sprintf("password update failed for user %s: %s", user.Email, err)
	//	GetLogger().Println(reason)
	//	err = errors.New(reason)
	//} else {
	//	reason := fmt.Sprintf("password update successful for user %s: %+v", user.Email, result)
	//	GetLogger().Println(reason)
	//}
	return
}

func (user *User) UpdateName(firstName, lastName string) (err error) {
	//var updatedDoc *bson.Document
	//if firstName != "" && lastName != "" {
	//	updatedDoc = bson.NewDocument(
	//		bson.EC.SubDocumentFromElements(MongoSetOperator,
	//			bson.EC.String(UserFirstNameField, firstName),
	//			bson.EC.String(UserLastNameField, lastName),
	//		),
	//	)
	//} else if firstName != "" {
	//	updatedDoc = bson.NewDocument(
	//		bson.EC.SubDocumentFromElements(MongoSetOperator,
	//			bson.EC.String(UserFirstNameField, firstName),
	//		),
	//	)
	//} else if lastName != "" {
	//	updatedDoc = bson.NewDocument(
	//		bson.EC.SubDocumentFromElements(MongoSetOperator,
	//			bson.EC.String(UserLastNameField, lastName),
	//		),
	//	)
	//} else { // nothing to update
	//	reason := "nothing to update as both firstName and lastName are blank"
	//	GetLogger().Println(reason)
	//	return
	//}
	//currentDocFilter := bson.NewDocument(
	//	bson.EC.String(UserEmailField, user.Email),
	//)
	//result, err := GetDBConn().Collection(UserCollection).UpdateOne(
	//	context.Background(),
	//	currentDocFilter,
	//	updatedDoc,
	//)
	//if err != nil {
	//	reason := fmt.Sprintf("name update failed for user %s: %s", user.Email, err)
	//	GetLogger().Println(reason)
	//	err = errors.New(reason)
	//} else {
	//	reason := fmt.Sprintf("name update successful for user %s: %+v", user.Email, result)
	//	GetLogger().Println(reason)
	//}
	return
}

func (user *User) SeeOnlineFriends() (onlineFriends []string, err error) {
	//fetchedUser := &User{}
	//GetDBConn().Collection(UserCollection).FindOne(
	//	context.Background(),
	//	bson.NewDocument(
	//		bson.EC.String(UserEmailField, user.Email),
	//	),
	//).Decode(fetchedUser)
	//invitesData, err := GetUserInvitesData(fetchedUser.InvitesDataId)
	//if err != nil {
	//	return
	//}
	//friendEmails, err := GetAcceptedInvitations(user.Id.(string))
	//if err != nil {
	//	reason := fmt.Sprintf("error while fetching user %s accepted invitations: %s", user.Id, err)
	//}
	//onlineFriends = make([]string, 5)
	//for _, acceptedInvite := range friendEmails {
	//	friend := &User{}
	//	if acceptedInvite.Sender != user.Id {
	//		GetDBConn().Collection(UserCollection).FindOne(
	//			context.Background(),
	//			bson.NewDocument(
	//				bson.EC.String(UserEmailField, acceptedInvite.Sender),
	//				bson.EC.Boolean(UserLoggedIn, true),
	//			),
	//		).Decode(friend)
	//	}
	//	if friend.Password != "" { // a user is found
	//		onlineFriends = append(onlineFriends, fmt.Sprintf("%s %s: %s", friend.FirstName, friend.LastName, friendEmail))
	//	}
	//}
	return
}

func (user *User) Logout() (err error) {
	//userFilter := bson.NewDocument(
	//	bson.EC.String(UserEmailField, user.Email),
	//)
	//userData := bson.NewDocument(
	//	bson.EC.SubDocumentFromElements(MongoSetOperator,
	//		bson.EC.Boolean(UserLoggedIn, false)),
	//)
	//result, err := GetDBConn().Collection(UserCollection).UpdateOne(
	//	context.Background(),
	//	userFilter,
	//	userData,
	//)
	//if err != nil {
	//	reason := fmt.Sprintf("error while logging out user %s: %s", user.Email, err)
	//	GetLogger().Println(reason)
	//	err = errors.New(reason)
	//	return
	//} else if result.ModifiedCount != 1 {
	//	reason := fmt.Sprintf("error while logging out user %s: %s", user.Email, err)
	//	GetLogger().Println(reason)
	//	err = errors.New(reason)
	//	return
	//}
	return
}

func (u *User) SendInvitation(user *User) error {
	return nil
}

func (u *User) GetActiveReceivedInvitations() ([]Invite, error) {
	return nil, nil
}

func (u *User) AddFriend(user *User) error {
	return nil
}

func (u *User) GetActiveSentInvitations() ([]Invite, error) {
	return nil, nil
}

func (u *User) CancelInvitation(user *User) error {
	return nil
}

func (u *User) SeeFriends() ([]User, error) {
	return nil, nil
}

func ValidUserEmail(email string) bool {
	return regexp.MustCompile(ValidEmailRegex).MatchString(email)
}
