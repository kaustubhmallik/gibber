package main

import (
	"gibber/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Foo struct {
	ID        primitive.ObjectID `bson:"_id" json:"-"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`     // object ID of user
	Sent      listObjID          `bson:"sent" json:"sent"`           // active sent request by user
	Received  listObjID          `bson:"received" json:"received"`   // active received request by user
	Accepted  listObjID          `bson:"accepted" json:"accepted"`   // accepted requests by user
	Rejected  listObjID          `bson:"rejected" json:"rejected"`   // rejected requests by user
	Cancelled listObjID          `bson:"cancelled" json:"cancelled"` // cancelled sent requests by user
}

type listObjID []primitive.ObjectID

func main() {
	service.StartServer()
	//conn := service.MongoConn()
	//objID, _ := primitive.ObjectIDFromHex("5deeb054520ac65330c7ba2d")
	//userInv := service.UserInvitesData{}
	//err := conn.Collection(service.UserInvitesCollection).
	//	FindOne(context.Background(), bson.M{"_id": objID}).
	//	Decode(&userInv)
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Printf("%+v\n", userInv)
}
