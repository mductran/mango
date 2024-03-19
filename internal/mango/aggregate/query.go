package search

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"slices"

	connector "search/internal/mango/mongo"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Hex struct {
	HexId int    `bson:"hex_id,omitempty"`
	Hex   string `bson:"hex,omitempty"`
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func splitLength(s string, buckets int) int {
	length := math.Ceil(float64(len(s)) / float64(buckets))
	return int(length)
}

func SplitSegments(s string, segment int) []string {
	i := 0
	out := []string{}
	for i < len(s) {
		out = append(out, s[i:min(len(s), i+segment)])
		i += segment
	}

	return out
}

func readCsvFile(filePath string) [][]string {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal("Unable to read input file "+filePath, err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal("Unable to parse file as CSV for "+filePath, err)
	}

	return records
}

func compare(s1, s2 string) int {
	i := 0
	count := 0
	for i < len(s1) {
		if s1[i] != s2[i] {
			count += 1
		}
		i += 1
	}
	return count
}

func SearchBucket(bucket *mongo.Collection, query string, radius int, ctx context.Context) ([]Hex, error) {
	cursor, err := bucket.Find(ctx, bson.D{})
	if err != nil {
		return []Hex{}, err
	}
	results := []Hex{}
	if err = cursor.All(ctx, &results); err != nil {
		return []Hex{}, err
	}

	var neighbours []Hex
	for _, line := range results {
		if compare(line.Hex, query) <= radius {
			neighbours = append(neighbours, line)
		}
	}

	return neighbours, nil
}

func Insert(database *mongo.Database, buckets int, ctx context.Context) {
	csv := readCsvFile("lines.csv")

	// building m substrings hashtable
	for i, line := range csv {
		database.Collection("mih").InsertOne(ctx, Hex{i + 1, line[0]})
		segments := SplitSegments(line[0], splitLength(line[0], buckets))
		for j, s := range segments {
			_, err := database.Collection(fmt.Sprintf("mih%d", j+1)).InsertOne(ctx, Hex{i + 1, s})
			if err != nil {
				panic(err)
			}
		}
	}
}

func AggregateNeighbourSegments(candidates *[]Hex, buckets int) *map[int][]string {
	groups := make(map[int][]string)
	for _, i := range *candidates {
		groups[i.HexId] = append(groups[i.HexId], i.Hex)
	}

	return &groups
}

func Query(hexstring string) {
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
		Insert(database, buckets, ctx)
	}

	// r-neighbour search for query g
	query := "7C1C024EFF958506E90347DBC343A8DBFD9820AA046EB1BD1948C6D177BFCB60"
	segments := SplitSegments(query, buckets)
	searchRadius := 30
	substringRadius := int(math.Floor(float64(searchRadius) / float64(buckets)))
	fmt.Println("substring radius: ", substringRadius)

	var candidates []Hex

	for i := 0; i < (searchRadius - substringRadius*buckets); i++ {
		bucket := database.Collection(fmt.Sprintf("mih%d", i+1))
		bucketNeighbour, err := SearchBucket(bucket, segments[i], substringRadius, ctx)
		if err != nil {
			panic(err)
		}
		candidates = append(candidates, bucketNeighbour...)
	}

	for i := (searchRadius - substringRadius*buckets); i < buckets; i++ {
		bucket := database.Collection(fmt.Sprintf("mih%d", i+1))
		bucketNeighbour, err := SearchBucket(bucket, segments[i], substringRadius-1, ctx)
		if err != nil {
			panic(err)
		}
		candidates = append(candidates, bucketNeighbour...)
	}

	fmt.Println("candidates count: ", len(candidates))

	// Remove all non r-neighbors from the candidate set
	groups := AggregateNeighbourSegments(&candidates, buckets)
	for key, val := range *groups {
		fmt.Printf("key: %d, val: %q, len(val): %d\n", key, val, len(val))
	}
}
