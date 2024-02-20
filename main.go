package main

import (
	"context"
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func search(collection *mongo.Collection, target string, maxEdit int) ([]Hex, error) {
	filter := bson.D{{
		"$text",
		bson.D{{
			"$search", target,
		}},
	}}

	fmt.Println("created filter")

	var results []Hex
	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		return results, err
	}

	fmt.Println("cursor ran")
	if err = cursor.All(context.TODO(), &results); err != nil {
		return results, err
	}

	fmt.Println("result read")
	for _, result := range results {
		res, _ := json.Marshal(result)
		fmt.Println(string(res))
	}

	return results, err
}

func main() {
	client, err := Connect()
	if err != nil {
		panic(err)
	}
	collection := client.Database("fnn").Collection("hash")

	target := "d92ce59244421398048b173e17dda78f0abc8f1c758411f0f9b7952693eccc14b68aa046bed88e7ea9d791718572c46947849f5307153aed27cd8471fbc7f01e"
	results, err := search(collection, target, 5)
	if err != nil {
		panic(err)
	}
	fmt.Println(results)
}
