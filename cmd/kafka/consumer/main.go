package main

import (
	//"context"
	"os"
	"sync"

	"github.com/ctolon/cdn-api/internal/app/providers"
	"github.com/ctolon/cdn-api/internal/app/workers"
	"github.com/ctolon/cdn-api/internal/config"
	"github.com/ctolon/cdn-api/internal/core/queue"
	"github.com/rs/zerolog"
)

func main() {

	cfgProvider := providers.NewConfigProvider(providers.LOCAL, "", providers.DOTENV, "", "")
	cfg := config.LoadConfigIntoStruct(cfgProvider)

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	topics := []string{"test-topic"}
	//ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	/*
		// 1- Build a Kafka consumer with group
		consumer, err := queue.NewKafkaConsumer(nil, &logger, cfg.KafkaURL, "test-topic", topics)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to create Kafka consumer")
			return
		}


		// Build a Kafka worker
		worker := workers.NewKafkaHandler(consumer, &logger)

		// 1.1 - Start the consumer
		//worker.StartConsumer(ctx, cancel, worker.Consumer.Topics)

		// 1.2 - Start the consumer with workers

		worker.StartWithWorkers(ctx, cancel, "test-consumer", 5, &wg, 1)
	*/

	// 2- Build a Kafka consumer with partition
	consumer, err := queue.NewKafkaPartitionedConsumer(cfg.KafkaURL, nil, topics, "all", &logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create Kafka consumer")
		return
	}

	// 2.1 Setup signal channel for sigterm and sigint
	go consumer.SetupSignalCh()

	// 2.2. Setup workers for the consumer
	worker := workers.NewPartitionedKafkaWorker(consumer, &logger)
	worker.StartPartitionedKafkaWorker(&wg)
	//handler.SetupHandler(&wg)

}
