package user

import (
	"context"
	"fmt"
	"gibber/datastore"
	"gibber/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

// user document collection name and fields
const (
	chatCollection = "chats"
	chatUser1      = "user_1"
	chatUser2      = "user_2"
	chatMessages   = "messages"
)

// message depicts the way in which a chat message is stored in the database
type message struct {
	Sender    primitive.ObjectID `json:"sender" bson:"sender"`
	Text      string             `json:"text" bson:"text"`
	Timestamp time.Time          `json:"timestamp,omitempty" bson:"timestamp"`
}

// chat stores the conversation b/w two users
type chat struct {
	ID       primitive.ObjectID `json:"-" bson:"_id"`
	User1    primitive.ObjectID `json:"user_1" bson:"user_1"`
	User2    primitive.ObjectID `json:"user_2" bson:"user_2"`
	Messages []message          `json:"messages" bson:"messages"`
}

// FetchIncomingMessages fetches the incoming messages for the given user from the other user
// that came after given timestamp
func FetchIncomingMessages(timestamp time.Time, self, other primitive.ObjectID) (msgs []message, err error) {
	chat, err := getChatByUserIDs(self, other, datastore.MongoConn().Collection(chatCollection))
	if err != nil {
		log.Logger().Printf("error fetching chat for user %s: %s", self, err)
		return
	}
	msgs = make([]message, 0)
	for _, msg := range chat.Messages {
		if msg.Timestamp.After(timestamp) && msg.Sender.String() == other.String() {
			msgs = append(msgs, msg)
			timestamp = msg.Timestamp
		}
	}
	return
}

// SendMessage sends a given message from sender to receiver
func SendMessage(sender, receiver primitive.ObjectID, text string, updater datastore.DatabaseUpdater) (err error) {
	msg := message{
		Sender:    sender,
		Text:      text,
		Timestamp: time.Now().UTC(),
	}
	if sender.Hex() > receiver.Hex() { // ordering IDs
		sender, receiver = receiver, sender
	}
	res, err := updater.UpdateOne(context.Background(),
		bson.D{
			{Key: chatUser1, Value: sender},
			{Key: chatUser2, Value: receiver},
		},
		bson.D{
			{Key: datastore.MongoPushOperator, Value: bson.D{{Key: chatMessages, Value: msg}}},
		},
		options.Update().SetUpsert(true))
	if err != nil {
		log.Logger().Printf("error while sending msg from %s to %s: %s", sender, receiver, err)
	} else if res.ModifiedCount+res.UpsertedCount != 1 {
		log.Logger().Printf("document not created/updated while sending msg from %s to %s", sender, receiver)
		err = datastore.ErrNoDocUpdate
	}
	return
}

// getChatByUserIDs fetches the chat b/w two users
// it sorts the user IDs as to avoid storing both combination of userIds in the database
func getChatByUserIDs(userID1, userID2 primitive.ObjectID, finder datastore.DatabaseFinder) (ch *chat, err error) {
	ch = &chat{}
	if userID1.Hex() > userID2.Hex() { // ordering IDs
		userID1, userID2 = userID2, userID1
	}
	err = finder.FindOne(context.Background(),
		bson.D{
			{Key: chatUser1, Value: userID1},
			{Key: chatUser2, Value: userID2},
		}).
		Decode(ch)
	if err != nil {
		log.Logger().Printf("no ch found with user IDs %s and %s", userID1, userID2) // other errors are not expected
	}
	return
}

// printMessage gives the string representation for a given message
// TODO: convert it into a Stringify interface and use it
func printMessage(msg message, sender string) string {
	return fmt.Sprintf("%s (%s): %s", sender, msg.Timestamp, msg.Text)
}
