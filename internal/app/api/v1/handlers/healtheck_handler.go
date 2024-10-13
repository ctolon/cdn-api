package handlers

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

type HealthCheckHandler struct {
}

func NewHealthCheckHandler() *HealthCheckHandler {
	return &HealthCheckHandler{}
}

func (h *HealthCheckHandler) Healtcheck(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}
