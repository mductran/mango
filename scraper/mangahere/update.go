package scraper

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

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

func check(e error) {
	if e != nil {
		panic(e)
	}
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

		parsed := time.Date(year, time.Month(month), date, 0, 0, 0, 0, nil)

		return parsed
	}
}

func UpdateTitle(title *Manga) *[]Chapter {
	detailsPage, err := http.Get("https://mangahere.cc" + title.Url)
	check(err)
	detailsPageBody, err := goquery.NewDocumentFromReader(detailsPage.Body)
	check(err)

	var newChapterList []Chapter

	detailsPageBody.Find(".detail-main-list li a").EachWithBreak(func(i int, s *goquery.Selection) bool {

		node := s.Find(".detail-main-list-main .title2").First()

		uploadTime, _ := node.Html()

		if ParseDate(uploadTime).After(title.LastUpdate) {
			newChapterList = append(newChapterList, *ParseChapter(s))
			return true
		} else {
			return false
		}
	})

	return &newChapterList
}
