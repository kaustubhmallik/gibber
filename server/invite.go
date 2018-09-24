package server

import (
	"context"
	"fmt"
	"github.com/fatih/structs"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/pkg/errors"
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
	Sender   string // sender User ObjectId
	Receiver string // receiver User ObjectId
	State    string // active, accepted, rejected
}

//func NewInvitesData() (inviteData InvitesData) {
//	inviteData = InvitesData{
//		Sent:     make([]Invite, 0),
//		Received: make([]Invite, 0),
//		Accepted: make([]Invite, 0),
//		Rejected: make([]Invite, 0),
//	}
//	return
//}

func CreateInvitation(senderUserId, receiverUserId string) (objectId string, err error) {
	invite := &Invite{
		Sender:   senderUserId,
		Receiver: receiverUserId,
		State:    ActiveInvite,
	}
	inviteMap := MapLowercaseKeys(structs.Map(*invite))
	res, err := GetDBConn().Collection(InviteCollection).InsertOne(context.Background(), inviteMap)
	if err != nil {
		reason := fmt.Sprintf("error while creating new user invite %#v: %s", inviteMap, err)
		err = errors.New(reason)
		GetLogger().Printf(reason)
	} else {
		objectId = res.InsertedID.(string) // TODO: Check if it is mostly string (expected), change the id to string, and use reflection on InsertID
		GetLogger().Printf("user invite %#v successfully created with id: %v", inviteMap, res)
	}
	return

}

func GetInvitation(inviteId string) (invite *Invite, err error) {
	invite = &Invite{}
	err = GetDBConn().Collection(InviteCollection).FindOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(UserIdField, inviteId),
		),
	).Decode(invite)
	if err != nil {
		reason := fmt.Sprintf("error while fetching invites data for inviteId %s: %s", inviteId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	}
	return
}

// a user can see her active invitations and take an action on it i.e. accept/reject it
// once he/she takes an action, it is pushed to the inactive invitations with the added
// details of action taken
func AlreadyConnected(senderUserId, receiverUserId string) (err error) {
	// check if the user doesn't have an active sent invitation to invitedUser
	count, err := GetDBConn().Collection(InviteCollection).Count(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, senderUserId),
			bson.EC.String(SentInvitesField, receiverUserId),
		),
	)
	if err != nil {
		reason := fmt.Sprintf("error while checking for the user %s already has an active sent invitation to %s",
			senderUserId, receiverUserId)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	} else if count == 1 {
		reason := fmt.Sprintf("user %s already has an active sent invitation to %s", senderUserId, receiverUserId)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}

	// check if the user doesn't have an active received invitation to invitedUser
	count, err = GetDBConn().Collection(InviteCollection).Count(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, senderUserId),
			bson.EC.String(ReceivedInvitesField, receiverUserId),
		),
	)
	if err != nil {
		reason := fmt.Sprintf("error while checking for the user %s already has an active received invitation to %s",
			senderUserId, receiverUserId)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	} else if count == 1 {
		reason := fmt.Sprintf("user %s already has an active received invitation to %s", senderUserId, receiverUserId)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}

	// check if the user doesn't have an accepted invitation from invitedUser
	count, err = GetDBConn().Collection(InviteCollection).Count(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, senderUserId),
			bson.EC.String(AcceptedInvitesField, receiverUserId),
		),
	)
	if err != nil {
		reason := fmt.Sprintf("error while checking for the user %s already has an accepted invitation to %s",
			senderUserId, receiverUserId)
		GetLogger().Println(reason)
		err = errors.New(reason)
	} else if count == 1 {
		reason := fmt.Sprintf("user %s already has an accepted invitation to %s", senderUserId, receiverUserId)
		GetLogger().Println(reason)
		err = errors.New(reason)
	}
	return
}

func SendInvitation(senderUserId, invitationId string) (err error) {
	// add an invite to the sent user's invites array
	result, err := GetDBConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, senderUserId),
		),
		bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoAddToSetOperator,
				// just storing as name can be changed b/w sending request and seen by intended receiver
				// so will fetch other details when the receiver will see the invitation
				bson.EC.String(SentInvitesField, invitationId)),
		),
	)

	if err != nil {
		reason := fmt.Sprintf("error while adding invitation %s into %s's active sent invitation: %s", invitationId,
			senderUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while adding invitation %s into %s's active sent invitation: %s",
			invitationId, senderUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	}
	return
}

func CancelInvitation(senderUserId, invitationId string) (err error) {
	// change the state of invitation from active to cancelled
	result, err := GetDBConn().Collection(InviteCollection).UpdateOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, invitationId),
		),
		bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoSetOperator,
				bson.EC.String(InviteStateField, CancelledInvite)),
		),
	)
	if err != nil {
		reason := fmt.Sprintf("error while changing invitation %s state to %s: %s", invitationId,
			CancelledInvite, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while changing invitation %s  state to %s's : %s",
			invitationId, CancelledInvite, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}

	// remove invite from the sent user's invites array
	result, err = GetDBConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, senderUserId),
		),
		bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoPullOperator,
				// just storing as name can be changed b/w sending request and seen by intended receiver
				// so will fetch other details when the receiver will see the invitation
				bson.EC.String(SentInvitesField, invitationId)),
		),
	)

	if err != nil {
		reason := fmt.Sprintf("error while adding invitation %s into %s's active sent invitation: %s", invitationId,
			senderUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while adding invitation %s into %s's active sent invitation: %s",
			invitationId, senderUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}

	// add invite to the user's cancelled invites array
	result, err = GetDBConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, senderUserId),
		),
		bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoAddToSetOperator,
				// just storing as name can be changed b/w sending request and seen by intended receiver
				// so will fetch other details when the receiver will see the invitation
				bson.EC.String(CancelledInvitesField, invitationId)),
		),
	)
	if err != nil {
		reason := fmt.Sprintf("error while adding invitation %s into %s's cancelled invitation: %s", invitationId,
			senderUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while adding invitation %s into %s's cancelled invitation: %s",
			invitationId, senderUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	}
	return
}

func ReceiveInvitation(receiverUserId, invitationId string) (err error) {
	// add an invite to the receiver user's received invites array
	result, err := GetDBConn().Collection(InviteCollection).UpdateOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, receiverUserId),
		),
		bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoAddToSetOperator,
				// just storing as name can be changed b/w sending request and seen by intended receiver
				// so will fetch other details when the receiver will see the invitation
				bson.EC.String(ReceivedInvitesField, invitationId)),
		),
	)

	if err != nil {
		reason := fmt.Sprintf("error while adding invitation %s into %s's active received invitation: %s",
			invitationId, receiverUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while adding invitation %s into %s's received sent "+
			"invitation: %s", invitationId, receiverUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
	}
	return
}

func AcceptInvitation(receiverUserId, invitationId string) (err error) {
	// change the state of invitation from active to accepted
	result, err := GetDBConn().Collection(InviteCollection).UpdateOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, invitationId),
		),
		bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoSetOperator,
				bson.EC.String(InviteStateField, AcceptedInvite)),
		),
	)
	if err != nil {
		reason := fmt.Sprintf("error while changing invitation %s state to %s: %s", invitationId,
			CancelledInvite, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while changing invitation %s  state to %s's : %s",
			invitationId, CancelledInvite, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}

	// move the invite from receiver user's received invites to accepted invites
	result, err = GetDBConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, receiverUserId),
		),
		bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoAddToSetOperator,
				// just storing as name can be changed b/w sending request and seen by intended receiver
				// so will fetch other details when the receiver will see the invitation
				bson.EC.String(AcceptedInvitesField, invitationId)),
		),
	)
	if err != nil {
		reason := fmt.Sprintf("error while adding %s into %s's active accepted invitation: %s", receiverUserId,
			receiverUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while adding %s into %s's received accepted invitation: %s",
			receiverUserId, receiverUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}

	result, err = GetDBConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(UserEmailField, receiverUserId),
		),
		bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoPullOperator,
				// just storing as name can be changed b/w sending request and seen by intended receiver
				// so will fetch other details when the receiver will see the invitation
				bson.EC.String(ReceivedInvitesField, invitationId)),
		),
	)
	if err != nil {
		reason := fmt.Sprintf("error while removing %s from %s's received invitation: %s", receiverUserId,
			receiverUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while removing %s from %s's received invitation: %s",
			receiverUserId, receiverUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}

	// Changing the sender user's invitation data too

	invitationData, err := GetInvitation(invitationId)
	if err != nil {
		reason := fmt.Sprintf("error while fetching invitation %s data: %s", invitationId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}

	// move the invite from sender user's sent invites to accepted invites
	result, err = GetDBConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, invitationData.Sender),
		),
		bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoAddToSetOperator,
				// just storing as name can be changed b/w sending request and seen by intended receiver
				// so will fetch other details when the receiver will see the invitation
				bson.EC.String(AcceptedInvitesField, invitationId)),
		),
	)
	if err != nil {
		reason := fmt.Sprintf("error while adding %s into %s's accepted invitation: %s", receiverUserId,
			receiverUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while adding %s into %s's accepted invitation: %s",
			receiverUserId, receiverUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}

	result, err = GetDBConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, invitationData.Sender),
		),
		bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoPullOperator,
				// just storing as name can be changed b/w sending request and seen by intended receiver
				// so will fetch other details when the receiver will see the invitation
				bson.EC.String(SentInvitesField, invitationId)),
		),
	)
	if err != nil {
		reason := fmt.Sprintf("error while removing %s from %s's sent invitation: %s", receiverUserId,
			receiverUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while removing %s from %s's sent invitation: %s",
			receiverUserId, receiverUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}
	return
}

func RejectInvitation(receiverUserId, invitationId string) (err error) {
	// change the state of invitation from active to accepted
	result, err := GetDBConn().Collection(InviteCollection).UpdateOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, invitationId),
		),
		bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoSetOperator,
				bson.EC.String(InviteStateField, RejectedInvite)),
		),
	)
	if err != nil {
		reason := fmt.Sprintf("error while changing invitation %s state to %s: %s", invitationId,
			CancelledInvite, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while changing invitation %s  state to %s's : %s",
			invitationId, CancelledInvite, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}

	// move the invite from receiver user's received invites to rejected invites
	result, err = GetDBConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, receiverUserId),
		),
		bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoAddToSetOperator,
				// just storing as name can be changed b/w sending request and seen by intended receiver
				// so will fetch other details when the receiver will see the invitation
				bson.EC.String(RejectedInvitesField, invitationId)),
		),
	)
	if err != nil {
		reason := fmt.Sprintf("error while adding %s into %s's rejected invitation: %s", receiverUserId,
			receiverUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while adding %s into %s's rejected invitation: %s",
			receiverUserId, receiverUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}

	result, err = GetDBConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(UserEmailField, receiverUserId),
		),
		bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoPullOperator,
				// just storing as name can be changed b/w sending request and seen by intended receiver
				// so will fetch other details when the receiver will see the invitation
				bson.EC.String(ReceivedInvitesField, invitationId)),
		),
	)
	if err != nil {
		reason := fmt.Sprintf("error while removing %s from %s's received invitation: %s", receiverUserId,
			receiverUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while removing %s from %s's received invitation: %s",
			receiverUserId, receiverUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}

	// Changing the sender user's invitation data too
	invitationData, err := GetInvitation(invitationId)
	if err != nil {
		reason := fmt.Sprintf("error while fetching invitation %s data: %s", invitationId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}

	// move the invite from sender user's sent invites to accepted invites
	result, err = GetDBConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, invitationData.Sender),
		),
		bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoAddToSetOperator,
				// just storing as name can be changed b/w sending request and seen by intended receiver
				// so will fetch other details when the receiver will see the invitation
				bson.EC.String(RejectedInvitesField, invitationId)),
		),
	)
	if err != nil {
		reason := fmt.Sprintf("error while adding %s into %s's accepted invitation: %s", receiverUserId,
			receiverUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while adding %s into %s's accepted invitation: %s",
			receiverUserId, receiverUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}

	result, err = GetDBConn().Collection(UserInvitesCollection).UpdateOne(
		context.Background(),
		bson.NewDocument(
			bson.EC.String(ObjectID, invitationData.Sender),
		),
		bson.NewDocument(
			bson.EC.SubDocumentFromElements(MongoPullOperator,
				// just storing as name can be changed b/w sending request and seen by intended receiver
				// so will fetch other details when the receiver will see the invitation
				bson.EC.String(SentInvitesField, invitationId)),
		),
	)
	if err != nil {
		reason := fmt.Sprintf("error while removing %s from %s's sent invitation: %s", receiverUserId,
			receiverUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	} else if result.ModifiedCount != 1 {
		reason := fmt.Sprintf("invalid update count while removing %s from %s's sent invitation: %s",
			receiverUserId, receiverUserId, err)
		GetLogger().Println(reason)
		err = errors.New(reason)
		return
	}
	return
}
