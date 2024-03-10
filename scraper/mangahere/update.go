package scraper

import (
	"context"
	"fmt"
	"net/http"

	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

var MONTHS = map[string]int{
	"Jan": 1,
	"Feb": 2,
	"Mar": 3,
	"Apr": 4,
	"May": 5,
	"Jun": 6,
	"Jul": 7,
	"Aug": 8,
	"Sep": 9,
	"Oct": 10,
	"Nov": 11,
	"Dec": 12,
}

func Split(r rune) bool {
	return r == ' ' || r == ','
}

func ParseDate(s string) time.Time {
	if s == "Yesterday" {
		return time.Now().AddDate(0, 0, -1)
	} else {
		r := strings.FieldsFunc(s, Split)
		month := MONTHS[r[0]]
		date, _ := strconv.Atoi(r[1])
		year, _ := strconv.Atoi(r[2])

		parsed := time.Date(year, time.Month(month), date, 0, 0, 0, 0, time.Local)

		return parsed
	}
}

func UpdateTitle(title *mangahere.Manga) *[]mangahere.Chapter {
	detailsPage, err := http.Get("https://mangahere.cc" + title.Url)
	check(err)
	detailsPageBody, err := goquery.NewDocumentFromReader(detailsPage.Body)
	check(err)

	var newChapterList []mangahere.Chapter

	detailsPageBody.Find(".detail-main-list li a").EachWithBreak(func(i int, s *goquery.Selection) bool {

		node := s.Find(".detail-main-list-main .title2").First()

		uploadTime, _ := node.Html()

		if ParseDate(uploadTime).After(title.LastUpdate) || title.LastUpdate.IsZero() {
			newChapterList = append(newChapterList, ParseChapter(s))
			return true
		} else {
			return false
		}
	})

	return &newChapterList
}

func Update(col *mongo.Collection) {
	cursor, err := col.Find(context.TODO(), bson.D{})
	check(err)

	var results []Manga
	if err = cursor.All(context.TODO(), &results); err != nil {
		check(err)
	}

	for _, title := range results {
		newChaptersList := UpdateTitle(&title)

		fmt.Println(len(*newChaptersList))

		update := bson.M{
			"$addToSet": bson.M{
				"chapters": bson.M{
					"$each": *newChaptersList,
				},
			},
			"$set": bson.M{
				"last_update": time.Now(),
			},
		}

		filter := bson.M{"title": title.Title}
		result, err := col.UpdateOne(context.TODO(), filter, update)
		check(err)
		fmt.Println(result.ModifiedCount)
	}
}
