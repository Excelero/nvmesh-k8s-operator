package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	managementDBName = "management"
)

func FindOne(client *mongo.Client, collectionName string, filter interface{}, projection interface{}, result interface{}) error {
	opts := options.FindOne().SetProjection(projection)
	collection := client.Database(managementDBName).Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := collection.FindOne(ctx, bson.D{}, opts).Decode(result)
	return err
}

func UpdateOne(client *mongo.Client, collectionName string, filter interface{}, update interface{}) error {
	collection := client.Database(managementDBName).Collection(collectionName)
	result, err := collection.UpdateOne(context.TODO(), filter, update)

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New(fmt.Sprintf("Failed to update document in collection %s and filter %+v", collectionName, filter))
	}

	fmt.Printf("Documents matched: %v\n", result.MatchedCount)
	fmt.Printf("Documents updated: %v\n", result.ModifiedCount)

	return err
}
