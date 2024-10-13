package routes

import (
	"net/http"

	"github.com/ctolon/cdn-api/internal/app/api/v1/handlers"
	"github.com/ctolon/cdn-api/internal/app/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// RegisterRoutes registers all routes for the API
func RegisterRoutes(
	minioService *service.MinioService,
) *echo.Echo {

	// Create a new echo instance
	e := echo.New()

	// Swagger
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Remove trailing slash
	e.Pre(middleware.RemoveTrailingSlash())
	//e.Pre(middleware.HTTPSNonWWWRedirect())
	//e.Pre(middleware.WWWRedirect())

	// Middleware
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())
	e.Use(middleware.Secure())
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("32M"))

	//e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
	//	Format: "method=${method}, uri=${uri}, status=${status}\n",
	//}))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
	}))

	//** routes **//

	// /api/v1
	apiGroup := e.Group("/api/v1")
	healtcheckHandler := handlers.NewHealthCheckHandler()
	apiGroup.GET("/healthcheck", healtcheckHandler.Healtcheck)

	// /minio
	minioGroup := apiGroup.Group("/minio")
	minioHandler := handlers.NewMinioHandler(minioService)
	minioGroup.GET("/list-objects", minioHandler.ListObjects)
	minioGroup.GET("/get-object", minioHandler.GetObject)

	return e
}
