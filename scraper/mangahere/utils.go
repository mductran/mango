package scraper

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/buke/quickjs-go"
)

type Manga struct {
	Url          string `json:"url,omitempty"`
	Title        string `json:"title,omitempty"`
	Artist       string `json:"artist,omitempty"`
	Author       string `json:"author,omitempty"`
	Description  string `json:"description,omitempty"`
	Genre        string `json:"genre,omitempty"`
	Status       int    `json:"status,omitempty"`
	ThumbnailUrl string `json:"thumbnail_url,omitempty"`
}

type Chapter struct {
	Url       string  `json:"url,omitempty"`
	Name      string  `json:"name,omitempty"`
	Number    float32 `json:"number,omitempty"` // for .5 Chapter
	Scanlator string  `json:"scanlator,omitempty"`
}

type Page struct {
	Index    int    `json:"index,omitempty"`
	Url      string `json:"url,omitempty"`
	ImageUrl string `json:"image_url,omitempty"`
}

var Genres = map[string]int{
	"Action":        1,
	"Adventure":     2,
	"Comedy":        3,
	"Fantasy":       4,
	"Historical":    5,
	"Horror":        6,
	"Martial Arts":  7,
	"Mystery":       8,
	"Romance":       9,
	"Shounen Ai":    10,
	"Supernatural":  11,
	"Drama":         12,
	"Shounen":       13,
	"School Life":   14,
	"Shoujo":        15,
	"Gender Bender": 16,
	"Josei":         17,
	"Psychological": 18,
	"Seinen":        19,
	"Slice of Life": 20,
	"Sci-fi":        21,
	"Ecchi":         22,
	"Harem":         23,
	"Shoujo Ai":     24,
	"Yuri":          25,
	"Mature":        26,
	"Tragedy":       27,
	"Yaoi":          28,
	"Doujinshi":     29,
	"Sports":        30,
	"Adult":         31,
	"One Shot":      32,
	"Smut":          33,
	"Mecha":         34,
	"Shotacon":      35,
	"Lolicon":       36,
	"Webtoons":      37,
}

var (
	BaseUrl       = "https://www.mangahere.cc"
	DefaultLang   = "en"
	MangaSelector = ".manga-list-1-list li"
)

func ValidateUrl(url string) bool {
	return false
}

func LatestUpdateRequest(page int) (*http.Response, error) {
	url := BaseUrl + "/directory/" + strconv.Itoa(page) + ".htm"
	client := http.Client{Timeout: time.Second * 30}
	request, _ := http.NewRequest("GET", url, nil)

	res, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func parsePopular(node *goquery.Selection) *Manga {
	manga := Manga{}

	titleNode := node.Find("a").First()
	title, _ := titleNode.Attr("title")
	link, _ := titleNode.Attr("href")

	thumbnailLink := ""
	thumbnailNode := node.Find("img.manga-list-1-cover").First()
	if thumbnailNode != nil {
		thumbnailLink, _ = thumbnailNode.Attr("src")
	}

	manga.Title = title
	manga.Url = link
	manga.ThumbnailUrl = thumbnailLink

	return &manga
}

func ParseManga(body *goquery.Selection) *Manga {
	manga := Manga{}

	authorNode := body.Find(".detail-info-right-say > a").First()
	author := ""
	if authorNode != nil {
		author = authorNode.Text()
	}
	manga.Author = author

	genreNode := body.Find(".detail-info-right-tag-list > a").First()
	genre := ""
	if genreNode != nil {
		genre = genreNode.Text()
	}
	manga.Genre = genre

	descriptionNode := body.Find(".fullcontent").First()
	description := ""
	if descriptionNode != nil {
		description = descriptionNode.Text()
	}
	manga.Description = description

	thumbnailNode := body.Find("img.detail-info-cover-img").First()
	thumbnailLink := ""
	if thumbnailNode != nil {
		thumbnailLink, _ = thumbnailNode.Attr("src")
	}
	manga.ThumbnailUrl = thumbnailLink

	return &manga
}

func ParseChapter(node *goquery.Selection) *Chapter {
	chapter := Chapter{}

	linkNode := node.Find("a").First()
	chapterLink := ""
	if linkNode != nil {
		chapterLink, _ = linkNode.Attr("href")
	}
	chapter.Url = chapterLink

	nameNode := node.Find("a p.title3").First()
	name := ""
	if nameNode != nil {
		name = nameNode.Text()
	}
	chapter.Name = name

	return &chapter
}

func ParseMangaListByPage(page int) *[]Manga {
	url := fmt.Sprintf("https://www.mangahere.cc/directory/%d", page)

	response, err := http.Get(url)
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

	mangas := []Manga{}
	document.Find(".manga-list-1-list li").Each(func(i int, s *goquery.Selection) {
		fmt.Println(i, s.Text())
		mangas = append(mangas, *parsePopular(s))
	})

	fmt.Println(len(mangas))
	fmt.Printf("%+v\n", mangas[0])

	return &mangas
}

func ParsePopularPage(pageLink string) {
	fmt.Println("parsing popular page ", pageLink)
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

	// get page content
	document.Find(mangaSelector).Each(func(i int, s *goquery.Selection) {
		manga := parsePopular(s)
		fmt.Printf("%+v\n\n", manga)
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
		ParsePopularPage(nextPage)
	} else {
		return
	}
}

func IsEvalFunction(script string) bool {
	script = strings.TrimSpace(script)
	return len(script) > 4 && script[:4] == "eval"
}

func DropLastPageIfBroken(pages *[]Page) *[]Page {
	lastTwo := (*pages)[len(*pages)-2:]
	pageNums := []int{}
	for _, p := range lastTwo {
		if p.Url != "" {
			v := strings.Split(p.Url, "/")
			s := v[len(v)-1]
			pn := strings.Split(s, ".")[0]
			pageNumber, err := strconv.Atoi(pn[len(pn)-2:])
			if err != nil {
				*pages = (*pages)[:len(*pages)-1]
				return pages
			}
			pageNums = append(pageNums, pageNumber)
		} else {
			// remove last page
			*pages = (*pages)[:len(*pages)-1]
			return pages
		}
	}

	if pageNums[0] == 0 && pageNums[1] == 1 {
		return pages
	} else if pageNums[1]-pageNums[0] == 1 {
		return pages
	}
	*pages = (*pages)[:len(*pages)-1]
	return pages
}

func ExtractSecretKey(doc string) (string, error) {
	runtime := quickjs.NewRuntime()
	defer runtime.Close()
	context := runtime.NewContext()
	defer context.Close()

	scriptStart := strings.Index(doc, "eval(function(p,a,c,k,e,d)")
	scriptEnd := strings.Index(doc[scriptStart:], "</script>")
	script := doc[scriptStart : scriptStart+scriptEnd]
	script = strings.TrimPrefix(script, "eval")

	s, err := context.Eval(script)
	if err != nil {
		return "", err
	}
	deObsfucatedScript := s.String()

	keyStart := strings.Index(deObsfucatedScript, "'")
	keyEnd := strings.Index(deObsfucatedScript, ";")
	keyString := deObsfucatedScript[keyStart:keyEnd]

	key, err := context.Eval(keyString)
	if err != nil {
		return "", nil
	}

	return key.String(), nil
}

func ParseManhwaPageList(doc *goquery.Document, context *quickjs.Context) *[]Page {
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

func ParseMangaPageList(doc *goquery.Document, url string, context *quickjs.Context) *[]Page {
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

		// fmt.Println("base link: ", baseLink)

		imageLinkStart := strings.Index(script, "pvalue=") + 9
		imageLinkEnd := strings.Index(script[imageLinkStart:], "\"")
		imageLink := script[imageLinkStart : imageLinkStart+imageLinkEnd]

		// fmt.Println("image link: ", imageLink)

		pages = append(pages, Page{i - 1, "", "https:" + baseLink + imageLink})
	}

	pages = *DropLastPageIfBroken(&pages)

	return &pages
}

func ParsePageList(doc *goquery.Document, url string) (*[]Page, error) {
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
		ParseMangaPageList(doc, url, context)
	} else {
		ParseManhwaPageList(doc, context)
	}

	return &[]Page{}, nil
}
