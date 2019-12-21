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

type InviteType string

const (
	Sent      InviteType = "sent"
	Received  InviteType = "received"
	Accepted  InviteType = "accepted"
	Rejected  InviteType = "rejected"
	Cancelled InviteType = "cancelled"
)

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
	user.Password, err = GenerateHash(user.Password)
	if err != nil {
		return
	}
	user.LoggedIn = true // as user is just created, he becomes online, until he quits the session
	userMap, err := GetMap(user)
	if err != nil {
		return
	}
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
			_ = session.AbortTransaction(sc)
			Logger().Printf("error while setting up invites data for user%s: %s", userId, err)
			err = NoDocUpdate
			return er
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
		Logger().Printf("no user found with email: %s", email)
		// no changes in error so that it can be used to verify unique email ID before insertion
	} else if err != nil {
		Logger().Printf("decoding(unmarshal) user fetch result for email %s failed: %s", email, err)
		err = FetchUserFailed
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
			Logger().Printf("no user found with ID: %s", objectID.String())
		} else {
			Logger().Printf("decoding(unmarshal) user fetch result for email %s failed: %s", objectID.String(), err)
		}
	}
	return
}

// raises an error if authentication fails due to any reason, including password mismatch
func (u *User) LoginUser(password string) (lastLogin string, err error) {
	fetchDBUser, err := GetUserByEmail(u.Email)
	if err != nil {
		Logger().Printf("authenticate u failed: %s", err)
		return
	}
	err = MatchHashAndPlainText(fetchDBUser.Password, password)
	if err != nil {
		Logger().Print(err)
		return
	}

	result, err := MongoConn().Collection(UserCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   UserEmailField,
				Value: u.Email,
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
		Logger().Printf("error while logging out u %s: %s", u.Email, err)
		return
	} else if result.MatchedCount != 1 {
		Logger().Printf("error while logging out %s as no doc updated for", u.Email)
		err = NoDocUpdate
		return
	}

	u.ID = fetchDBUser.ID
	u.FirstName = fetchDBUser.FirstName
	u.LastName = fetchDBUser.LastName
	u.Email = fetchDBUser.Email
	u.Password = fetchDBUser.Password
	u.LastLogin = fetchDBUser.LastLogin
	u.LoggedIn = fetchDBUser.LoggedIn
	u.InvitesId = fetchDBUser.InvitesId
	lastLogin = fetchDBUser.LastLogin.Format(time.RFC3339)
	return
}

func (u *User) ExistingUser() (exists bool) {
	_, err := GetUserByEmail(u.Email) // if u not exists, it will throw an error
	if err == mongo.ErrNoDocuments {
		return
	}
	if err != nil { // some other error occurred
		Logger().Printf("u email unique check failed: %s", err)
		return
	}
	exists = true
	return
}

func (u *User) UpdatePassword(newEncryptedPassword string) (err error) {
	result, err := MongoConn().Collection(UserCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   UserEmailField,
				Value: u.Email,
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
		Logger().Printf("password update failed for u %s: %s", u.Email, err)
	} else {
		Logger().Printf("password update successful for u %s: %+v", u.Email, result)
	}
	return
}

func (u *User) UpdateName(firstName, lastName string) (err error) {
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
		Logger().Println("nothing to update as both firstName and lastName are blank")
		return
	}
	result, err := MongoConn().Collection(UserCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   UserEmailField,
				Value: u.Email,
			},
		},
		bson.D{{Key: MongoSetOperator, Value: updatedDoc}},
	)
	if err != nil {
		Logger().Printf("name update failed for u %s: %s", u.Email, err)
	} else {
		Logger().Printf("name update successful for u %s: %+v", u.Email, result)
	}
	return
}

func (u *User) SeeOnlineFriends() (onlineFriends []string, err error) {
	//fetchedUser := &User{}
	//MongoConn().Collection(UserCollection).FindOne(
	//	context.Background(),
	//	bson.M{
	//		UserEmailField: u.Email,
	//	}).Decode(fetchedUser)
	//_, err = GetUserInvitesData(fetchedUser.InvitesId)
	//if err != nil {
	//	return
	//}
	//friendEmails, err := GetAcceptedInvitations(u.Email)
	//if err != nil {
	//	reason := fmt.Sprintf("error while fetching u %s accepted invitations: %s", u.Email, err)
	//	Logger().Print(reason)
	//}
	//onlineFriends = make([]string, 5)
	//for _, acceptedInvite := range friendEmails {
	//	friend := &User{}
	//	if acceptedInvite.Sender != u.Email {
	//		MongoConn().Collection(UserCollection).FindOne(
	//			context.Background(),
	//			bson.M{
	//				UserEmailField: acceptedInvite.Sender,
	//				UserLoggedIn:   true,
	//			}).Decode(friend)
	//	}
	//	if friend.Password != "" { // a u is found
	//		onlineFriends = append(onlineFriends, fmt.Sprintf("%s %s: %s", friend.FirstName, friend.LastName,
	//			friend.Email))
	//	}
	//}
	return
}

func (u *User) Logout() (err error) {
	result, err := MongoConn().Collection(UserCollection).UpdateOne(
		context.Background(),
		bson.D{
			{Key: UserEmailField, Value: u.Email},
		},
		bson.D{
			{Key: MongoSetOperator, Value: bson.D{{Key: UserLoggedIn, Value: false}}},
		},
	)
	if err != nil {
		Logger().Printf("error while logging out u %s: %s", u.Email, err)
		return
	} else if result.ModifiedCount != 1 {
		Logger().Printf("error while logging out u %s", u.Email)
		err = NoDocUpdate
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
			Logger().Printf("sending invitation failed from %s to %s as no doc modified", u.Email, recv.Email)
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
	// Delete the sent request from u's invites data (THINK OF SOFT DELETE) - Done
	// Add u as u's friend
	// Add u as u's
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
				Logger().Printf("invite data not found for invite accepting u %s", u.ID.String())
			} else {
				Logger().Printf("invite data fetch failed for invite accepting u %s: %s", u.ID.String(), er)
			}
			return
		} else if res.ModifiedCount != 1 {
			Logger().Printf("invite data not updated for invite accepting u %s", u.ID.String())
			er = NoDocUpdate
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
				Logger().Printf("invite data not found for invite sending u %s", u.ID.String())
			} else {
				Logger().Printf("invite data fetch failed for invite sending u %s: %s", u.ID.String(), er)
			}
			return
		} else if res.ModifiedCount != 1 {
			Logger().Printf("invite data not updated for invite sending u %s", userID.String())
			err = NoDocUpdate
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
			Logger().Printf("error while adding %s as friend for %s: %s", userID.String(), u.ID.String(), er)
			return
		} else if res.ModifiedCount+res.UpsertedCount != 1 {
			Logger().Printf("document not created/updated to add %s as friend for %s", userID.String(), u.ID.String())
			err = NoDocUpdate
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
			Logger().Printf("error while adding %s as friend for %s: %s", u.ID.String(), userID.String(), err)
			return
		} else if res.ModifiedCount+res.UpsertedCount != 1 {
			Logger().Printf("document not created/updated to add %s as friend for %s", u.ID.String(),
				userID.String())
			err = NoDocUpdate
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

func (u *User) GetSentInvitations() ([]primitive.ObjectID, error) {
	return u.getInvitations(Sent)
}

func (u *User) GetReceivedInvitations() ([]primitive.ObjectID, error) {
	return u.getInvitations(Received)
}

func (u *User) GetCanceledSentInvitations() ([]primitive.ObjectID, error) {
	return u.getInvitations(Cancelled)
}

func (u *User) GetAcceptedInvitations() ([]primitive.ObjectID, error) {
	return u.getInvitations(Accepted)
}

func (u *User) GetRejectedInvitations() ([]primitive.ObjectID, error) {
	return u.getInvitations(Rejected)
}

func (u *User) CancelInvitation(user *User) error {
	Logger().Printf("user %s invitation cancelled", user.Email)
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
			Logger().Printf("friends data not found for u %s", u.ID.String())
		} else {
			Logger().Printf("friends data fetch failed for u %s: %s", u.ID.String(), err)
		}
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

func (u *User) getInvitations(invType InviteType) (invites []primitive.ObjectID, err error) {
	invitesData := UserInvites{}
	err = MongoConn().Collection(UserInvitesCollection).FindOne(
		context.Background(),
		bson.M{UserIdField: u.ID}).
		Decode(&invitesData)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			Logger().Printf("invite data not found for u %s", u.ID.String())
		} else {
			Logger().Printf("invite data fetch failed for u %s: %s", u.ID.String(), err)
		}
		return
	}
	switch invType {
	case Sent:
		invites = invitesData.Sent
	case Received:
		invites = invitesData.Received
	case Accepted:
		invites = invitesData.Accepted
	case Rejected:
		invites = invitesData.Rejected
	case Cancelled:
		invites = invitesData.Cancelled
	default:
		err = fmt.Errorf("invalid invite type %s", invType)
	}
	return
}
