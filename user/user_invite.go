package user

import (
	"context"
	"encoding/json"
	"gibber/datastore"
	"gibber/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	UserInvitesCollection = "user_invites"
)

// user invites data fields
const (
	SentInvitesField     = "sent"
	ReceivedInvitesField = "received"
	UserIdField          = "user_id"
)

// stores reference for all the invitations sent and received associated with a user
type UserInvites struct {
	ID        primitive.ObjectID   `bson:"_id" json:"-"`
	UserID    primitive.ObjectID   `bson:"user_id" json:"user_id"`     // object ID of user
	Sent      []primitive.ObjectID `bson:"sent" json:"sent"`           // active sent request by user
	Received  []primitive.ObjectID `bson:"received" json:"received"`   // active received request by user
	Accepted  []primitive.ObjectID `bson:"accepted" json:"accepted"`   // accepted requests by user
	Rejected  []primitive.ObjectID `bson:"rejected" json:"rejected"`   // rejected requests by user
	Cancelled []primitive.ObjectID `bson:"cancelled" json:"cancelled"` // cancelled sent requests by user
}

func CreateUserInvitesData(userId interface{}, ctx context.Context, dbConn datastore.DatabaseInserter) (userInvitesDataId interface{}, err error) {
	userInvites := &UserInvites{
		UserID:    userId.(primitive.ObjectID),
		Sent:      make([]primitive.ObjectID, 0),
		Received:  make([]primitive.ObjectID, 0),
		Accepted:  make([]primitive.ObjectID, 0),
		Rejected:  make([]primitive.ObjectID, 0),
		Cancelled: make([]primitive.ObjectID, 0),
	}
	userInvitesDataMap, _ := GetMap(*userInvites)
	userInvitesDataMap["user_id"] = userId.(primitive.ObjectID)
	res, err := dbConn.InsertOne(ctx, userInvitesDataMap)
	if err != nil {
		log.Logger().Printf("error creating user %s invites data: %s", userId, err)
	} else {
		userInvitesDataId = res.InsertedID
	}
	return
}

func (u *UserInvites) String() string {
	return u.ID.String()
}

func GetMap(data interface{}) (dataMap map[string]interface{}, err error) {
	if data == nil {
		return
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return
	}
	err = json.Unmarshal(bytes, &dataMap)
	return
}
