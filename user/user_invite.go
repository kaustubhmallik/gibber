package user

import (
	"context"
	"encoding/json"
	"gibber/datastore"
	"gibber/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// user invites collection and fields
const (
	userInvitesCollection = "user_invites"
	userIdField           = "user_id"
)

// userInvites stores reference for all the invitations sent and received associated with a user
type userInvites struct {
	ID        primitive.ObjectID   `bson:"_id" json:"-"`
	UserID    primitive.ObjectID   `bson:"user_id" json:"user_id"`     // object ID of user
	Sent      []primitive.ObjectID `bson:"sent" json:"sent"`           // active sent request by user
	Received  []primitive.ObjectID `bson:"received" json:"received"`   // active received request by user
	Accepted  []primitive.ObjectID `bson:"accepted" json:"accepted"`   // accepted requests by user
	Rejected  []primitive.ObjectID `bson:"rejected" json:"rejected"`   // rejected requests by user
	Cancelled []primitive.ObjectID `bson:"cancelled" json:"cancelled"` // cancelled sent requests by user
}

// createUserInvitesData creates the empty invites collection for the given user
func createUserInvitesData(ctx context.Context, userId interface{}, dbConn datastore.DatabaseInserter) (userInvitesDataId interface{}, err error) {
	userInvites := &userInvites{
		UserID:    userId.(primitive.ObjectID),
		Sent:      make([]primitive.ObjectID, 0),
		Received:  make([]primitive.ObjectID, 0),
		Accepted:  make([]primitive.ObjectID, 0),
		Rejected:  make([]primitive.ObjectID, 0),
		Cancelled: make([]primitive.ObjectID, 0),
	}
	userInvitesDataMap, _ := getMap(*userInvites)
	userInvitesDataMap["user_id"] = userId.(primitive.ObjectID)
	res, err := dbConn.InsertOne(ctx, userInvitesDataMap)
	if err != nil {
		log.Logger().Printf("error creating user %s invites data: %s", userId, err)
	} else {
		userInvitesDataId = res.InsertedID
	}
	return
}

// String representation for a user
func (u *userInvites) String() string {
	return u.ID.String()
}

// getMap gives the map converted form for a given struct, to be used to store as document (JSON) in mongo
func getMap(data interface{}) (dataMap map[string]interface{}, err error) {
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
