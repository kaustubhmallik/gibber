package user

import (
	"context"
	"errors"
	"gibber/datastore"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strings"
	"testing"
	"time"
)

var errUpdateFailed = errors.New("update failure")

type databaseUpdateFail struct {
	// implement DatabaseUpdate interface with failure update operation
}

func (d *databaseUpdateFail) UpdateOne(ctx context.Context, filter interface{}, update interface{},
	opts ...*options.UpdateOptions) (res *mongo.UpdateResult, err error) {
	err = errUpdateFailed
	return
}

type databaseUpdateNoEffect struct {
	// implement DatabaseUpdate interface with failure update operation
}

func (d *databaseUpdateNoEffect) UpdateOne(ctx context.Context, filter interface{}, update interface{},
	opts ...*options.UpdateOptions) (res *mongo.UpdateResult, err error) {
	res = new(mongo.UpdateResult) // default count values (fields) will be zero which is required
	return
}

func TestGetChatByUserIDs(t *testing.T) {
	userId1, userId2 := primitive.NewObjectID(), primitive.NewObjectID()
	chat, err := getChatByUserIDs(userId1, userId2, datastore.MongoConn().Collection(chatCollection))
	assert.Equal(t, err, mongo.ErrNoDocuments, "invalid IDs provided so chat should be unavailable")
	assert.Equal(t, primitive.ObjectID{}.String(), chat.ID.String(), "invalid IDs provided so chat should be unavailable")
}

func TestSendMessage(t *testing.T) {
	sender, receiver := primitive.NewObjectID(), primitive.NewObjectID()
	err := SendMessage(sender, receiver, "test message", datastore.MongoConn().Collection(chatCollection))
	assert.NoError(t, err, "new document should be created for the chat")

	err = SendMessage(sender, receiver, "test message", new(databaseUpdateFail))
	assert.Equal(t, errUpdateFailed, err, "operation should fail")

	err = SendMessage(sender, receiver, "test message", new(databaseUpdateNoEffect))
	assert.Equal(t, datastore.ErrNoDocUpdate, err, "operation should fail")
}

func TestPrintMessage(t *testing.T) {
	self, other := new(User), new(User)
	selfID, otherID := primitive.NewObjectID(), primitive.NewObjectID()
	self.ID, other.ID = selfID, otherID
	other.FirstName = "John"

	msg := new(message)
	msg.Sender = selfID
	msg.Timestamp = time.Now().UTC()
	msg.Text = "self message"
	msgText := printMessage(*msg, "You")
	assert.True(t, strings.Contains(msgText, "You"), "as you are the sender")
	assert.True(t, strings.Contains(msgText, "self message"), "text body of the message")
	assert.True(t, strings.Contains(msgText, msg.Timestamp.String()), "timestamp of the message")

	msg2 := new(message)
	msg2.Sender = otherID
	msg2.Timestamp = time.Now().UTC()
	msg2.Text = "self message"
	msgText = printMessage(*msg2, other.FirstName)
	assert.True(t, strings.Contains(msgText, other.FirstName), "as other person is the sender")
	assert.True(t, strings.Contains(msgText, "self message"), "text body of the message")
	assert.True(t, strings.Contains(msgText, msg2.Timestamp.String()), "timestamp of the message")
}

func TestFetchIncomingMessages(t *testing.T) {
	self, other := primitive.NewObjectID(), primitive.NewObjectID()
	msgs, err := FetchIncomingMessages(time.Now().UTC(), self, other)
	assert.Equal(t, mongo.ErrNoDocuments, err, "error as invalid users")
	assert.True(t, len(msgs) == 0, "empty list of messages expected")

	// fetching a genuine chat, so creating users and chat b/w them
	user1 := new(User)
	user1.FirstName = "John"
	user1.LastName = "Doe"
	user1.Email = "john" + randomString(15) + "@doe.com"
	user1.Password = "password"
	user1ID, err := CreateUser(user1)
	assert.NoError(t, err, "user creation failed")

	user2 := new(User)
	user2.FirstName = "John2"
	user2.LastName = "Doe2"
	user2.Email = "john" + randomString(15) + "@doe.com"
	user2.Password = "password"
	user2ID, err := CreateUser(user2)
	assert.NoError(t, err, "user creation failed")

	err = SendMessage(user1ID.(primitive.ObjectID), user2ID.(primitive.ObjectID), "test message",
		datastore.MongoConn().Collection(datastore.ChatCollection))
	assert.NoError(t, err, "new document should be created for the chat")

	msgs, err = FetchIncomingMessages(time.Now().UTC().Add(-time.Minute), user2ID.(primitive.ObjectID), user1ID.(primitive.ObjectID))
	assert.NoError(t, err, "error fetching just sent message")
	assert.Equal(t, 1, len(msgs), "a single message is expected")
}
