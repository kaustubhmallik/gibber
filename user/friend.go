package user

import "go.mongodb.org/mongo-driver/bson/primitive"

// user document collection name and fields
const (
	friendsCollection = "friends"
	friendsField      = "friend_ids"
)

// friends lists all the connected user for a given user
type friends struct {
	ID        primitive.ObjectID   `bson:"_id" json:"-"`
	UserID    primitive.ObjectID   `bson:"user_id" json:"user_id"`
	FriendIDs []primitive.ObjectID `bson:"friend_ids" json:"friend_ids"`
}
