package service

import (
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
)

func TestMongoConn(t *testing.T) {
	conn := MongoConn()
	assert.NotNil(t, conn, "mongo connection is nil")
}

func TestSortObjectIDs(t *testing.T) {
	for i := 0; i < 5; i++ { // repeating it 5 times
		id1 := primitive.NewObjectID()
		id2 := primitive.NewObjectID()
		sortID1, sortID2 := SortObjectIDs(id1, id2)
		assert.True(t, sortID1.Hex() < sortID2.Hex(), "incorrect order after sorting")
	}
}
