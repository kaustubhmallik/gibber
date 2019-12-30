package user

import "go.mongodb.org/mongo-driver/bson/primitive"

// user document collection name and fields
const (
	FriendsCollection = "friends"
	FriendsField      = "friend_ids"
)

type Friends struct {
	ID        primitive.ObjectID   `bson:"_id" json:"-"`
	UserID    primitive.ObjectID   `bson:"user_id" json:"user_id"`
	FriendIDs []primitive.ObjectID `bson:"friend_ids" json:"friend_ids"`
}

func NewFriends() *Friends {
	return new(Friends)
}
