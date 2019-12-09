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
	ID        primitive.ObjectID `bson:"_id" json:"-"`
	UserId    primitive.ObjectID `bson:"user_id" json:"user_id"`     // object ID of user
	Sent      []Invite           `bson:"send" json:"send"`           // active sent request by user
	Received  []Invite           `bson:"received" json:"received"`   // active received request by user
	Accepted  []Invite           `bson:"accepted" json:"accepted"`   // accepted requests by user
	Rejected  []Invite           `bson:"rejected" json:"rejected"`   // rejected requests by user
	Cancelled []Invite           `bson:"cancelled" json:"cancelled"` // cancelled sent requests by user
}

func CreateUserInvitesData(userId interface{}) (userInvitesDataId string, err error) {
	userInvitesData := &UserInvitesData{
		UserId:    userId.(primitive.ObjectID),
		Sent:      make([]Invite, 0),
		Received:  make([]Invite, 0),
		Accepted:  make([]Invite, 0),
		Rejected:  make([]Invite, 0),
		Cancelled: make([]Invite, 0),
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
		userInvitesDataId = fmt.Sprintf("%s", res.InsertedID)
	}
	return
}

func GetUserInvitesData(userId string) (userInvitesData *UserInvitesData, err error) {
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

func GetReceivedInvitations(userId string) (invites []Invite, err error) {
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

func GetSentInvitations(userId string) (invites []Invite, err error) {
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

func GetAcceptedInvitations(userEmail string) (invites []Invite, err error) {
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
		invites = userInvitesData.Sent
	}
	return
}

func GetRejectedInvitations(userId string) (invites []Invite, err error) {
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
