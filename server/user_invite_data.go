package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/fatih/structs"
	"github.com/mongodb/mongo-go-driver/bson"
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
	UserIdField           = "userid"
)

// stores reference for all the invitations sent and received associated with a user
type UserInvitesData struct {
	UserId    string   // object ID of user
	Sent      []Invite // active sent request by user
	Received  []Invite // active received request by user
	Accepted  []Invite // accepted requests by user
	Rejected  []Invite // rejected requests by user
	Cancelled []Invite // cancelled sent requests by user
}

func CreateUserInvitesData(userId string) (userInvitesDataId string, err error) {
	userInvitesData := &UserInvitesData{
		UserId:    userId,
		Sent:      make([]Invite, 0),
		Received:  make([]Invite, 0),
		Accepted:  make([]Invite, 0),
		Rejected:  make([]Invite, 0),
		Cancelled: make([]Invite, 0),
	}
	userInvitesDataMap := MapLowercaseKeys(structs.Map(*userInvitesData))
	res, err := GetDBConn().Collection(UserInvitesCollection).InsertOne(
		context.Background(),
		userInvitesDataMap)
	if err != nil {
		reason := fmt.Sprintf("error creating user %s invites data: %s", userId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	} else {
		userInvitesDataId = res.InsertedID.(string)
	}
	return
}

func GetUserInvitesData(userId string) (userInvitesData *UserInvitesData, err error) {
	userInvitesData = &UserInvitesData{}
	err = GetDBConn().Collection(InviteCollection).FindOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(UserIdField, userId),
		),
	).Decode(userInvitesData)
	if err != nil {
		reason := fmt.Sprintf("error while fetching user invites data for user %s: %s", userId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	}
	return
}

func GetReceivedInvitations(userId string) (invites []Invite, err error) {
	userInvitesData := &UserInvitesData{}
	err = GetDBConn().Collection(UserInvitesCollection).FindOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, userId),
		),
	).Decode(userInvitesData)
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
	err = GetDBConn().Collection(UserInvitesCollection).FindOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, userId),
		),
	).Decode(userInvitesData)
	if err != nil {
		reason := fmt.Sprintf("error while fetching received invitations for user %s: %s", userId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	} else {
		invites = userInvitesData.Sent
	}
	return
}

func GetAcceptedInvitations(userId string) (invites []Invite, err error) {
	userInvitesData := &UserInvitesData{}
	err = GetDBConn().Collection(UserInvitesCollection).FindOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, userId),
		),
	).Decode(userInvitesData)
	if err != nil {
		reason := fmt.Sprintf("error while fetching received invitations for user %s: %s", userId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	} else {
		invites = userInvitesData.Sent
	}
	return
}

func GetRejectedInvitations(userId string) (invites []Invite, err error) {
	userInvitesData := &UserInvitesData{}
	err = GetDBConn().Collection(UserInvitesCollection).FindOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, userId),
		),
	).Decode(userInvitesData)
	if err != nil {
		reason := fmt.Sprintf("error while fetching received invitations for user %s: %s", userId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	} else {
		invites = userInvitesData.Sent
	}
	return
}
