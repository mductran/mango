package scraper

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/buke/quickjs-go"
	"go.mongodb.org/mongo-driver/mongo"
)

func parseManhwaPageList(doc *goquery.Document, context *quickjs.Context) *[]Page {
	pages := []Page{}

	var script string
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		html, _ := s.Html()
		if IsEvalFunction(html) {
			script = html
		}
	})

	script = html.UnescapeString(script) // un-escaped special character in script breaks Eval function
	script = strings.TrimSpace(script)
	script = strings.TrimLeft(script, "eval")

	deObsfucatedScript, err := context.Eval(script)
	if err != nil {
		panic(err)
	}

	splitAfter := strings.Index(deObsfucatedScript.String(), "newImgs=[") + len("newImgs=[")
	splitBefore := strings.Index(deObsfucatedScript.String(), "'];") + 1

	urls := strings.Split(deObsfucatedScript.String()[splitAfter:splitBefore], ",")
	for i, link := range urls {
		pages = append(pages, Page{i, "", "https://" + link})
	}

	return &pages
}

func parseMangaPageList(doc *goquery.Document, url string, context *quickjs.Context) *[]Page {
	pages := []Page{}

	html, err := doc.Html()
	if err != nil {
		panic(err)
	}
	secretKey, err := ExtractSecretKey(html)
	if err != nil {
		panic(err)
	}
	chapterIdStart := strings.Index(html, "chapterid")
	chapterIdEnd := strings.Index(html[chapterIdStart:], ";")

	chapterId := html[chapterIdStart+11 : chapterIdStart+chapterIdEnd]
	chapterId = strings.TrimSpace(chapterId)

	var pn string
	var exists bool
	chapterPagesElement := doc.Find(".pager-list-left > span").First()

	aTags := chapterPagesElement.Find("a")
	aTags.Each(func(i int, s *goquery.Selection) {
		if i == aTags.Length()-2 {
			pn, exists = s.Attr("data-page")
			if !exists {
				panic("page number does not exist")
			}
		}
	})

	pageNumber, err := strconv.Atoi(pn)
	if err != nil {
		panic(err)
	}
	pageBase := url[:strings.LastIndex(url, "/")]

	for i := 0; i < pageNumber; i++ {
		pageLink := fmt.Sprintf("%s/chapterfun.ashx?cid=%s&page=%d&key=%s", pageBase, chapterId, i, secretKey)

		var responseText string
		for j := 0; j < 3; j++ {
			request, err := http.NewRequest(http.MethodGet, pageLink, nil)
			if err != nil {
				panic(err)
			}
			request.Header.Set("Referer", url)
			request.Header.Set("Accept", "*/*")
			request.Header.Set("Accept-Language", "en-US;en;q=0.9")
			request.Header.Set("Connection", "keep-alive")
			request.Header.Set("Host", "www.mangahere.cc")
			// TODO: set user-agent from https://explore.whatismybrowser.com/useragents/explore/
			request.Header.Set("User-Agent", "")
			request.Header.Set("X-Requested-With", "XMLHttpRequest")

			response, err := http.DefaultClient.Do(request)
			if err != nil {
				panic(err)
			}
			bodyBytes, err := io.ReadAll(response.Body)
			if err != nil {
				panic(err)
			}
			if len(bodyBytes) > 0 {
				responseText = string(bodyBytes)
				responseText = strings.TrimLeft(responseText, "eval")
				break
			} else {
				secretKey = ""
			}
		}

		deObsfucatedScript, err := context.Eval(responseText)
		if err != nil {
			panic(err)
		}
		script := deObsfucatedScript.String()

		baseLinkStart := strings.Index(script, "pix=") + 5
		baseLinkEnd := strings.Index(script[baseLinkStart:], ";") - 1
		baseLink := script[baseLinkStart : baseLinkStart+baseLinkEnd]
		baseLink = strings.ReplaceAll(baseLink, ".org", ".cc")

		imageLinkStart := strings.Index(script, "pvalue=") + 9
		imageLinkEnd := strings.Index(script[imageLinkStart:], "\"")
		imageLink := script[imageLinkStart : imageLinkStart+imageLinkEnd]

		pages = append(pages, Page{i - 1, "", "https:" + baseLink + imageLink})
	}

	pages = *DropLastPageIfBroken(&pages)

	return &pages
}

func ParsePageList(doc *goquery.Document, url string) *[]Page {
	scrollBarSelector := "script[src*=chapter_bar]"
	scrollBar := doc.Find(scrollBarSelector)

	runtime := quickjs.NewRuntime()
	defer runtime.Close()
	context := runtime.NewContext()
	defer context.Close()

	// parse manga/manhwa pages
	// manhwa reader use continuous scroll -> has scrollbar
	// manga reader has no scrollbar
	if scrollBar.Length() == 0 {
		// is manga
		return parseMangaPageList(doc, url, context)
	} else {
		return parseManhwaPageList(doc, context)
	}
}

func ParseChapter(s *goquery.Selection) *Chapter {
	chapterLink, _ := s.Attr("href")
	chapterTitle, _ := s.Attr("title")

	fmt.Println("parsed chapter: ", chapterTitle)
	response, err := http.Get("https://www.mangahere.cc" + chapterLink)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()
	doc, _ := goquery.NewDocumentFromReader(response.Body)

	pageList := ParsePageList(doc, "https://www.mangahere.cc"+chapterLink)

	return &Chapter{
		Url: chapterLink, Name: chapterTitle, Pages: *pageList,
	}
}

func parseChapterList(url string, chapterList *[]Chapter) {
	response, err := http.Get("https://www.mangahere.cc" + url)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		panic(err)
	}

	doc.Find(".detail-main-list li a").Each(func(i int, s *goquery.Selection) {
		chapter := ParseChapter(s)
		*chapterList = append(*chapterList, *chapter)
	})
}

func parseThumbnail(node *goquery.Selection) *Manga {
	title := Manga{}

	titleNode := node.Find("a").First()
	name, _ := titleNode.Attr("title")
	link, _ := titleNode.Attr("href")

	thumbnailLink := ""
	thumbnailNode := node.Find("img.manga-list-1-cover").First()
	if thumbnailNode != nil {
		thumbnailLink, _ = thumbnailNode.Attr("src")
	}

	fmt.Printf("parsing manga: %s\n", name)
	title.Title = name
	title.Url = link
	title.ThumbnailUrl = thumbnailLink
	var chapterList []Chapter
	parseChapterList(link, &chapterList)
	title.Chapters = &chapterList

	return &title
}

func Crawl(pageLink string, collection *mongo.Collection) {
	nextPageSelector := ".pager-list-left a"
	mangaSelector := ".manga-list-1-list.line li"

	response, err := http.Get(pageLink)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		panic("status code not 200")
	}
	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		panic(err)
	}

	// gather mangas in page
	document.Find(mangaSelector).Each(func(i int, s *goquery.Selection) {
		manga := parseThumbnail(s)
		// *mangaList = append(*mangaList, *manga)
		result, err := collection.InsertOne(context.TODO(), manga)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("inserted %v\n", result.InsertedID)
	})

	// navigate to next page
	nextPageSelections := document.Find(nextPageSelector)
	nextPageButton := nextPageSelections.Last()
	nextPageHref, exists := nextPageButton.Attr("href")
	if !exists {
		fmt.Println("cannot find next page button")
	}

	nextPage := ""
	if strings.Contains(nextPageHref, "htm") {
		s := strings.Split(nextPageHref, "/")
		page := s[len(s)-1]
		nextPage = "https://www.mangahere.cc/directory/" + page
	}

	if nextPage != "" {
		Crawl(nextPage, collection)
	} else {
		return
	}
}
