package handlers

// https://github.com/emretiryaki/echoErrorHandlerExample/tree/main
import (
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
)

func customHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}
	c.Logger().Error(err)
	errorPage := fmt.Sprintf("%d.html", code)
	if err := c.File(errorPage); err != nil {
		c.Logger().Error(err)
	}
}
