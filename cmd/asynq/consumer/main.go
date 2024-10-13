package main

import (
	"os"
	// "sync"

	"github.com/ctolon/cdn-api/internal/app/providers"
	"github.com/ctolon/cdn-api/internal/app/workers"
	"github.com/ctolon/cdn-api/internal/config"
	"github.com/rs/zerolog"
)

func main() {

	cfgProvider := providers.NewConfigProvider(providers.LOCAL, "", providers.DOTENV, "", "")
	cfg := config.LoadConfigIntoStruct(cfgProvider)

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Start Asynq worker
	workers.RunAsynqWorker(cfg, logger)

}
