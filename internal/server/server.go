package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ctolon/cdn-api/internal/app/api/v1/routes"
	"github.com/ctolon/cdn-api/internal/app/service"
	"github.com/ctolon/cdn-api/internal/config"
	log "github.com/ctolon/cdn-api/internal/logger"
)

func NewServer(
	logger log.LoggerAdapter,
	config *config.AppConfig,
	minioService *service.MinioService,
) {

	// Initialize the repository layer

	// Initialize the service layer

	// Initialize the API routes
	e := routes.RegisterRoutes(minioService)

	// Start the server with the API routes
	var handler http.Handler = e
	s := http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: handler,
		//ReadTimeout: 30 * time.Second, // customize http.Server timeouts
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	logger.Info("Starting server", log.LogFields{"port": config.Port})
	go func() {
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error("Server error", err, log.LogFields{"error": err})
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

}
