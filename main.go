package main

import (
	"context"
	"fmt"
	"math"
	"slices"

	aggregator "search/aggregate"
	connector "search/mongo"

	"go.mongodb.org/mongo-driver/bson"
)

func main() {
	client, err := connector.Connect()
	if err != nil {
		panic(err)
	}
	database := client.Database("playground")
	buckets := 8
	ctx := context.TODO()

	exists, err := client.ListDatabaseNames(ctx, bson.D{})
	if err != nil {
		panic(err)
	}
	if !slices.Contains(exists, "playground") {
		aggregator.Insert(database, buckets, ctx)
	}

	// r-neighbour search for query g
	query := "338E33D6E60487CEB30B0D0A276B5E889A0C07FB2A9ABEA038C6CD9C49F4C521"
	segments := aggregator.SplitSegments(query, buckets)
	searchRadius := 30
	substringRadius := int(math.Floor(float64(searchRadius) / float64(buckets)))
	fmt.Println("substring radius: ", substringRadius)

	var candidates []aggregator.Hex

	for i := 0; i < (searchRadius - substringRadius*buckets); i++ {
		bucket := database.Collection(fmt.Sprintf("mih%d", i+1))
		bucketNeighbour, err := aggregator.SearchBucket(bucket, segments[i], substringRadius, ctx)
		if err != nil {
			panic(err)
		}
		candidates = append(candidates, bucketNeighbour...)
	}

	for i := (searchRadius - substringRadius*buckets); i < buckets; i++ {
		bucket := database.Collection(fmt.Sprintf("mih%d", i+1))
		bucketNeighbour, err := aggregator.SearchBucket(bucket, segments[i], substringRadius-1, ctx)
		if err != nil {
			panic(err)
		}
		candidates = append(candidates, bucketNeighbour...)
	}

	fmt.Println("candidates count: ", len(candidates))

	// Remove all non r-neighbors from the candidate set
	groups := aggregator.AggregateNeighbourSegments(&candidates, buckets)
	for key, val := range *groups {
		fmt.Printf("key: %d, val: %q, len(val): %d\n", key, val, len(val))
	}
}
