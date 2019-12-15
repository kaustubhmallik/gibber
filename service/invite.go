package service

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"time"
)

const (
	InviteCollection = "invites"
)

// invite data fields
const (
	InviteSenderField   = "sender"
	InviteReceiverField = "receiver"
	InviteStateField    = "state"
)

// invite states
const (
	ActiveInvite    = "active"
	AcceptedInvite  = "accepted"
	RejectedInvite  = "rejected"
	CancelledInvite = "cancelled"
)

// invite is a single invitation initiated from a sender for a receiver
type Invite struct {
	Sender   string // sender User Email
	Receiver string // receiver User Email
	State    string // active, accepted, rejected
}

func GetInvitation(inviteID string) (invite *Invite, err error) {
	invite = &Invite{}
	err = MongoConn().Collection(InviteCollection).FindOne(
		context.Background(),
		bson.M{UserIdField: inviteID},
	).Decode(invite)
	if err != nil {
		reason := fmt.Sprintf("error while fetching invites data for inviteID %s: %s", inviteID, err)
		Logger().Println(reason)
		err = errors.New(reason)
	}
	return
}

// a user can see her active invitations and take an action on it i.e. accept/reject it
// once he/she takes an action, it is pushed to the inactive invitations with the added
// details of action taken
func AlreadyConnected(senderUserId, receiverUserId string) (err error) {
	// check if the user doesn't have an active sent invitation to invitedUser
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Minute)
	count, err := MongoConn().Collection(InviteCollection).CountDocuments(
		ctx,
		bson.M{
			ObjectID:         senderUserId,
			SentInvitesField: receiverUserId,
		},
	)
	if err != nil {
		reason := fmt.Sprintf("error while checking for the user %s already has an active sent invitation to %s",
			senderUserId, receiverUserId)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	} else if count == 1 {
		reason := fmt.Sprintf("user %s already has an active sent invitation to %s", senderUserId, receiverUserId)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}

	//check if the user doesn't have an active received invitation to invitedUser
	ctx, _ = context.WithTimeout(context.Background(), 30*time.Minute)
	count, err = MongoConn().Collection(InviteCollection).CountDocuments(
		ctx,
		bson.M{
			ObjectID:             senderUserId,
			ReceivedInvitesField: receiverUserId,
		})
	if err != nil {
		reason := fmt.Sprintf("error while checking for the user %s already has an active received invitation to %s",
			senderUserId, receiverUserId)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	} else if count == 1 {
		reason := fmt.Sprintf("user %s already has an active received invitation to %s", senderUserId, receiverUserId)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}

	//check if the user doesn't have an accepted invitation from invitedUser
	ctx, _ = context.WithTimeout(context.Background(), 30*time.Minute)
	count, err = MongoConn().Collection(InviteCollection).CountDocuments(
		ctx,
		bson.M{
			ObjectID:             senderUserId,
			AcceptedInvitesField: receiverUserId,
		})
	if err != nil {
		reason := fmt.Sprintf("error while checking for the user %s already has an accepted invitation to %s",
			senderUserId, receiverUserId)
		Logger().Println(reason)
		err = errors.New(reason)
	} else if count == 1 {
		reason := fmt.Sprintf("user %s already has an accepted invitation to %s", senderUserId, receiverUserId)
		Logger().Println(reason)
		err = errors.New(reason)
	}
	return
}

func SendInvitation(senderUserId, invitationId string) (err error) {
	// add an invite to the sent user's invites array
	result, err := MongoConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   ObjectID,
				Value: senderUserId,
			},
		},
		bson.D{
			{
				Key:   SentInvitesField,
				Value: invitationId,
			},
		})
	if err != nil {
		reason := fmt.Sprintf("error while adding invitation %s into %s's active sent invitation: %s", invitationId,
			senderUserId, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while adding invitation %s into %s's active sent invitation: %s",
			invitationId, senderUserId, err)
		Logger().Println(reason)
		err = errors.New(reason)
	}
	return
}

func CancelInvitation(senderUserId, invitationId string) (err error) {
	// change the state of invitation from active to cancelled
	result, err := MongoConn().Collection(InviteCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   ObjectID,
				Value: invitationId,
			},
		},
		bson.D{
			{
				Key:   InviteStateField,
				Value: CancelledInvite,
			},
		})
	if err != nil {
		reason := fmt.Sprintf("error while changing invitation %s state to %s: %s", invitationId,
			CancelledInvite, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while changing invitation %s  state to %s's : %s",
			invitationId, CancelledInvite, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}

	// remove invite from the sent user's invites array
	result, err = MongoConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   ObjectID,
				Value: senderUserId,
			},
		},
		bson.D{
			{
				Key:   SentInvitesField,
				Value: invitationId,
			},
		})

	if err != nil {
		reason := fmt.Sprintf("error while adding invitation %s into %s's active sent invitation: %s", invitationId,
			senderUserId, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while adding invitation %s into %s's active sent invitation: %s",
			invitationId, senderUserId, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}

	// add invite to the user's cancelled invites array
	result, err = MongoConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   ObjectID,
				Value: senderUserId,
			},
		},
		bson.D{
			{
				Key:   CancelledInvitesField,
				Value: invitationId,
			},
		})
	if err != nil {
		reason := fmt.Sprintf("error while adding invitation %s into %s's cancelled invitation: %s", invitationId,
			senderUserId, err)
		Logger().Println(reason)
		err = errors.New(reason)
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while adding invitation %s into %s's cancelled invitation: %s",
			invitationId, senderUserId, err)
		Logger().Println(reason)
		err = errors.New(reason)
	}
	return
}

func ReceiveInvitation(receiverUserId, invitationId string) (err error) {
	// add an invite to the receiver user's received invites array
	result, err := MongoConn().Collection(InviteCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   ObjectID,
				Value: receiverUserId,
			},
		},
		bson.D{
			{
				Key:   ReceivedInvitesField,
				Value: invitationId,
			},
		})
	if err != nil {
		reason := fmt.Sprintf("error while adding invitation %s into %s's active received invitation: %s",
			invitationId, receiverUserId, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while adding invitation %s into %s's received sent "+
			"invitation: %s", invitationId, receiverUserId, err)
		Logger().Println(reason)
		err = errors.New(reason)
	}
	return
}

func AcceptInvitation(receiverUserID, invitationID string) (err error) {
	// change the state of invitation from active to accepted
	result, err := MongoConn().Collection(InviteCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   UserEmailField,
				Value: invitationID,
			},
		},
		bson.D{
			{
				Key:   InviteStateField,
				Value: AcceptedInvite,
			},
		},
	)
	if err != nil {
		reason := fmt.Sprintf("error while changing invitation %s state to %s: %s", invitationID,
			CancelledInvite, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while changing invitation %s  state to %s's : %s",
			invitationID, CancelledInvite, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}

	// move the invite from receiver user's received invites to accepted invites
	result, err = MongoConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   ObjectID,
				Value: receiverUserID,
			},
		},
		bson.D{
			{
				Key:   AcceptedInvitesField,
				Value: invitationID,
			},
		},
	)
	if err != nil {
		reason := fmt.Sprintf("error while adding %s into %s's active accepted invitation: %s", receiverUserID,
			receiverUserID, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while adding %s into %s's received accepted invitation: %s",
			receiverUserID, receiverUserID, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}

	result, err = MongoConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   UserEmailField,
				Value: receiverUserID,
			},
		},
		bson.D{
			{
				Key:   ReceivedInvitesField,
				Value: invitationID,
			},
		},
	)
	if err != nil {
		reason := fmt.Sprintf("error while removing %s from %s's received invitation: %s", receiverUserID,
			receiverUserID, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while removing %s from %s's received invitation: %s",
			receiverUserID, receiverUserID, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}

	// Changing the sender user's invitation data too

	invitationData, err := GetInvitation(invitationID)
	if err != nil {
		reason := fmt.Sprintf("error while fetching invitation %s data: %s", invitationID, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}

	// move the invite from sender user's sent invites to accepted invites
	result, err = MongoConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   ObjectID,
				Value: invitationData.Sender,
			},
		},
		bson.D{
			{
				Key:   AcceptedInvitesField,
				Value: invitationID,
			},
		},
	)
	if err != nil {
		reason := fmt.Sprintf("error while adding %s into %s's accepted invitation: %s", receiverUserID,
			receiverUserID, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while adding %s into %s's accepted invitation: %s",
			receiverUserID, receiverUserID, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}

	result, err = MongoConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   ObjectID,
				Value: invitationData.Sender,
			},
		},
		bson.D{
			{
				Key:   SentInvitesField,
				Value: invitationID,
			},
		},
	)
	if err != nil {
		reason := fmt.Sprintf("error while removing %s from %s's sent invitation: %s", receiverUserID,
			receiverUserID, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while removing %s from %s's sent invitation: %s",
			receiverUserID, receiverUserID, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}
	return
}

func RejectInvitation(receiverUserId, invitationId string) (err error) {
	// change the state of invitation from active to accepted
	result, err := MongoConn().Collection(InviteCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   ObjectID,
				Value: invitationId,
			},
		},
		bson.D{
			{
				Key:   InviteStateField,
				Value: RejectedInvite,
			},
		},
	)
	if err != nil {
		reason := fmt.Sprintf("error while changing invitation %s state to %s: %s", invitationId,
			CancelledInvite, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while changing invitation %s  state to %s's : %s",
			invitationId, CancelledInvite, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}

	// move the invite from receiver user's received invites to rejected invites
	result, err = MongoConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   ObjectID,
				Value: receiverUserId,
			},
		},
		bson.D{
			{
				Key:   RejectedInvitesField,
				Value: invitationId,
			},
		},
	)
	if err != nil {
		reason := fmt.Sprintf("error while adding %s into %s's rejected invitation: %s", receiverUserId,
			receiverUserId, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while adding %s into %s's rejected invitation: %s",
			receiverUserId, receiverUserId, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}

	result, err = MongoConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   UserEmailField,
				Value: receiverUserId,
			},
		},
		bson.D{
			{
				Key:   ReceivedInvitesField,
				Value: invitationId,
			},
		},
	)
	if err != nil {
		reason := fmt.Sprintf("error while removing %s from %s's received invitation: %s", receiverUserId,
			receiverUserId, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while removing %s from %s's received invitation: %s",
			receiverUserId, receiverUserId, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}

	// Changing the sender user's invitation data too
	invitationData, err := GetInvitation(invitationId)
	if err != nil {
		reason := fmt.Sprintf("error while fetching invitation %s data: %s", invitationId, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}

	// move the invite from sender user's sent invites to accepted invites
	result, err = MongoConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   ObjectID,
				Value: invitationData.Sender,
			},
		},
		bson.D{
			{
				Key:   RejectedInvitesField,
				Value: invitationId,
			},
		},
	)
	if err != nil {
		reason := fmt.Sprintf("error while adding %s into %s's accepted invitation: %s", receiverUserId,
			receiverUserId, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while adding %s into %s's accepted invitation: %s",
			receiverUserId, receiverUserId, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}

	result, err = MongoConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.D{
			{
				Key:   ObjectID,
				Value: invitationData.Sender,
			},
		},
		bson.D{
			{
				Key:   SentInvitesField,
				Value: invitationId,
			},
		},
	)
	if err != nil {
		reason := fmt.Sprintf("error while removing %s from %s's sent invitation: %s", receiverUserId,
			receiverUserId, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while removing %s from %s's sent invitation: %s",
			receiverUserId, receiverUserId, err)
		Logger().Println(reason)
		err = errors.New(reason)
		return
	}
	return
}
