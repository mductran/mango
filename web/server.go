package web

import (
	"net/http"
	query "search/internal/mango/aggregate"
	"search/internal/mango/hash"
	"sync"

	"github.com/labstack/echo/v4"
)

var lock sync.Mutex

type QueryResult struct {
	PHash   string            `json:"phash,omitempty" xml:"phash"`
	DHash   string            `json:"dhash,omitempty" xml:"dhash"`
	Results *map[int][]string `json:"results,omitempty" xml:"results"`
}

type HelloWorldResponse struct {
	Message string `json:"message,omitempty" xml:"message"`
}

func SearchHandler(c echo.Context) error {
	lock.Lock()
	defer lock.Unlock()

	// check if user-uploaded file or from url
	// if form's text field is empty, user submitted file
	c.Request().ParseForm()
	if c.Request().PostForm.Has("image-url") {
		url := c.FormValue("image-url")
		mat, err := hash.ReadImageFromURL(url)
		if err != nil {
			return err
		}
		dhash := hash.Dhash(mat)
		phash := hash.Phash(mat)
		results := query.Query(dhash)
		for k, v := range *query.Query(phash) {
			(*results)[k] = v
		}
		return c.JSON(http.StatusOK, QueryResult{phash, dhash, results})
	} else {
		return nil
	}
}

func Serve() {
	server := echo.New()
	server.POST("/search/", SearchHandler)
	server.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, &HelloWorldResponse{"hello client"})
	})
	server.Logger.Fatal(server.Start("localhost:5000"))
}
