package service

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	UserInvitesCollection = "user_invites"
)

// user invites data fields
const (
	SentInvitesField      = "sent"
	ReceivedInvitesField  = "received"
	AcceptedInvitesField  = "accepted"
	RejectedInvitesField  = "rejected"
	CancelledInvitesField = "cancelled"
	UserIdField           = "user_id"
)

// stores reference for all the invitations sent and received associated with a user
type UserInvitesData struct {
	ID        primitive.ObjectID   `bson:"_id" json:"-"`
	UserID    primitive.ObjectID   `bson:"user_id" json:"user_id"`     // object ID of user
	Sent      []primitive.ObjectID `bson:"sent" json:"sent"`           // active sent request by user
	Received  []primitive.ObjectID `bson:"received" json:"received"`   // active received request by user
	Accepted  []primitive.ObjectID `bson:"accepted" json:"accepted"`   // accepted requests by user
	Rejected  []primitive.ObjectID `bson:"rejected" json:"rejected"`   // rejected requests by user
	Cancelled []primitive.ObjectID `bson:"cancelled" json:"cancelled"` // cancelled sent requests by user
}

func CreateUserInvitesData(userId interface{}) (userInvitesDataId interface{}, err error) {
	userInvitesData := &UserInvitesData{
		UserID:    userId.(primitive.ObjectID),
		Sent:      make([]primitive.ObjectID, 0),
		Received:  make([]primitive.ObjectID, 0),
		Accepted:  make([]primitive.ObjectID, 0),
		Rejected:  make([]primitive.ObjectID, 0),
		Cancelled: make([]primitive.ObjectID, 0),
	}
	userInvitesDataMap := GetMap(*userInvitesData)
	userInvitesDataMap["user_id"] = userId.(primitive.ObjectID)
	res, err := MongoConn().Collection(UserInvitesCollection).InsertOne(
		context.Background(),
		userInvitesDataMap)
	if err != nil {
		reason := fmt.Sprintf("error creating user %s invites data: %s", userId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	} else {
		userInvitesDataId = res.InsertedID
	}
	return
}

func GetUserInvitesData(userId primitive.ObjectID) (userInvitesData *UserInvitesData, err error) {
	userInvitesData = &UserInvitesData{}
	err = MongoConn().Collection(InviteCollection).FindOne(
		context.Background(),
		bson.M{
			UserIdField: userId,
		}).Decode(userInvitesData)
	if err != nil {
		reason := fmt.Sprintf("error while fetching user invites data for user %s: %s", userId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	}
	return
}

func GetReceivedInvitations(userId primitive.ObjectID) (invites []primitive.ObjectID, err error) {
	userInvitesData := &UserInvitesData{}
	err = MongoConn().Collection(UserInvitesCollection).FindOne(
		context.Background(),
		bson.M{
			ObjectID: userId,
		}).Decode(userInvitesData)
	if err != nil {
		reason := fmt.Sprintf("error while fetching received invitations for user %s: %s", userId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	} else {
		invites = userInvitesData.Received
	}
	return
}

func GetSentInvitations(userId primitive.ObjectID) (invites []primitive.ObjectID, err error) {
	userInvitesData := &UserInvitesData{}
	err = MongoConn().Collection(UserInvitesCollection).FindOne(
		context.Background(),
		bson.M{
			ObjectID: userId,
		}).Decode(userInvitesData)
	if err != nil {
		reason := fmt.Sprintf("error while fetching received invitations for user %s: %s", userId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	} else {
		invites = userInvitesData.Sent
	}
	return
}

func GetAcceptedInvitations(userEmail string) (invites []primitive.ObjectID, err error) {
	userInvitesData := &UserInvitesData{}
	err = MongoConn().Collection(UserInvitesCollection).FindOne(
		context.Background(),
		bson.M{
			UserEmailField: userEmail,
		}).Decode(userInvitesData)
	if err != nil {
		reason := fmt.Sprintf("error while fetching received invitations for user %s: %s", userEmail, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	} else {
		invites = userInvitesData.Accepted
	}
	return
}

func GetRejectedInvitations(userId primitive.ObjectID) (invites []primitive.ObjectID, err error) {
	userInvitesData := &UserInvitesData{}
	err = MongoConn().Collection(UserInvitesCollection).FindOne(
		context.Background(),
		bson.M{
			ObjectID: userId,
		}).Decode(userInvitesData)
	if err != nil {
		reason := fmt.Sprintf("error while fetching received invitations for user %s: %s", userId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	} else {
		invites = userInvitesData.Rejected
	}
	return
}

func (u *UserInvitesData) String() string {
	return u.ID.String()
}