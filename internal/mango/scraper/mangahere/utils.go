package scraper

import (
	"strconv"
	"strings"
	"time"

	"github.com/buke/quickjs-go"
)

type Manga struct {
	Url          string     `bson:"url,omitempty"`
	Title        string     `bson:"title,omitempty"`
	Artist       string     `bson:"artist,omitempty"`
	Author       string     `bson:"author,omitempty"`
	Description  string     `bson:"description,omitempty"`
	Genre        string     `bson:"genre,omitempty"`
	Status       int        `bson:"status,omitempty"`
	ThumbnailUrl string     `bson:"thumbnail_url,omitempty"`
	LastUpdate   time.Time  `bson:"last_update,omitempty"`
	Chapters     *[]Chapter `bson:"chapters,omitempty"`
}

type Chapter struct {
	MangaId   string  `bson:"manga_id,omitempty"`
	Url       string  `bson:"url,omitempty"`
	Name      string  `bson:"name,omitempty"`
	Number    float32 `bson:"number,omitempty"` // float for .5 Chapter
	Scanlator string  `bson:"scanlator,omitempty"`
	Pages     []Page  `bson:"pages,omitempty"`
}

type Page struct {
	Index    int    `bson:"index,omitempty"`
	Url      string `bson:"url,omitempty"`
	ImageUrl string `bson:"image_url,omitempty"`
}

var (
	BaseUrl       = "https://www.mangahere.cc"
	DefaultLang   = "en"
	MangaSelector = ".manga-list-1-list li"
)

func ValidateUrl(url string) bool {
	return false
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
