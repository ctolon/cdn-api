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
	logger.Info().Msgf("Rabbit URL: %s", cfg.RabbitURL)

	listener := workers.NewRabbitListener[any](cfg.RabbitURL, &logger, &workers.QueueProps{
		QueueName:    "test-queue",
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-key",
	})

	// 1. Start the listener
	listener.Start("test-consumer")

	// 2. Start the listener with workers
	//var wg sync.WaitGroup
	//listener.StartWithWorkers("test-consumer", 5, &wg, 1)
}
