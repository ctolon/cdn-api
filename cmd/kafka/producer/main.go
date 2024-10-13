package main

import (
	"os"
	// "sync"

	"github.com/ctolon/cdn-api/internal/app/providers"
	"github.com/ctolon/cdn-api/internal/config"
	"github.com/ctolon/cdn-api/internal/core/queue"
	"github.com/rs/zerolog"
)

func main() {

	cfgProvider := providers.NewConfigProvider(providers.LOCAL, "", providers.DOTENV, "", "")
	cfg := config.LoadConfigIntoStruct(cfgProvider)

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Build a Kafka producer
	producer, err := queue.NewKafkaProducer(cfg.KafkaURL, nil, &logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create Kafka producer")
		return
	}

	// Build a Kafka message
	message, err := producer.BuildMessage("key", "value", "test-topic", "")
	if err != nil {
		logger.Error().Err(err).Msg("Failed to build message")
	}

	// Send the message
	err = producer.SendMessage(message, true)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to send message")
	}

}
