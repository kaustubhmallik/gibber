package service

import (
	"context"
	"fmt"
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
	userID1, userID2 = SortObjectIDs(userID1, userID2)
	err = MongoConn().
		Collection(ChatCollection).
		FindOne(context.Background(),
			bson.D{
				{Key: ChatUser1, Value: userID1},
				{Key: ChatUser2, Value: userID2},
			}).
		Decode(chat)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			Logger().Printf("no chat found with user IDs %s and %s", userID1, userID2)
		} else {
			Logger().Printf("decoding(unmarshal) chat result for users %s and %s failed: %s", userID1, userID2, err)
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
	sender, receiver = SortObjectIDs(sender, receiver)
	res, err := MongoConn().Collection(ChatCollection).UpdateOne(context.Background(),
		bson.D{
			{Key: ChatUser1, Value: sender},
			{Key: ChatUser2, Value: receiver},
		},
		bson.D{
			{Key: MongoPushOperator, Value: bson.D{{Key: ChatMessages, Value: msg}}},
		},
		options.Update().SetUpsert(true))
	if err != nil {
		Logger().Printf("error while sending msg from %s to %s: %s", sender, receiver, err)
	} else if res.ModifiedCount+res.UpsertedCount != 1 {
		Logger().Printf("document not created/updated while sending msg from %s to %s", sender, receiver)
		err = NoDocUpdate
	}
	return
}

//func PrintMessage(msg Message, self, other *User) string {
func PrintMessage(msg Message, sender string) string {
	return fmt.Sprintf("%s (%s): %s", sender, msg.Timestamp, msg.Text)
}

// sort on timestamp
func fetchIncomingMessages(timestamp time.Time, self, other primitive.ObjectID) (msgs []Message, err error) {
	chat, err := GetChatByUserIDs(self, other)
	if err != nil {
		Logger().Printf("error fetching chat for user %s: %s", self, err)
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
