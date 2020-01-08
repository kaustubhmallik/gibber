package datastore

import (
	"context"
	"errors"
	"gibber/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"sync"
)

// mongodb query operators
const (
	MongoSetOperator  = "$set"
	MongoPushOperator = "$push"
	MongoPullOperator = "$pull"
)

// various collections to be used by the service, which need to be initialized
const (
	UserCollection        = "users"
	UserInvitesCollection = "user_invites"
	FriendsCollection     = "friends"
	ChatCollection        = "chats"
)

// common fields/attributes of documents in various collections
const (
	ObjectID = "_id" // document level Primary Key
)

// ENV params to get details about mongo instance to connect, along with other connection args
var (
	mongoConnScheme = os.Getenv("GIBBER_MONGO_CONN_SCHEME")
	mongoHost       = os.Getenv("GIBBER_MONGO_HOST")
	mongoUser       = os.Getenv("GIBBER_MONGO_USER")
	mongoPwd        = os.Getenv("GIBBER_MONGO_PWD")
	mongoDatabase   = os.Getenv("GIBBER_MONGO_DB")
	mongoOptions    = os.Getenv("GIBBER_MONGO_OPTS")
)

// generic mongo errors
var (
	NoDocUpdate = errors.New("no document updated")
)

// instead of a generic client, return the target DB handler, to avoid selecting it again and again in each query
var mongoConn *mongo.Database
var initMongoConn sync.Once

func init() {
	initCollections()
}

// initMongoConnPool initializes a new client, and set the target database handler
func initMongoConnPool() {
	addressURL := mongoConnScheme + "://"
	if mongoUser != "" {
		addressURL += mongoUser + ":" + mongoPwd + "@"
	}
	addressURL += mongoHost
	addressURL += "/" + mongoDatabase
	if mongoOptions != "" {
		addressURL += "?" + mongoOptions
	}
	opts := options.Client().ApplyURI(addressURL)
	client, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		log.Logger().Fatalf("create mongo connection on %s pool failed: %s", addressURL, err)
	} else {
		log.Logger().Printf("mongo successfully connected on %s", addressURL)
	}
	mongoConn = client.Database("gibber")
}

// MongoConn returns database handler instance of mongo for target database
func MongoConn() *mongo.Database {
	initMongoConn.Do(initMongoConnPool)
	return mongoConn
}

// initCollections create the document collections (if non-existent) to be utilized by the service.
func initCollections() {
	mongoConn := MongoConn()
	collections := []string{
		UserCollection,
		UserInvitesCollection,
		FriendsCollection,
		ChatCollection,
	}
	for _, coll := range collections {
		count, err := mongoConn.Collection(coll).CountDocuments(context.Background(), bson.D{})
		if err != nil {
			log.Logger().Printf("%s count fetch failed: %s", coll, err)
		}
		if count == 0 {
			_, err = mongoConn.Collection(coll).InsertOne(context.Background(), bson.D{})
			if err != nil {
				log.Logger().Printf("%s collection creation failed: %s", coll, err)
			}
		}
	}
}
