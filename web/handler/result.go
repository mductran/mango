package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func ResultHandler(c echo.Context) error {
	return c.Render(http.StatusOK, "result", map[string]interface{}{
		"name": "Result",
		"msg":  "Matched pages",
	})
}
