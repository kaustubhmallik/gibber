package service

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"sync"
	"time"
)

const ConnScheme = "mongodb"

// mongodb query operators
const (
	MongoSetOperator      = "$set"
	MongoPushOperator     = "$push"
	MongoAddToSetOperator = "$addToSet"
	MongoPullOperator     = "$pull"
)

// common fields/attributes of documents in various collections
const (
	ObjectID = "_id" // document level Primary Key
	// Creation time of the document can be fetched via ObjectId.getTimestamp()
	// see https://docs.mongodb.com/manual/reference/method/ObjectId.getTimestamp/#ObjectId.getTimestamp
)

var MongoHost = os.Getenv("GIBBER_MONGO_HOST")
var MongoPort = os.Getenv("GIBBER_MONGO_PORT")
var MongoUser = os.Getenv("GIBBER_MONGO_USER")
var MongoPwd = os.Getenv("GIBBER_MONGO_PWD")
var MongoDatabase = os.Getenv("GIBBER_MONGO_DB")

// instead of a generic client, return the target DB handler, to avoid selecting it again and again in each query
var dbConn *mongo.Database
var connOnce sync.Once

// initializes a new client, and set the target database handler
func createConnectionPool() {
	address := fmt.Sprintf("%s://%s:%s@%s:%s/%s", ConnScheme, MongoUser, MongoPwd, MongoHost,
		MongoPort, MongoDatabase)
	opts := options.Client().ApplyURI("mongodb://localhost:27017")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		GetLogger().Fatalf("create mongo connection on %s pool failed: %s", address, err)
	} else {
		GetLogger().Printf("mongo successfully connected on %s", address)
	}
	err = client.Connect(context.TODO())
	if err != nil {
		GetLogger().Fatalf("creating mongo context failed: %s", err)
	}
	dbConn = client.Database(MongoDatabase)
}

// returns database handler instance of mongo for target database
func GetDBConn() *mongo.Database {
	connOnce.Do(createConnectionPool)
	return dbConn
}
