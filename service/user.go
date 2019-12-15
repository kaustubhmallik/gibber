package service

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	LastLogin          = "last_login"
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
		Logger().Printf(reason)
		err = errors.New(reason)
		return
	}
	user.Password = GenerateHash(user.Password)
	user.LoggedIn = true // as user is just created, he becomes online, until he quits the session
	userMap := GetMap(user)
	userMap["last_login"] = time.Now().UTC()

	session, err := MongoConn().Client().StartSession()
	if err != nil {
		Logger().Printf("initializing mongo session failed: %s", err)
		return
	}

	err = session.StartTransaction()
	if err != nil {
		Logger().Printf("initializing mongo transaction failed: %s", err)
		return
	}

	err = mongo.WithSession(context.Background(), session, func(sc mongo.SessionContext) (er error) {
		// create user
		res, er := MongoConn().Collection(UserCollection).InsertOne(sc, userMap)
		if er != nil {
			_ = session.AbortTransaction(sc)
			er = fmt.Errorf("error while creating new user %#v: %s", userMap, er)
			Logger().Print(er)
			return
		} else {
			userId = res.InsertedID
			user.ID = res.InsertedID.(primitive.ObjectID)
			Logger().Printf("user %#v successfully created with userId: %v", userMap, res)
		}

		// create user_invite
		var invitesId primitive.ObjectID
		invitesDataId, er := CreateUserInvitesData(userId, sc)
		invitesId = invitesDataId.(primitive.ObjectID)
		if er != nil {
			_ = session.AbortTransaction(sc)
			Logger().Printf("error creating user invite: %s", er)
			return er
		}

		// update invite_user doc ID in user's doc
		updateRes, er := MongoConn().Collection(UserCollection).UpdateOne(
			sc,
			bson.M{ObjectID: userId},
			bson.D{{
				Key: MongoSetOperator, Value: bson.D{{Key: InvitesDataField, Value: invitesId}},
			}})
		if er != nil || updateRes.ModifiedCount != 1 {
			er = fmt.Errorf("error while setting up invites data for user%s: %s", userId, err)
			Logger().Print(er)
		}

		// commit transaction
		if er := session.CommitTransaction(sc); er != nil {
			Logger().Printf("committing mongo transaction failed: %s", er)
			er = session.AbortTransaction(sc)
			if er != nil {
				Logger().Printf("aborting mongo transaction failed: %s", er)
			}
		}
		return nil
	})

	session.EndSession(context.Background())
	return
}

func GetUserByEmail(email string) (user *User, err error) {
	collection := MongoConn().Collection(UserCollection)
	user = &User{}
	err = collection.FindOne(context.Background(), bson.M{UserEmailField: email}).Decode(user)
	if err == mongo.ErrNoDocuments {
		reason := fmt.Sprintf("no user found with email: %s", email)
		Logger().Println(reason)
		// no changes in error so that it can be used to verify unique email ID before insertion
	} else if err != nil {
		reason := fmt.Sprintf("decoding(unmarshal) user fetch result for email %s failed: %s", email, err)
		Logger().Println(reason)
		err = errors.New(reason)
	}
	return
}

func GetUserByID(objectID primitive.ObjectID) (user *User, err error) {
	user = &User{}
	err = MongoConn().
		Collection(UserCollection).
		FindOne(context.Background(), bson.M{ObjectID: objectID}).
		Decode(user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			err = fmt.Errorf("no user found with ID: %s", objectID.String())
		} else {
			err = fmt.Errorf("decoding(unmarshal) user fetch result for email %s failed: %s", objectID.String(), err)
		}
		Logger().Println(err)
	}
	return
}

// raises an error if authentication fails due to any reason, including password mismatch
func (user *User) LoginUser(password string) (lastLogin string, err error) {
	fetchDBUser, err := GetUserByEmail(user.Email)
	if err != nil {
		reason := fmt.Sprintf("authenticate user failed: %s", err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}
	err = MatchHashAndPlainText(fetchDBUser.Password, password)
	if err != nil {
		//reason := PasswordMismatch
		Logger().Print(err)
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
					{
						Key:   LastLogin,
						Value: time.Now().UTC(),
					},
				},
			},
		},
	)
	if err != nil {
		reason := fmt.Sprintf("error while logging out user %s: %s", user.Email, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.MatchedCount != 1 { // TODO: should we check for logging status to change
		reason := fmt.Sprintf("error while logging out user %s: %s", user.Email, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}

	user.ID = fetchDBUser.ID
	user.FirstName = fetchDBUser.FirstName
	user.LastName = fetchDBUser.LastName
	user.Email = fetchDBUser.Email
	user.Password = fetchDBUser.Password
	user.LastLogin = fetchDBUser.LastLogin
	user.LoggedIn = fetchDBUser.LoggedIn
	user.InvitesId = fetchDBUser.InvitesId
	lastLogin = fetchDBUser.LastLogin.Format(time.RFC3339)
	return
}

func (user *User) ExistingUser() (exists bool) {
	_, err := GetUserByEmail(user.Email) // if user not exists, it will throw an error
	if err == mongo.ErrNoDocuments {
		return
	}
	if err != nil { // some other error occurred
		err = fmt.Errorf("user email unique check failed: %s", err)
		Logger().Println(err)
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
		Logger().Println(reason)
		err = errors.New(reason)
	} else {
		reason := fmt.Sprintf("password update successful for user %s: %+v", user.Email, result)
		Logger().Println(reason)
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
		Logger().Println(reason)
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
		Logger().Println(reason)
		err = errors.New(reason)
	} else {
		reason := fmt.Sprintf("name update successful for user %s: %+v", user.Email, result)
		Logger().Println(reason)
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
	//	Logger().Print(reason)
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
			{Key: UserEmailField, Value: user.Email},
		},
		bson.D{
			{Key: MongoSetOperator, Value: bson.D{{Key: UserLoggedIn, Value: false}}},
		},
	)
	if err != nil {
		reason := fmt.Sprintf("error while logging out user %s: %s", user.Email, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("error while logging out user %s: %s", user.Email, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}
	return
}

func (u *User) SendInvitation(recv *User) (err error) {
	session, err := MongoConn().Client().StartSession()
	if err != nil {
		Logger().Printf("initializing mongo session failed: %s", err)
		return
	}

	err = session.StartTransaction()
	if err != nil {
		Logger().Printf("initializing mongo transaction failed: %s", err)
		return
	}

	err = mongo.WithSession(context.Background(), session, func(sc mongo.SessionContext) (er error) {
		result, er := MongoConn().Collection(UserInvitesCollection).UpdateOne(
			sc,
			bson.D{
				{Key: UserIdField, Value: u.ID},
			},
			bson.D{
				{Key: MongoPushOperator, Value: bson.D{{Key: SentInvitesField, Value: recv.ID}}},
			},
		)
		if er != nil {
			_ = session.AbortTransaction(sc)
			er = fmt.Errorf("sending invitation failed from %s to %s: %s", u.Email, recv.Email, err)
			Logger().Println(er)
		} else if result.ModifiedCount != 1 {
			_ = session.AbortTransaction(sc)
			reason := fmt.Sprintf("sending invitation failed from %s to %s as no doc modified", u.Email, recv.Email)
			Logger().Println(reason)
		}

		result, er = MongoConn().Collection(UserInvitesCollection).UpdateOne(
			sc,
			bson.D{
				{Key: UserIdField, Value: recv.ID},
			},
			bson.D{
				{Key: MongoPushOperator, Value: bson.D{{Key: ReceivedInvitesField, Value: u.ID}}},
			},
		)
		if er != nil {
			_ = session.AbortTransaction(sc)
			er = fmt.Errorf("sending invitation failed from %s to %s: %s", u.Email, recv.Email, err)
			Logger().Print(er)
		} else if result.ModifiedCount != 1 {
			_ = session.AbortTransaction(sc)
			reason := fmt.Sprintf("sending invitation failed from %s to %s as no doc modified", u.Email, recv.Email)
			Logger().Println(reason)
		}

		// commit transaction
		if er := session.CommitTransaction(sc); er != nil {
			Logger().Printf("committing mongo transaction failed: %s", er)
			er = session.AbortTransaction(sc)
			if er != nil {
				Logger().Printf("aborting mongo transaction failed: %s", er)
			}
		}
		return
	})
	return
}

func (u *User) AddFriend(userID primitive.ObjectID) (err error) {
	// Delete the received request from u's invite data (THINK OF SOFT DELETE) - Done
	// Delete the sent request from user's invites data (THINK OF SOFT DELETE) - Done
	// Add user as u's friend
	// Add u as user's
	session, err := MongoConn().Client().StartSession()
	if err != nil {
		Logger().Printf("initializing mongo session failed: %s", err)
		return
	}

	err = session.StartTransaction()
	if err != nil {
		Logger().Printf("initializing mongo transaction failed: %s", err)
		return
	}

	err = mongo.WithSession(context.Background(), session, func(sc mongo.SessionContext) (er error) {
		var res *mongo.UpdateResult
		res, er = MongoConn().Collection(UserInvitesCollection).UpdateOne(
			sc,
			bson.M{UserIdField: u.ID},
			bson.D{
				{Key: MongoPullOperator, Value: bson.D{{Key: ReceivedInvitesField, Value: userID}}},
			})
		if er != nil {
			_ = session.AbortTransaction(sc) // ROLLBACK at the earliest to shorten transaction life-cycle
			if er == mongo.ErrNoDocuments {
				er = fmt.Errorf("invite data not found for invite accepting user %s", u.ID.String())
			} else {
				er = fmt.Errorf("invite data fetch failed for invite accepting user %s: %s", u.ID.String(), er)
			}
			Logger().Print(er)
			return
		} else if res.ModifiedCount != 1 {
			er = fmt.Errorf("invite data not updated for invite accepting user %s", u.ID.String())
			Logger().Print(er)
			return
		}

		res, er = MongoConn().Collection(UserInvitesCollection).UpdateOne(
			sc,
			bson.M{UserIdField: userID},
			bson.D{
				{Key: MongoPullOperator, Value: bson.D{{Key: SentInvitesField, Value: u.ID}}},
			})
		if er != nil {
			_ = session.AbortTransaction(sc)
			if er == mongo.ErrNoDocuments {
				er = fmt.Errorf("invite data not found for invite sending user %s", u.ID.String())
			} else {
				er = fmt.Errorf("invite data fetch failed for invite sending user %s: %s", u.ID.String(), er)
			}
			Logger().Print(er)
			return
		} else if res.ModifiedCount != 1 {
			er = fmt.Errorf("invite data not updated for invite sending user %s", userID.String())
			Logger().Print(er)
			return
		}

		// using upsert: true to create the friends document if non-existent
		res, er = MongoConn().Collection(FriendsCollection).UpdateOne(
			sc,
			bson.M{UserIdField: u.ID},
			bson.D{
				{Key: MongoPushOperator, Value: bson.D{{Key: FriendsField, Value: userID}}},
			},
			options.Update().SetUpsert(true))
		if er != nil {
			_ = session.AbortTransaction(sc) // ROLLBACK at the earliest to shorten transaction life-cycle
			er = fmt.Errorf("error while adding %s as friend for %s: %s", userID.String(), u.ID.String(), er)
			Logger().Print(er)
			return
		} else if res.ModifiedCount+res.UpsertedCount != 1 {
			er = fmt.Errorf("document not created/updated to add %s as friend for %s: %s", userID.String(),
				u.ID.String(), er)
			Logger().Print(er)
			return
		}

		// using upsert: true to create the friends document if non-existent
		res, err = MongoConn().Collection(FriendsCollection).UpdateOne(
			sc,
			bson.M{UserIdField: userID},
			bson.D{
				{Key: MongoPushOperator, Value: bson.D{{Key: FriendsField, Value: u.ID}}},
			},
			options.Update().SetUpsert(true))
		if err != nil {
			_ = session.AbortTransaction(sc) // ROLLBACK at the earliest to shorten transaction life-cycle
			err = fmt.Errorf("error while adding %s as friend for %s: %s", u.ID.String(), userID.String(), err)
			Logger().Print(err)
			return
		} else if res.ModifiedCount+res.UpsertedCount != 1 {
			err = fmt.Errorf("document not created/updated to add %s as friend for %s: %s", u.ID.String(),
				userID.String(), err)
			return
		}

		// commit transaction
		if er := session.CommitTransaction(sc); er != nil {
			Logger().Printf("committing mongo transaction failed: %s", er)
			er = session.AbortTransaction(sc)
			if er != nil {
				Logger().Printf("aborting mongo transaction failed: %s", er)
			}
		}
		return
	})
	return
}

func (u *User) GetSentInvitations() (invites []primitive.ObjectID, err error) {
	invitesData := UserInvites{}
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
		Logger().Print(err)
		return
	}
	invites = invitesData.Sent
	return
}

func (u *User) GetReceivedInvitations() (invites []primitive.ObjectID, err error) {
	invitesData := UserInvites{}
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
		Logger().Print(err)
		return
	}
	invites = invitesData.Received
	return
}

func (u *User) GetCanceledSentInvitations() (invites []primitive.ObjectID, err error) {
	invitesData := UserInvites{}
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
		Logger().Print(err)
		return
	}
	invites = invitesData.Cancelled
	return
}

func (u *User) GetAcceptedInvitations() (invites []primitive.ObjectID, err error) {
	invitesData := UserInvites{}
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
		Logger().Print(err)
		return
	}
	invites = invitesData.Accepted
	return
}

func (u *User) GetRejectedInvitations() (invites []primitive.ObjectID, err error) {
	invitesData := UserInvites{}
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
		Logger().Print(err)
		return
	}
	invites = invitesData.Rejected
	return
}

func (u *User) CancelInvitation(user *User) error {
	return nil
}

func (u *User) SeeFriends() (friends []primitive.ObjectID, err error) {
	friendData := Friends{}
	err = MongoConn().Collection(FriendsCollection).FindOne(
		context.Background(),
		bson.M{UserIdField: u.ID}).
		Decode(&friendData)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			err = fmt.Errorf("friends data not found for user %s", u.ID.String())
		} else {
			err = fmt.Errorf("friends data fetch failed for user %s: %s", u.ID.String(), err)
		}
		Logger().Print(err)
		return
	}
	friends = friendData.FriendIDs
	return
}

// shows the basic details about a given user based on the object
func UserProfile(userID primitive.ObjectID) string {
	user, _ := GetUserByID(userID)
	return fmt.Sprintf("%s %s : %s", user.FirstName, user.LastName, user.Email)
}

func (u *User) String() string {
	return u.ID.String()
}

func (u *User) ShowChat(friendID primitive.ObjectID) (content string, timestamp time.Time) {
	friend, _ := GetUserByID(friendID)
	content = fmt.Sprintf("\n\n******************* Chat: %s %s *****************\n\n",
		friend.FirstName, friend.LastName) // TODO: Use buffers instead
	chat, err := GetChatByUserIDs(u.ID, friendID)
	if err != nil {
		Logger().Print(err)
		return
	}
	for _, msg := range chat.Messages {
		content += fmt.Sprintf(PrintMessage(msg, u, friend) + "\n")
		timestamp = msg.Timestamp
	}
	return
}

func ValidUserEmail(email string) bool {
	return regexp.MustCompile(ValidEmailRegex).MatchString(email)
}
