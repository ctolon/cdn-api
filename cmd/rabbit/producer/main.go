package main

import (
	"os"
	"time"

	"github.com/ctolon/cdn-api/internal/app/providers"
	"github.com/ctolon/cdn-api/internal/config"
	"github.com/ctolon/cdn-api/internal/core/queue"
	"github.com/rs/zerolog"
)

func main() {

	cfgProvider := providers.NewConfigProvider(providers.LOCAL, "", providers.DOTENV, "", "")
	cfg := config.LoadConfigIntoStruct(cfgProvider)

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	logger.Info().Msgf("Rabbit URL: %s", cfg.RabbitURL)

	client := queue.NewRabbitClient("test-queue", cfg.RabbitURL, &logger, true, "test-exchange", "direct", "test-key", false)

	// TODO need wait for connection to be established
	time.Sleep(5 * time.Second)
	if err := client.Push([]byte("Hello, World!")); err != nil {
		logger.Error().Err(err).Msg("Error pushing message")
	}
}
