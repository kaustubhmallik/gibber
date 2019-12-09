package service

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"regexp"
	"time"
)

// user document collection name and fields
const (
	UserCollection     = "users"
	UserFirstNameField = "first_name"
	UserLastNameField  = "last_name"
	UserEmailField     = "email"
	UserLoggedIn       = "logged_in"
	UserPasswordField  = "password"
	InvitesDataField   = "invites_data_id"
)

const ValidEmailRegex = `^[\w\.=-]+@[\w\.-]+\.[\w]{2,3}$`

// User details
type User struct {
	ID        primitive.ObjectID `bson:"_id" json:"-"`
	FirstName string             `bson:"first_name" json:"first_name"`
	LastName  string             `bson:"last_name" json:"last_name"`
	Email     string             `bson:"email" json:"email"`
	Password  string             `bson:"password" json:"password"` // hashed
	LastLogin time.Time          `bson:"last_login" json:"last_login"`
	LoggedIn  bool               `bson:"logged_in" json:"logged_in"`             // depicts if the user is currently logged in
	InvitesId primitive.ObjectID `bson:"invites_data_id" json:"invites_data_id"` // object ID of invitesData
}

func CreateUser(user *User) (userId interface{}, err error) {
	var fetchUser *User
	if user.ExistingUser() {
		reason := fmt.Sprintf("user %#v already exist with email %s", fetchUser, user.Email) // passed email userId should be unique
		GetLogger().Printf(reason)
		err = errors.New(reason)
		return
	}
	user.Password = GenerateHash(user.Password)
	user.LoggedIn = true // as user is just created, he becomes online, until he quits the session
	userMap := GetMap(user)
	userMap["last_login"] = time.Now().UTC()
	collection := MongoConn().Collection(UserCollection)
	res, err := collection.InsertOne(context.Background(), userMap)
	if err != nil {
		reason := fmt.Sprintf("error while creating new user %#v: %s", userMap, err)
		err = errors.New(reason)
		GetLogger().Printf(reason)
	} else {
		userId = res.InsertedID
		user.ID = res.InsertedID.(primitive.ObjectID)
		GetLogger().Printf("user %#v successfully created with userId: %v", userMap, res)
	}
	var invitesId primitive.ObjectID
	invitesDataId, err := CreateUserInvitesData(userId)
	invitesId = invitesDataId.(primitive.ObjectID)
	if err != nil {
		var delRes *mongo.DeleteResult
		delRes, err = MongoConn().Collection(UserCollection).DeleteOne(
			context.Background(),
			bson.M{
				ObjectID: userId,
			})
		if err == nil || (delRes != nil && delRes.DeletedCount == 0) {
			reason := fmt.Sprintf("error while rollbacking deleting user %s: %s", userId, err)
			GetLogger().Println(reason)
			return
		}
	}
	updateRes, err := MongoConn().Collection(UserCollection).UpdateOne(
		context.Background(),
		bson.M{ObjectID: userId},
		bson.D{{
			MongoSetOperator, bson.D{{InvitesDataField, invitesId}},
		}})
	if err != nil || updateRes.ModifiedCount != 1 {
		reason := fmt.Sprintf("error while setting up invites data for user%s: %s", userId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	}
	return
}

func GetUserByEmail(email string) (user *User, err error) {
	collection := MongoConn().Collection(UserCollection)
	user = &User{}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = collection.FindOne(ctx, bson.M{UserEmailField: email}).Decode(user)
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

func GetUserByID(objectID primitive.ObjectID) (user *User, err error) {
	collection := MongoConn().Collection(UserCollection)
	user = &User{}
	//ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			err = fmt.Errorf("no user found with ID: %s", objectID.String())
		} else {
			err = fmt.Errorf("decoding(unmarshal) user fetch result for email %s failed: %s", objectID.String(), err)
		}
		GetLogger().Println(err)
	}
	return
}

// raises an error if authentication fails due to any reason, including password mismatch
func (user *User) LoginUser(password string) (err error) {
	fetchDBUser, err := GetUserByEmail(user.Email)
	if err != nil {
		reason := fmt.Sprintf("authenticate user failed: %s", err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}
	err = MatchHashAndPlainText(fetchDBUser.Password, password)
	if err != nil {
		//reason := PasswordMismatch
		GetLogger().Print(err)
		//err = errors.New(reason) // TODO: May be we can create a new collection just to store credential and other auth related data
		return
	}

	result, err := MongoConn().Collection(UserCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   UserEmailField,
				Value: user.Email,
			},
		},
		bson.D{
			{
				Key: MongoSetOperator,
				Value: bson.D{
					{
						Key:   UserLoggedIn,
						Value: true,
					},
				},
			},
		},
	)
	if err != nil {
		reason := fmt.Sprintf("error while logging out user %s: %s", user.Email, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.MatchedCount != 1 { // TODO: should we check for logging status to change
		reason := fmt.Sprintf("error while logging out user %s: %s", user.Email, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}

	//user.Password = fetchDBUser.Password
	//user.FirstName = fetchDBUser.FirstName
	//user.LastName = fetchDBUser.LastName
	//user.ID = fetchDBUser.ID
	user.ID = fetchDBUser.ID
	user.FirstName = fetchDBUser.FirstName
	user.LastName = fetchDBUser.LastName
	user.Email = fetchDBUser.Email
	user.Password = fetchDBUser.Password
	user.LastLogin = fetchDBUser.LastLogin
	user.LoggedIn = fetchDBUser.LoggedIn
	user.InvitesId = fetchDBUser.InvitesId
	return
}

func (user *User) ExistingUser() (exists bool) {
	_, err := GetUserByEmail(user.Email) // if user not exists, it will throw an error
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
	result, err := MongoConn().Collection(UserCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   UserEmailField,
				Value: user.Email,
			},
		},
		bson.D{
			{
				Key: MongoSetOperator,
				Value: bson.D{
					{
						Key:   UserPasswordField,
						Value: newEncryptedPassword,
					},
				},
			},
		})
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
	var updatedDoc bson.D
	if firstName != "" && lastName != "" {
		updatedDoc = bson.D{
			{
				Key:   UserFirstNameField,
				Value: firstName,
			},
			{
				Key:   UserLastNameField,
				Value: lastName,
			},
		}
	} else if firstName != "" {
		updatedDoc = bson.D{
			{
				Key:   UserFirstNameField,
				Value: firstName,
			},
		}
	} else if lastName != "" {
		updatedDoc = bson.D{
			{
				Key:   UserLastNameField,
				Value: lastName,
			},
		}
	} else { // nothing to update
		reason := "nothing to update as both firstName and lastName are blank"
		GetLogger().Println(reason)
		return
	}
	result, err := MongoConn().Collection(UserCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   UserEmailField,
				Value: user.Email,
			},
		},
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

func (user *User) SeeOnlineFriends() (onlineFriends []string, err error) {
	//fetchedUser := &User{}
	//MongoConn().Collection(UserCollection).FindOne(
	//	context.Background(),
	//	bson.M{
	//		UserEmailField: user.Email,
	//	}).Decode(fetchedUser)
	//_, err = GetUserInvitesData(fetchedUser.InvitesId)
	//if err != nil {
	//	return
	//}
	//friendEmails, err := GetAcceptedInvitations(user.Email)
	//if err != nil {
	//	reason := fmt.Sprintf("error while fetching user %s accepted invitations: %s", user.Email, err)
	//	GetLogger().Print(reason)
	//}
	//onlineFriends = make([]string, 5)
	//for _, acceptedInvite := range friendEmails {
	//	friend := &User{}
	//	if acceptedInvite.Sender != user.Email {
	//		MongoConn().Collection(UserCollection).FindOne(
	//			context.Background(),
	//			bson.M{
	//				UserEmailField: acceptedInvite.Sender,
	//				UserLoggedIn:   true,
	//			}).Decode(friend)
	//	}
	//	if friend.Password != "" { // a user is found
	//		onlineFriends = append(onlineFriends, fmt.Sprintf("%s %s: %s", friend.FirstName, friend.LastName,
	//			friend.Email))
	//	}
	//}
	return
}

func (user *User) Logout() (err error) {
	result, err := MongoConn().Collection(UserCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				UserEmailField,
				user.Email,
			},
		},
		bson.D{
			{
				MongoSetOperator, bson.D{{UserLoggedIn, false}},
			},
		},
	)
	if err != nil {
		reason := fmt.Sprintf("error while logging out user %s: %s", user.Email, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("error while logging out user %s: %s", user.Email, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}
	return
}

func (u *User) SendInvitation(recv *User) (err error) {
	// TODO: Add mongo transaction
	//ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	result, err := MongoConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				UserIdField,
				u.ID,
			},
		},
		bson.D{
			{
				MongoPushOperator, bson.D{{SentInvitesField, recv.ID}},
			},
		},
	)
	if err != nil {
		reason := fmt.Sprintf("sending invitation failed from %s to %s: %s", u.Email, recv.Email, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("sending invitation failed from %s to %s as no doc modified", u.Email, recv.Email)
		GetLogger().Println(reason)
	}

	//ctx, _ = context.WithTimeout(context.Background(), 5*time.Second)
	result, err = MongoConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				UserIdField,
				recv.ID,
			},
		},
		bson.D{
			{
				MongoPushOperator, bson.D{{ReceivedInvitesField, u.ID}},
			},
		},
	)
	if err != nil {
		reason := fmt.Sprintf("sending invitation failed from %s to %s: %s", u.Email, recv.Email, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("sending invitation failed from %s to %s as no doc modified", u.Email, recv.Email)
		GetLogger().Println(reason)
	}
	return
}

func (u *User) GetActiveReceivedInvitations() (invites []primitive.ObjectID, err error) {
	invitesData := UserInvitesData{}
	err = MongoConn().Collection(UserInvitesCollection).FindOne(
		context.Background(),
		bson.M{UserIdField: u.ID}).
		Decode(&invitesData)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			err = fmt.Errorf("invite data not found for user %s", u.ID.String())
		} else {
			err = fmt.Errorf("invite data fetch failed for user %s: %s", u.ID.String(), err)
		}
		GetLogger().Print(err)
		return
	}
	invites = invitesData.Received
	return
}

func (u *User) AddFriend(user *User) error {
	return nil
}

func (u *User) GetActiveSentInvitations() (invites []primitive.ObjectID, err error) {
	invitesData := UserInvitesData{}
	err = MongoConn().Collection(UserInvitesCollection).FindOne(
		context.Background(),
		bson.M{UserIdField: u.ID}).
		Decode(&invitesData)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			err = fmt.Errorf("invite data not found for user %s", u.ID.String())
		} else {
			err = fmt.Errorf("invite data fetch failed for user %s: %s", u.ID.String(), err)
		}
		GetLogger().Print(err)
		return
	}
	invites = invitesData.Sent
	return
}

func (u *User) CancelInvitation(user *User) error {
	return nil
}

func (u *User) SeeFriends() ([]User, error) {
	return nil, nil
}

func (u *User) String() string {
	return u.ID.String()
}

func ValidUserEmail(email string) bool {
	return regexp.MustCompile(ValidEmailRegex).MatchString(email)
}
