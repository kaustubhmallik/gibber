package user

import (
	"context"
	"fmt"
	"gibber/datastore"
	"gibber/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

// user document collection name and fields
const (
	ChatCollection = "chats"
	ChatUser1      = "user_1"
	ChatUser2      = "user_2"
	ChatMessages   = "messages"
)

type Message struct {
	Sender    primitive.ObjectID `json:"sender" bson:"sender"`
	Text      string             `json:"text" bson:"text"`
	Timestamp time.Time          `json:"timestamp,omitempty" bson:"timestamp"`
}

type Chat struct {
	ID       primitive.ObjectID `json:"-" bson:"_id"`
	User1    primitive.ObjectID `json:"user_1" bson:"user_1"`
	User2    primitive.ObjectID `json:"user_2" bson:"user_2"`
	Messages []Message          `json:"messages" bson:"messages"`
}

func GetChatByUserIDs(userID1, userID2 primitive.ObjectID) (chat *Chat, err error) {
	chat = &Chat{}
	userID1, userID2 = datastore.SortObjectIDs(userID1, userID2)
	err = datastore.MongoConn().
		Collection(ChatCollection).
		FindOne(context.Background(),
			bson.D{
				{Key: ChatUser1, Value: userID1},
				{Key: ChatUser2, Value: userID2},
			}).
		Decode(chat)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Logger().Printf("no chat found with user IDs %s and %s", userID1, userID2)
		} else {
			log.Logger().Printf("decoding(unmarshal) chat result for users %s and %s failed: %s", userID1, userID2, err)
		}
	}
	return
}

func SendMessage(sender, receiver primitive.ObjectID, text string) (err error) {
	msg := Message{
		Sender:    sender,
		Text:      text,
		Timestamp: time.Now().UTC(),
	}
	sender, receiver = datastore.SortObjectIDs(sender, receiver)
	res, err := datastore.MongoConn().Collection(ChatCollection).UpdateOne(context.Background(),
		bson.D{
			{Key: ChatUser1, Value: sender},
			{Key: ChatUser2, Value: receiver},
		},
		bson.D{
			{Key: datastore.MongoPushOperator, Value: bson.D{{Key: ChatMessages, Value: msg}}},
		},
		options.Update().SetUpsert(true))
	if err != nil {
		log.Logger().Printf("error while sending msg from %s to %s: %s", sender, receiver, err)
	} else if res.ModifiedCount+res.UpsertedCount != 1 {
		log.Logger().Printf("document not created/updated while sending msg from %s to %s", sender, receiver)
		err = datastore.NoDocUpdate
	}
	return
}

//func PrintMessage(msg Message, self, other *User) string {
func PrintMessage(msg Message, sender string) string {
	return fmt.Sprintf("%s (%s): %s", sender, msg.Timestamp, msg.Text)
}

// sort on timestamp
func FetchIncomingMessages(timestamp time.Time, self, other primitive.ObjectID) (msgs []Message, err error) {
	chat, err := GetChatByUserIDs(self, other)
	if err != nil {
		log.Logger().Printf("error fetching chat for user %s: %s", self, err)
		return
	}
	msgs = make([]Message, 0)
	for _, msg := range chat.Messages {
		if msg.Timestamp.After(timestamp) && msg.Sender.String() == other.String() {
			msgs = append(msgs, msg)
			timestamp = msg.Timestamp
		}
	}
	return
}
