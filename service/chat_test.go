package service

import (
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"strings"
	"testing"
	"time"
)

func TestGetChatByUserIDs(t *testing.T) {
	userId1, userId2 := primitive.NewObjectID(), primitive.NewObjectID()
	chat, err := GetChatByUserIDs(userId1, userId2)
	assert.Equal(t, err, mongo.ErrNoDocuments, "invalid IDs provided so chat should be unavailable")
	assert.Equal(t, primitive.ObjectID{}.String(), chat.ID.String(), "invalid IDs provided so chat should be unavailable")
}

func TestSendMessage(t *testing.T) {
	sender, receiver := primitive.NewObjectID(), primitive.NewObjectID()
	err := SendMessage(sender, receiver, "test message")
	assert.NoError(t, err, "new document should be created for the chat")
}

func TestPrintMessage(t *testing.T) {
	self, other := new(User), new(User)
	selfID, otherID := primitive.NewObjectID(), primitive.NewObjectID()
	self.ID, other.ID = selfID, otherID
	other.FirstName = "John"

	msg := new(Message)
	msg.Sender = selfID
	msg.Timestamp = time.Now().UTC()
	msg.Text = "self message"
	msgText := PrintMessage(*msg, self, other)
	assert.True(t, strings.Contains(msgText, "You"), "as you are the sender")
	assert.True(t, strings.Contains(msgText, "self message"), "text body of the message")
	assert.True(t, strings.Contains(msgText, msg.Timestamp.String()), "timestamp of the message")

	msg2 := new(Message)
	msg2.Sender = otherID
	msg2.Timestamp = time.Now().UTC()
	msg2.Text = "self message"
	msgText = PrintMessage(*msg2, self, other)
	assert.True(t, strings.Contains(msgText, other.FirstName), "as other person is the sender")
	assert.True(t, strings.Contains(msgText, "self message"), "text body of the message")
	assert.True(t, strings.Contains(msgText, msg2.Timestamp.String()), "timestamp of the message")
}

func TestFetchIncomingMessages(t *testing.T) {
	self, other := primitive.NewObjectID(), primitive.NewObjectID()
	msgs, err := fetchIncomingMessages(time.Now().UTC(), self, other)
	assert.Equal(t, mongo.ErrNoDocuments, err, "error as invalid users")
	assert.True(t, len(msgs) == 0, "empty list of messages expected")
}
