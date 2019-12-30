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

const (
	UserCollection        = "users"
	UserInvitesCollection = "user_invites"
	FriendsCollection     = "friends"
	ChatCollection        = "chats"
)

// common fields/attributes of documents in various collections
const (
	ObjectID = "_id" // document level Primary Key
	// Creation time of the document can be fetched via ObjectId.getTimestamp()
	// see https://docs.mongodb.com/manual/reference/method/ObjectId.getTimestamp/#ObjectId.getTimestamp
)

var (
	MongoConnScheme = os.Getenv("GIBBER_MONGO_CONN_SCHEME")
	MongoHost       = os.Getenv("GIBBER_MONGO_HOST")
	MongoUser       = os.Getenv("GIBBER_MONGO_USER")
	MongoPwd        = os.Getenv("GIBBER_MONGO_PWD")
	MongoDatabase   = os.Getenv("GIBBER_MONGO_DB")
	MongoOptions    = os.Getenv("GIBBER_MONGO_OPTS") // retryWrites=true&w=majority
)

var (
	NoDocUpdate = errors.New("no document updated")
)

// instead of a generic client, return the target DB handler, to avoid selecting it again and again in each query
var mongoConn *mongo.Database
var initMongoConn sync.Once

func init() {
	initCollections()
}

// mongodb+srv://<username>:<password>@gibber-qiquc.gcp.mongodb.net/test?retryWrites=true&w=majority
// initializes a new client, and set the target database handler
func initMongoConnPool() {
	addressURL := MongoConnScheme + "://"
	if MongoUser != "" {
		addressURL += MongoUser + ":" + MongoPwd + "@"
	}
	addressURL += MongoHost
	addressURL += "/" + MongoDatabase
	if MongoOptions != "" {
		addressURL += "?" + MongoOptions
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

// returns database handler instance of mongo for target database
func MongoConn() *mongo.Database {
	initMongoConn.Do(initMongoConnPool)
	return mongoConn
}

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
