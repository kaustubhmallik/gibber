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
	ChatCollection   = "chats"
	ChatUser1        = "user_1"
	ChatUser2        = "user_2"
	ChatMessages     = "messages"
	MessageSender    = "sender"
	MessageText      = "text"
	MessageTimestamp = "timestamp"
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
			err = fmt.Errorf("no chat found with user IDs %s and %s", userID1.String(), userID2.String())
		} else {
			err = fmt.Errorf("decoding(unmarshal) chat result for users %s and %s failed: %s",
				userID1.String(), userID2.String(), err)
		}
		Logger().Println(err)
	}
	return
}

func GetChatByObjID(ID primitive.ObjectID) (chat *Chat, err error) {
	chat = &Chat{}
	err = MongoConn().
		Collection(ChatCollection).
		FindOne(context.Background(),
			bson.M{ObjectID: ID}).
		Decode(chat)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			err = fmt.Errorf("no chat found with ID: %s", ID.String())
		} else {
			err = fmt.Errorf("decoding(unmarshal) chat result with ID %s failed: %s", ID.String(), err)
		}
		Logger().Println(err)
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
		err = fmt.Errorf("error while sending msg from %s to %s: %s", sender.String(), receiver.String(), err)
		Logger().Print(err)
		return
	} else if res.ModifiedCount+res.UpsertedCount != 1 {
		err = fmt.Errorf("document not created/updated while sending msg from %s to %s", sender.String(),
			receiver.String())
		return
	}
	return
}

func PrintMessage(msg Message, self, other *User) string {
	var sender string
	if msg.Sender == self.ID {
		sender = "You"
	} else {
		sender = other.FirstName
	}
	return fmt.Sprintf("%s (%s): %s", sender, msg.Timestamp, msg.Text)
}

// sort on timestamp
func fetchIncomingMessages(timestamp time.Time, self, other primitive.ObjectID) (msgs []Message, err error) {
	chat, err := GetChatByUserIDs(self, other)
	if err != nil {
		Logger().Print(err)
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
