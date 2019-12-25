package service

import (
	"context"
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"testing"
)

func TestCreateUserInvitesData(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expIdLen int
		err      error
	}{
		{
			name:     "success scenario",
			input:    primitive.NewObjectID().Hex(),
			expIdLen: len(primitive.NewObjectID().Hex()),
			err:      nil,
		},
		{
			name:     "failed scenario: invalid user ID",
			input:    "invalid userId",
			expIdLen: 0,
			err:      *new(hex.InvalidByteError),
		},
	}
	for _, tc := range tests {
		userId, err := primitive.ObjectIDFromHex(tc.input)
		assert.Equal(t, reflect.TypeOf(tc.err), reflect.TypeOf(err), "%s failed", tc.name)
		if err == nil {
			userInvId, err := CreateUserInvitesData(userId, context.Background())
			assert.Equal(t, tc.err, err, "%s failed as user invite creation failed", tc.name)
			_, err = primitive.ObjectIDFromHex(userInvId.(primitive.ObjectID).Hex())
			assert.Equal(t, tc.err, err, "%s failed as invalid object ID returned", tc.name)
			assert.Equal(t, tc.expIdLen, len(userInvId.(primitive.ObjectID).Hex()),
				"%s failed as invalid length object ID returned")
		}
	}
}
