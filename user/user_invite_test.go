package user

import (
	"context"
	"errors"
	"gibber/datastore"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"testing"
)

var InsertFailed = errors.New("insert failure")

type databaseInsertFail struct {
	// implementing DatabaseInserter interface for failure inserts
}

func (d *databaseInsertFail) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	return nil, InsertFailed
}

func TestCreateUserInvitesData(t *testing.T) {
	tests := []struct {
		name     string
		userId   primitive.ObjectID
		expIdLen int
		dbConn   datastore.DatabaseInserter
		err      error
	}{
		{
			name:     "success scenario",
			userId:   primitive.NewObjectID(),
			expIdLen: len(primitive.NewObjectID().Hex()),
			dbConn:   datastore.MongoConn().Collection(userInvitesCollection),
			err:      nil,
		},
		{
			name:     "insert failure by custom interface implementation",
			userId:   primitive.NewObjectID(),
			expIdLen: 0,
			dbConn:   new(databaseInsertFail),
			err:      InsertFailed,
		},
	}
	for _, tc := range tests {
		userInvId, err := createUserInvitesData(tc.userId, context.Background(), tc.dbConn)
		assert.Equal(t, tc.err, err, "%s failed as user invite creation failed", tc.name)
		if err == nil {
			_, err = primitive.ObjectIDFromHex(userInvId.(primitive.ObjectID).Hex())
			assert.Equal(t, tc.err, err, "%s failed as invalid object ID returned", tc.name)
			assert.Equal(t, tc.expIdLen, len(userInvId.(primitive.ObjectID).Hex()), "%s failed as invalid length object ID returned")
		}
	}
}

func TestUserInvites_String(t *testing.T) {
	ui := new(userInvites)
	assert.Equal(t, primitive.ObjectID{}.String(), ui.String(), "uninitialized object ID should be empty string")
	ui.ID = primitive.NewObjectID()
	assert.True(t, ui.String() != primitive.ObjectID{}.String(), "initialized object ID should be non-empty string")
}

func TestGetMap(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		errOccur bool
	}{
		{
			name: "success flow",
			input: struct {
				foo     string
				num     int
				float   float64
				success bool
			}{
				foo:     "bar",
				num:     1,
				float:   1.3,
				success: true,
			},
			errOccur: false,
		},
		{
			name:     "empty struct",
			input:    nil,
			errOccur: false,
		},
		{
			name:     "primitive data type",
			input:    1,
			errOccur: true,
		},
		{
			name:     "primitive data type",
			input:    true,
			errOccur: true,
		},
		{
			name:     "empty function",
			input:    func() {},
			errOccur: true,
		},
	}
	for _, tc := range tests {
		_, err := getMap(tc.input)
		assert.Equal(t, tc.errOccur, err != nil, "%s failed", tc.name)
	}
}
