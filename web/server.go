package web

import (
	"errors"
	"fmt"
	"html/template"
	"io"

	"search/web/handler"

	"github.com/labstack/echo/v4"
)

type TemplateRegistry struct {
	templates map[string]*template.Template
}

func (t *TemplateRegistry) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, ok := t.templates[name]
	if !ok {
		fmt.Println("template not found", name)
		err := errors.New("Template not found -> " + name)
		return err
	}
	return tmpl.ExecuteTemplate(w, "base.html", data)
}

func Serve() {
	server := echo.New()

	templates := make(map[string]*template.Template)
	templates["home"] = template.Must(template.ParseFiles("web/templates/home.html", "web/templates/base.html"))
	templates["result"] = template.Must(template.ParseFiles("web/templates/result.html", "web/templates/base.html"))

	server.Renderer = &TemplateRegistry{
		templates: templates,
	}

	server.GET("/", handler.HomeHandler)
	server.GET("/results", handler.ResultHandler)

	server.Logger.Fatal(server.Start(":8080"))
}
