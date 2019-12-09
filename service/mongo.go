package service

import (
	"context"
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
var mongoConn *mongo.Database
var initMongoConn sync.Once

// initializes a new client, and set the target database handler
func initMongoConnPool() {
	//address := fmt.Sprintf("%s://%s:%s@%s:%s/%s", ConnScheme, MongoUser, MongoPwd, MongoHost,
	//	MongoPort, MongoDatabase)
	address := "mongodb://localhost:27017/gibber"
	opts := options.Client().ApplyURI(address)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		GetLogger().Fatalf("create mongo connection on %s pool failed: %s", address, err)
	} else {
		GetLogger().Printf("mongo successfully connected on %s", address)
	}
	//mongoConn = client.Database(MongoDatabase)
	mongoConn = client.Database("gibber")
}

// returns database handler instance of mongo for target database
func MongoConn() *mongo.Database {
	initMongoConn.Do(initMongoConnPool)
	return mongoConn
}
