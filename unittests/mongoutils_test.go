package unittests

import (
	"fmt"
	"testing"

	"github.com/kevinyjn/gocom/mongodb"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type mongoTestModel struct {
	ID   primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Code string             `bson:"code" json:"code"`
}

func TestMongoGenerateUpdateFields(t *testing.T) {
	m := mongoTestModel{
		ID:   primitive.NewObjectID(),
		Code: "123",
	}
	val := mongodb.MongoOrganizeUpdatesFields(&m)
	fmt.Printf("Testing update fields finished %+v\n", val)
}
