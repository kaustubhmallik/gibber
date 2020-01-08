package user

import (
	"context"
	"errors"
	"fmt"
	"gibber/datastore"
	"gibber/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
	"regexp"
	"time"
)

// user document collection name and fields
const (
	userCollection     = "users"
	userFirstNameField = "first_name"
	userLastNameField  = "last_name"
	userEmailField     = "email"
	userLoggedIn       = "logged_in"
	lastLogin          = "last_login"
	userPasswordField  = "password"
	invitesDataField   = "invites_data_id"
)

const validEmailRegex = `^[\w\.=-]+@[\w\.-]+\.[\w]{2,3}$`

// invitation types
const (
	sent      inviteType = "sent"
	received  inviteType = "received"
	accepted  inviteType = "accepted"
	rejected  inviteType = "rejected"
	cancelled inviteType = "cancelled"
)

// user invite errors
var (
	fetchUserFailed   = errors.New("fetch user details failed")
	invalidInviteType = errors.New("invalid invite type")
)

// an enum to restrict invitation types
type inviteType string

// User captures the details about a client connected to the service
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

// CreateUser create a new user with given user details
func CreateUser(user *User) (userId interface{}, err error) {
	var fetchUser *User
	if user.existingUser() {
		reason := fmt.Sprintf("user %#v already exist with email %s", fetchUser, user.Email) // passed email userId should be unique
		log.Logger().Printf(reason)
		err = errors.New(reason)
		return
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	user.Password = string(hashedPassword)
	user.LoggedIn = true // as user is just created, he becomes online, until he quits the session
	userMap, _ := getMap(user)
	userMap["last_login"] = time.Now().UTC()

	session, err := datastore.MongoConn().Client().StartSession()
	if err != nil {
		log.Logger().Printf("initializing mongo session failed: %s", err)
		return
	}

	err = session.StartTransaction()
	if err != nil {
		log.Logger().Printf("initializing mongo transaction failed: %s", err)
		return
	}

	err = mongo.WithSession(context.Background(), session, func(sc mongo.SessionContext) (er error) {
		// create user
		res, er := datastore.MongoConn().Collection(userCollection).InsertOne(sc, userMap)
		if er != nil {
			_ = session.AbortTransaction(sc)
			er = fmt.Errorf("error while creating new user %#v: %s", userMap, er)
			log.Logger().Print(er)
			return
		} else {
			userId = res.InsertedID
			user.ID = res.InsertedID.(primitive.ObjectID)
			log.Logger().Printf("user %#v successfully created with userId: %v", userMap, res)
		}

		// create user_invite
		var invitesId primitive.ObjectID
		invitesDataId, er := createUserInvitesData(userId, sc, datastore.MongoConn().Collection(userInvitesCollection))
		invitesId = invitesDataId.(primitive.ObjectID)
		if er != nil {
			_ = session.AbortTransaction(sc)
			log.Logger().Printf("error creating user invite: %s", er)
			return er
		}

		// update invite_user doc ID in user's doc
		updateRes, er := datastore.MongoConn().Collection(userCollection).UpdateOne(
			sc,
			bson.M{datastore.ObjectID: userId},
			bson.D{{
				Key: datastore.MongoSetOperator, Value: bson.D{{Key: invitesDataField, Value: invitesId}},
			}})
		if er != nil || updateRes.ModifiedCount != 1 {
			_ = session.AbortTransaction(sc)
			log.Logger().Printf("error while setting up invites data for user%s: %s", userId, err)
			err = datastore.NoDocUpdate
			return er
		}

		// commit transaction
		if er := session.CommitTransaction(sc); er != nil {
			log.Logger().Printf("committing mongo transaction failed: %s", er)
			er = session.AbortTransaction(sc)
			if er != nil {
				log.Logger().Printf("aborting mongo transaction failed: %s", er)
			}
		}
		return nil
	})

	session.EndSession(context.Background())
	return
}

// GetUserByEmail gets the details of a user by email
func GetUserByEmail(email string) (user *User, err error) {
	collection := datastore.MongoConn().Collection(userCollection)
	user = &User{}
	err = collection.FindOne(context.Background(), bson.M{userEmailField: email}).Decode(user)
	if err == mongo.ErrNoDocuments {
		log.Logger().Printf("no user found with email: %s", email)
		// no changes in error so that it can be used to verify unique email ID before insertion
	} else if err != nil {
		log.Logger().Printf("decoding(unmarshal) user fetch result for email %s failed: %s", email, err)
		err = fetchUserFailed
	}
	return
}

// GetUserByEmail gets the details of a user by ID (object ID)
func GetUserByID(objectID primitive.ObjectID) (user *User, err error) {
	user = &User{}
	err = datastore.MongoConn().
		Collection(userCollection).
		FindOne(context.Background(), bson.M{datastore.ObjectID: objectID}).
		Decode(user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Logger().Printf("no user found with ID: %s", objectID.String())
		} else {
			log.Logger().Printf("decoding(unmarshal) user fetch result for email %s failed: %s", objectID.String(), err)
		}
	}
	return
}

// LoginUser logs in a given user with the given password. In case of successful login, it returns
// the last login time of the user. In case of password mismatch or any other issue, an error will be raised.
func (u *User) LoginUser(password string) (lastLoginTime string, err error) {
	fetchDBUser, err := GetUserByEmail(u.Email)
	if err != nil {
		log.Logger().Printf("authenticate u failed: %s", err)
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(fetchDBUser.Password), []byte(password))
	if err != nil {
		log.Logger().Print(err)
		return
	}

	result, err := datastore.MongoConn().Collection(userCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   userEmailField,
				Value: u.Email,
			},
		},
		bson.D{
			{
				Key: datastore.MongoSetOperator,
				Value: bson.D{
					{
						Key:   userLoggedIn,
						Value: true,
					},
					{
						Key:   lastLogin,
						Value: time.Now().UTC(),
					},
				},
			},
		},
	)
	if err != nil {
		log.Logger().Printf("error while logging out u %s: %s", u.Email, err)
		return
	} else if result.MatchedCount != 1 {
		log.Logger().Printf("error while logging out %s as no doc updated for", u.Email)
		err = datastore.NoDocUpdate
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
	lastLoginTime = fetchDBUser.LastLogin.Format(time.RFC3339)
	return
}

// UpdatePassword updates the password for the current user
func (u *User) UpdatePassword(newEncryptedPassword string) (err error) {
	result, err := datastore.MongoConn().Collection(userCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   userEmailField,
				Value: u.Email,
			},
		},
		bson.D{
			{
				Key: datastore.MongoSetOperator,
				Value: bson.D{
					{
						Key:   userPasswordField,
						Value: newEncryptedPassword,
					},
				},
			},
		})
	if err != nil {
		log.Logger().Printf("password update failed for u %s: %s", u.Email, err)
	} else {
		log.Logger().Printf("password update successful for u %s: %+v", u.Email, result)
	}
	return
}

// UpdateName updates the first and/or last name of the given user
func (u *User) UpdateName(firstName, lastName string) (err error) {
	var updatedDoc bson.D
	if firstName != "" && lastName != "" {
		updatedDoc = bson.D{
			{
				Key:   userFirstNameField,
				Value: firstName,
			},
			{
				Key:   userLastNameField,
				Value: lastName,
			},
		}
	} else if firstName != "" {
		updatedDoc = bson.D{
			{
				Key:   userFirstNameField,
				Value: firstName,
			},
		}
	} else if lastName != "" {
		updatedDoc = bson.D{
			{
				Key:   userLastNameField,
				Value: lastName,
			},
		}
	} else { // nothing to update
		log.Logger().Println("nothing to update as both firstName and lastName are blank")
		return
	}
	result, err := datastore.MongoConn().Collection(userCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   userEmailField,
				Value: u.Email,
			},
		},
		bson.D{{Key: datastore.MongoSetOperator, Value: updatedDoc}},
	)
	if err != nil {
		log.Logger().Printf("name update failed for u %s: %s", u.Email, err)
	} else {
		log.Logger().Printf("name update successful for u %s: %+v", u.Email, result)
	}
	return
}

// SeeOnlineFriends fetches the details of the users which are currently online
func (u *User) SeeOnlineFriends() (onlineFriends []string, err error) {
	//fetchedUser := &User{}
	//MongoConn().Collection(userCollection).FindOne(
	//	context.Background(),
	//	bson.M{
	//		userEmailField: u.Email,
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
	//		MongoConn().Collection(userCollection).FindOne(
	//			context.Background(),
	//			bson.M{
	//				userEmailField: acceptedInvite.Sender,
	//				userLoggedIn:   true,
	//			}).Decode(friend)
	//	}
	//	if friend.Password != "" { // a u is found
	//		onlineFriends = append(onlineFriends, fmt.Sprintf("%s %s: %s", friend.FirstName, friend.LastName,
	//			friend.Email))
	//	}
	//}
	return
}

// Logout logs out the current user from the service
// TODO: avoid multiple login for a single user
func (u *User) Logout() (err error) {
	result, err := datastore.MongoConn().Collection(userCollection).UpdateOne(
		context.Background(),
		bson.D{
			{Key: userEmailField, Value: u.Email},
		},
		bson.D{
			{Key: datastore.MongoSetOperator, Value: bson.D{{Key: userLoggedIn, Value: false}}},
		},
	)
	if err != nil {
		log.Logger().Printf("error while logging out u %s: %s", u.Email, err)
		return
	} else if result.ModifiedCount != 1 {
		log.Logger().Printf("error while logging out u %s", u.Email)
		err = datastore.NoDocUpdate
		return
	}
	return
}

// SendInvitation sends an invite to a given user
func (u *User) SendInvitation(recv *User) (err error) {
	session, err := datastore.MongoConn().Client().StartSession()
	if err != nil {
		log.Logger().Printf("initializing mongo session failed: %s", err)
		return
	}

	err = session.StartTransaction()
	if err != nil {
		log.Logger().Printf("initializing mongo transaction failed: %s", err)
		return
	}

	err = mongo.WithSession(context.Background(), session, func(sc mongo.SessionContext) (er error) {
		result, er := datastore.MongoConn().Collection(userInvitesCollection).UpdateOne(
			sc,
			bson.D{
				{Key: userIdField, Value: u.ID},
			},
			bson.D{
				{Key: datastore.MongoPushOperator, Value: bson.D{{Key: string(sent), Value: recv.ID}}},
			},
		)
		if er != nil {
			_ = session.AbortTransaction(sc)
			er = fmt.Errorf("sending invitation failed from %s to %s: %s", u.Email, recv.Email, err)
			log.Logger().Println(er)
		} else if result.ModifiedCount != 1 {
			_ = session.AbortTransaction(sc)
			log.Logger().Printf("sending invitation failed from %s to %s as no doc modified", u.Email, recv.Email)
		}

		result, er = datastore.MongoConn().Collection(userInvitesCollection).UpdateOne(
			sc,
			bson.D{
				{Key: userIdField, Value: recv.ID},
			},
			bson.D{
				{Key: datastore.MongoPushOperator, Value: bson.D{{Key: string(received), Value: u.ID}}},
			},
		)
		if er != nil {
			_ = session.AbortTransaction(sc)
			er = fmt.Errorf("sending invitation failed from %s to %s: %s", u.Email, recv.Email, err)
			log.Logger().Print(er)
		} else if result.ModifiedCount != 1 {
			_ = session.AbortTransaction(sc)
			reason := fmt.Sprintf("sending invitation failed from %s to %s as no doc modified", u.Email, recv.Email)
			log.Logger().Println(reason)
		}

		// commit transaction
		if er := session.CommitTransaction(sc); er != nil {
			log.Logger().Printf("committing mongo transaction failed: %s", er)
			er = session.AbortTransaction(sc)
			if er != nil {
				log.Logger().Printf("aborting mongo transaction failed: %s", er)
			}
		}
		return
	})
	return
}

// AddFriend accepts the invite sent from a given user to the current user. After this action,
// they become friends, and can start a chat (conversation)
func (u *User) AddFriend(userID primitive.ObjectID) (err error) {
	// Delete the received request from u's invite data (THINK OF SOFT DELETE) - Done
	// Delete the sent request from u's invites data (THINK OF SOFT DELETE) - Done
	// Add u as u's friend
	// Add u as u's
	session, err := datastore.MongoConn().Client().StartSession()
	if err != nil {
		log.Logger().Printf("initializing mongo session failed: %s", err)
		return
	}

	err = session.StartTransaction()
	if err != nil {
		log.Logger().Printf("initializing mongo transaction failed: %s", err)
		return
	}

	err = mongo.WithSession(context.Background(), session, func(sc mongo.SessionContext) (er error) {
		var res *mongo.UpdateResult
		res, er = datastore.MongoConn().Collection(userInvitesCollection).UpdateOne(
			sc,
			bson.M{userIdField: u.ID},
			bson.D{
				{Key: datastore.MongoPullOperator, Value: bson.D{{Key: string(received), Value: userID}}},
			})
		if er != nil {
			_ = session.AbortTransaction(sc) // ROLLBACK at the earliest to shorten transaction life-cycle
			if er == mongo.ErrNoDocuments {
				log.Logger().Printf("invite data not found for invite accepting u %s", u.ID.String())
			} else {
				log.Logger().Printf("invite data fetch failed for invite accepting u %s: %s", u.ID.String(), er)
			}
			return
		} else if res.ModifiedCount != 1 {
			log.Logger().Printf("invite data not updated for invite accepting u %s", u.ID.String())
			er = datastore.NoDocUpdate
			return
		}

		res, er = datastore.MongoConn().Collection(userInvitesCollection).UpdateOne(
			sc,
			bson.M{userIdField: userID},
			bson.D{
				{Key: datastore.MongoPullOperator, Value: bson.D{{Key: string(sent), Value: u.ID}}},
			})
		if er != nil {
			_ = session.AbortTransaction(sc)
			if er == mongo.ErrNoDocuments {
				log.Logger().Printf("invite data not found for invite sending u %s", u.ID.String())
			} else {
				log.Logger().Printf("invite data fetch failed for invite sending u %s: %s", u.ID.String(), er)
			}
			return
		} else if res.ModifiedCount != 1 {
			log.Logger().Printf("invite data not updated for invite sending u %s", userID.String())
			err = datastore.NoDocUpdate
			return
		}

		// using upsert: true to create the friends document if non-existent
		res, er = datastore.MongoConn().Collection(FriendsCollection).UpdateOne(
			sc,
			bson.M{userIdField: u.ID},
			bson.D{
				{Key: datastore.MongoPushOperator, Value: bson.D{{Key: FriendsField, Value: userID}}},
			},
			options.Update().SetUpsert(true))
		if er != nil {
			_ = session.AbortTransaction(sc) // ROLLBACK at the earliest to shorten transaction life-cycle
			log.Logger().Printf("error while adding %s as friend for %s: %s", userID.String(), u.ID.String(), er)
			return
		} else if res.ModifiedCount+res.UpsertedCount != 1 {
			log.Logger().Printf("document not created/updated to add %s as friend for %s", userID.String(), u.ID.String())
			err = datastore.NoDocUpdate
			return
		}

		// using upsert: true to create the friends document if non-existent
		res, err = datastore.MongoConn().Collection(FriendsCollection).UpdateOne(
			sc,
			bson.M{userIdField: userID},
			bson.D{
				{Key: datastore.MongoPushOperator, Value: bson.D{{Key: FriendsField, Value: u.ID}}},
			},
			options.Update().SetUpsert(true))
		if err != nil {
			_ = session.AbortTransaction(sc) // ROLLBACK at the earliest to shorten transaction life-cycle
			log.Logger().Printf("error while adding %s as friend for %s: %s", u.ID.String(), userID.String(), err)
			return
		} else if res.ModifiedCount+res.UpsertedCount != 1 {
			log.Logger().Printf("document not created/updated to add %s as friend for %s", u.ID.String(),
				userID.String())
			err = datastore.NoDocUpdate
			return
		}

		// commit transaction
		if er := session.CommitTransaction(sc); er != nil {
			log.Logger().Printf("committing mongo transaction failed: %s", er)
			er = session.AbortTransaction(sc)
			if er != nil {
				log.Logger().Printf("aborting mongo transaction failed: %s", er)
			}
		}
		return
	})
	return
}

// GetSentInvitations gets the list of sent invites to the other users, which are not
// yet accepted or rejected i.e. they are still active
func (u *User) GetSentInvitations() ([]primitive.ObjectID, error) {
	return u.getInvitations(sent)
}

// GetReceivedInvitations gets the list of received invites from other users to current user,
// which are not yet accepted or rejected
func (u *User) GetReceivedInvitations() ([]primitive.ObjectID, error) {
	return u.getInvitations(received)
}

// GetCanceledSentInvitations gets the list of cancelled invites sent to other users
func (u *User) GetCanceledSentInvitations() ([]primitive.ObjectID, error) {
	return u.getInvitations(cancelled)
}

// GetAcceptedInvitations gets the list of accepted invites
func (u *User) GetAcceptedInvitations() ([]primitive.ObjectID, error) {
	return u.getInvitations(accepted)
}

// GetRejectedInvitations gets the list of rejected invites
func (u *User) GetRejectedInvitations() ([]primitive.ObjectID, error) {
	return u.getInvitations(rejected)
}

// CancelInvitation cancels the invite received from a given user
func (u *User) CancelInvitation(user *User) error {
	log.Logger().Printf("user %s invitation cancelled", user.Email)
	return nil
}

// SeeFriends fetches the list of userIDs which are friends with current user
func (u *User) SeeFriends() (friends []primitive.ObjectID, err error) {
	friendData := Friends{}
	err = datastore.MongoConn().Collection(FriendsCollection).FindOne(
		context.Background(),
		bson.M{userIdField: u.ID}).
		Decode(&friendData)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Logger().Printf("friends data not found for u %s", u.ID.String())
		} else {
			log.Logger().Printf("friends data fetch failed for u %s: %s", u.ID.String(), err)
		}
		return
	}
	friends = friendData.FriendIDs
	return
}

// String representation of a user
func (u *User) String() string {
	return u.ID.String()
}

// GetChat fetches the chat b/w the current user and the given userID. It returns the actual
// content as a formatted string, and the timestamp of the last message.
func (u *User) GetChat(friendID primitive.ObjectID) (content string, timestamp time.Time) {
	friend, _ := GetUserByID(friendID)
	content = fmt.Sprintf("\n\n******************* chat: %s %s *****************\n\n",
		friend.FirstName, friend.LastName) // TODO: Use buffers instead
	chat, err := getChatByUserIDs(u.ID, friendID, datastore.MongoConn().Collection(chatCollection))
	if err != nil {
		log.Logger().Print(err)
		return
	}
	for _, msg := range chat.Messages {
		var sender string
		if msg.Sender == u.ID {
			sender = "You"
		} else {
			sender = friend.FirstName
		}
		content += fmt.Sprintf(printMessage(msg, sender) + "\n")
		timestamp = msg.Timestamp
	}
	return
}

// getInvitations fetches the invitation of a given type for the user
func (u *User) getInvitations(invType inviteType) (invites []primitive.ObjectID, err error) {
	invitesData := userInvites{}
	err = datastore.MongoConn().Collection(userInvitesCollection).FindOne(
		context.Background(),
		bson.M{userIdField: u.ID}).
		Decode(&invitesData)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Logger().Printf("invite data not found for u %s", u.ID.String())
		} else {
			log.Logger().Printf("invite data fetch failed for u %s: %s", u.ID.String(), err)
		}
		return
	}
	switch invType {
	case sent:
		invites = invitesData.Sent
	case received:
		invites = invitesData.Received
	case accepted:
		invites = invitesData.Accepted
	case rejected:
		invites = invitesData.Rejected
	case cancelled:
		invites = invitesData.Cancelled
	default:
		err = invalidInviteType
	}
	return
}

// existingUser checks that a given user already exists in the system based on the email
func (u *User) existingUser() (exists bool) {
	_, err := GetUserByEmail(u.Email) // if u not exists, it will throw an error
	if err == mongo.ErrNoDocuments {
		return
	}
	if err != nil { // some other error occurred
		log.Logger().Printf("u email unique check failed: %s", err)
		return
	}
	exists = true
	return
}

func ValidUserEmail(email string) bool {
	return regexp.MustCompile(validEmailRegex).MatchString(email)
}

// UserProfile fetches the basic details about a given user based on the ID (objectID)
func UserProfile(userID primitive.ObjectID) (string, error) {
	user, err := GetUserByID(userID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s %s : %s", user.FirstName, user.LastName, user.Email), nil
}
