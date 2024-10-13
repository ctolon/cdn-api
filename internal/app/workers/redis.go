package workers

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"sync"
	"syscall"
	"time"

	"github.com/ctolon/cdn-api/internal/config"
	"github.com/ctolon/cdn-api/internal/core/cache"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

const (

	// Redis PUB-SUB Channel name.
	PUBSUB_CHANNEL_NAME string = "redis-master"
)

// RunRedisSubscriber runs redis subscriber with goroutine and profile options.
func RunRedisSubscriber(config *config.AppConfig, profile string, wgoroutine string, logger zerolog.Logger) {

	client := cache.NewRedisClientFromURLWithDbAndPool(config.RedisURL, 1, 20)

	ctx := context.Background()
	redisPubSub := client.CreateSubscribeChannel(ctx, PUBSUB_CHANNEL_NAME)
	defer func() {
		err := redisPubSub.Close()
		if err != nil {
			logger.Error().Msgf("%v", err)
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			o := fmt.Errorf("error: %+v", r)
			logger.Error().Msgf("Recovered from panic: %v", o)
		}
	}()

	logger.Info().Msg("starting Redis Pubsub Subscriber")

	// Periodically write memory profiles
	if profile == "true" {
		go func() {
			for {
				writeHeapProfile("heap.pprof")
				time.Sleep(5 * time.Second) // Adjust the interval as needed
			}
		}()
	}

	// Periodically log the number of active goroutines
	if wgoroutine == "true" {
		go func() {
			for {
				time.Sleep(1 * time.Second) // Adjust the interval as needed
				numGoroutines := runtime.NumGoroutine()
				logger.Info().Msgf("Active goroutines: %d", numGoroutines)
			}
		}()
	}

	for msg := range redisPubSub.Channel() {
		go runListenerWorker(msg, logger)
	}
}

// RunRedisSubscriberWithPool runs redis subscriber with goroutine and profile options.
func RunRedisSubscriberWithPool(config *config.AppConfig, profile string, wgoroutine string, logger zerolog.Logger) {

	const MaxWorkers = 1000

	client := cache.NewRedisClientFromURLWithDbAndPool(config.RedisURL, 1, 20)

	// Create a channel to limit the number of concurrent workers
	workerPool := make(chan struct{}, MaxWorkers)

	defer func() {
		if r := recover(); r != nil {
			o := fmt.Errorf("error: %+v", r)
			logger.Error().Msgf("Recovered from panic: %v", o)
		}
	}()

	ctx := context.Background()
	redisPubSub := client.CreateSubscribeChannel(ctx, PUBSUB_CHANNEL_NAME)
	defer func() {
		err := redisPubSub.Close()
		if err != nil {
			logger.Error().Msgf("%v", err)
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			o := fmt.Errorf("error: %+v", r)
			logger.Error().Msgf("Recovered from panic: %v", o)
		}
	}()

	logger.Info().Msg("starting redis subscriber with goroutines...")
	var wg sync.WaitGroup

	// Periodically write memory profiles
	if profile == "true" {
		go func() {
			for {
				writeHeapProfile("heap.pprof")
				time.Sleep(5 * time.Second) // Adjust the interval as needed
			}
		}()
	}

	// Periodically log the number of active goroutines
	if wgoroutine == "true" {
		go func() {
			for {
				time.Sleep(1 * time.Second) // Adjust the interval as needed
				numGoroutines := runtime.NumGoroutine()
				logger.Info().Msgf("Active goroutines: %d", numGoroutines)
			}
		}()
	}

	for msg := range redisPubSub.Channel() {
		// Acquire a spot in the worker pool
		workerPool <- struct{}{}
		wg.Add(1)

		go func(msg *redis.Message) {
			defer wg.Done()
			defer func() { <-workerPool }() // Release the spot

			runListenerWorker(msg, logger)
		}(msg)
	}

	// Wait for all workers to finish
	wg.Wait()
}

// runListenerWorker main worker function.
func runListenerWorker(
	msg *redis.Message,
	logger zerolog.Logger,
) {

	logger.Info().Msgf("Message received: %v", msg.Payload)
}

func RunListenerStreamMock(logger zerolog.Logger) {

	rdb := redis.NewClient(&redis.Options{
		Addr:     "49.13.226.113:6379",
		Password: "wOlb1jAc1ZoFPe5",
		DB:       2,
	})

	rdb2 := redis.NewClient(&redis.Options{
		Addr:     "49.13.226.113:6379",
		Password: "wOlb1jAc1ZoFPe5",
		DB:       24,
	})

	streamName := "mystream"
	groupName := "mygroup"
	consumerName := "myconsumer"

	ctx := context.Background()

	if err := rdb.XGroupCreateMkStream(ctx, streamName, groupName, "$").Err(); err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		logger.Error().Msgf("Failed to create consumer group: %v", err)
		return
	}

	fmt.Println("Consumer start..")
	for {
		result, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    groupName,
			Consumer: consumerName,
			Streams:  []string{streamName, ">"}, // ">" ile yeni mesajları oku
			Block:    0,
			Count:    1,
		}).Result()

		if err != nil {
			logger.Error().Msgf("%v", err)
		}

		for _, stream := range result {
			for _, message := range stream.Messages {
				//var deneme Deneme
				fmt.Printf("Message ID: %s, Data: %v\n", message.ID, message.Values)

				if result, err := rdb2.Set(ctx, "wwwwww", message.Values["test"], 0).Result(); err != nil {
					fmt.Println(result)
					panic(err)
				}

				//if err := phpserialize.Unmarshal(message.Values, &deneme.Value); err != nil {
				//	panic(err)
				//}

				if err := rdb.XAck(ctx, streamName, groupName, message.ID).Err(); err != nil {
					logger.Error().Msgf("Failed to acknowledge message: %v", err)
				}

				// İsteğe bağlı olarak, mesajları silmek için XTRIM komutunu kullanabilirsiniz.
				// Burada örneğin 1000 mesajdan fazla varsa, en eski olanları sileriz.
				//if err := rdb.XTrimMinMaxLen(ctx, streamName, 1000).Err(); err != nil {
				//	logger.Error().Msgf("Failed to trim stream: %v", err)
				//}
			}
		}
	}
}

// RunRedisStreamWithWorkers runs redis stream with workers.
func RunRedisStreamWithWorkers(config *config.AppConfig, streamName string, groupName string, consumerName string, consumerNum int, db int, logger zerolog.Logger) {

	client := cache.NewRedisClientFromURLWithDbAndPool(config.RedisURL, db, 20)

	// Create a context that will be cancelled on signal
	keepRunning := true
	ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()

	// Signal handling
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	sigusr1 := make(chan os.Signal, 1)
	signal.Notify(sigusr1, syscall.SIGUSR1)

	// Create Streams before schedule goroutines (for every db as redis-master)
	logger.Info().Msgf("Streams will be created...")
	logger.Info().Msgf("%s stream creating with %s groupname on db", streamName, groupName)
	if err := client.Client.XGroupCreateMkStream(ctx, streamName, groupName, "$").Err(); err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		logger.Error().Msgf("Failed to create consumer group: %v", err)
		panic(err)

	}
	logger.Info().Msgf("Streams Created: O.K.")

	// Schedule Redis Streams w/Worker Pool Pattern
	logger.Info().Msg("Consumers will be scheduled...")
	var wg sync.WaitGroup
	logger.Info().Msgf("Schedule Consumer -> %s | %s", streamName, consumerName)
	wg.Add(1)
	go runListenerStream(groupName, consumerName, streamName, &wg, ctx, client, &logger)
	<-time.After(time.Millisecond * time.Duration(10))

	logger.Info().Msgf("Consumers Scheduled: O.K. with groupname: %s", groupName)

	// Consumer graceful shutdown manager loop
	for keepRunning {
		select {
		case <-ctx.Done():
			logger.Info().Msg("terminating: context cancelled")
			keepRunning = false
		case <-sigterm:
			logger.Info().Msg("terminating: context received")
			keepRunning = false
		case <-sigusr1:
			logger.Info().Msg("terminating: signal received")
			keepRunning = false
		}
	}
	cancel()
	logger.Info().Msg("Context cancel function called.")
	//wg.Wait()

	ctxN := context.Background()

	// Cleanup consumer groups
	logger.Info().Msgf("Shutdown event received, Consumer groups will be deleted...")
	if err := client.Client.XGroupDestroy(ctxN, streamName, groupName).Err(); err != nil {
		if err.Error() == "NOGROUP" {
			logger.Warn().Msgf("Consumer group %s does not exist for stream %s", groupName, streamName)
		} else {
			logger.Error().Msgf("Failed to delete consumer group: %v", err)
		}
	} else {
		logger.Info().Msgf("Consumer group %s deleted from stream %s", groupName, streamName)
	}

	logger.Info().Msg("All Consumer groups destroyed.")

	// Close Redis Clients
	//if err := client.Close(); err != nil {
	//	logger.Warn().Msgf("An error occurred when closed redis client: %v", err)
	//}

	logger.Info().Msg("All master redis clients destroyed.")
	if err := client.Close(); err != nil {
		logger.Warn().Msgf("An error occurred when closed redis client: %v", err)
	}

	logger.Info().Msg("Finished.")

}

// RunRedisStream runs redis stream without workers.
func RunRedisStream(config *config.AppConfig, streamName string, groupName string, consumerName string, consumerNum int, db int, logger zerolog.Logger) {

	client := cache.NewRedisClientFromURLWithDbAndPool(config.RedisURL, db, 20)

	// Create a context
	ctx := context.Background()

	// Create Streams before schedule goroutines
	logger.Info().Msgf("Streams will be created...")
	if err := client.Client.XGroupCreateMkStream(ctx, streamName, groupName, "$").Err(); err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		logger.Error().Msgf("Failed to create consumer group: %v", err)
		panic(err)
	}
	logger.Info().Msgf("Streams Created: O.K.")

	// Schedule Redis Streams w/Worker Pool Pattern
	logger.Info().Msg("Consumer will be scheduled...")
	logger.Info().Msgf("Schedule Consumer -> %v | %v", streamName, consumerName)
	runListenerStream(groupName, consumerName, streamName, nil, ctx, client, &logger)

	ctxN := context.Background()

	// Cleanup consumer groups
	logger.Info().Msgf("Shutdown event received, Consumer groups will be deleted...")

	// Attempt to destroy the consumer group
	if err := client.Client.XGroupDestroy(ctxN, streamName, groupName).Err(); err != nil {
		if err.Error() == "NOGROUP" {
			logger.Warn().Msgf("Consumer group %s does not exist for stream %s", groupName, streamName)
		} else {
			logger.Error().Msgf("Failed to delete consumer group: %v", err)
		}
	} else {
		logger.Info().Msgf("Consumer group %s deleted from stream %s", groupName, streamName)
	}

	if err := client.Close(); err != nil {
		logger.Warn().Msgf("An error occurred when closed redis client: %v", err)
	}
}

func runListenerStream(
	groupName,
	consumerName,
	streamName string,
	wg *sync.WaitGroup,
	ctx context.Context,
	client *cache.RedisClient,
	logger *zerolog.Logger,
) {

	if wg != nil {
		defer wg.Done()
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Error().Msgf("Recovered from panic: %v", r)
		}
	}()

	for {

		// context management
		select {
		case <-ctx.Done():
			logger.Info().Msgf("%d | Context cancelled, stopping the listener: %s", client.Client.Options().DB, consumerName)
			return
		default:
		}

		result, err := client.Client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    groupName,
			Consumer: consumerName,
			Streams:  []string{streamName, ">"},
			//Block:    time.Duration(60) * time.Minute,
			Block: 0,
			Count: 1,
		}).Result()
		start := time.Now()

		// Common Error Handler
		if err != nil {
			if err == context.Canceled {
				logger.Info().Msg("Context cancelled while reading from stream.")
				return
			} else if err.Error() == "BUSY" {
				logger.Warn().Msg("Stream is busy, retrying after 100ms...")
				time.Sleep(100 * time.Millisecond)
				continue
			} else if err.Error() == fmt.Sprintf("NOGROUP No such key '%s' or consumer group '%s' in XREADGROUP with GROUP option", streamName, groupName) {
				if err := client.Client.XGroupCreateMkStream(ctx, streamName, groupName, "$").Err(); err != nil && (err.Error() != "BUSYGROUP Consumer Group name already exists" && err.Error() != context.Canceled.Error()) {
					logger.Warn().Msgf("Failed to create consumer group in goroutine: %v", err)
					continue
				}
			} else {
				logger.Error().Msgf("Error reading from stream: %v", err)
				continue
			}
		}

		streamHandler(consumerName, groupName, streamName, result, client, ctx, logger)
		elapsed := time.Since(start)
		logger.Info().Msgf("Took: %s", elapsed)

	} // endfor
}

// streamHandler for main function for apply business logic w/ redis streams.
func streamHandler(
	consumerName,
	groupName,
	streamName string,
	result []redis.XStream,
	client *cache.RedisClient,
	ctx context.Context,
	logger *zerolog.Logger,

) {

	for _, stream := range result {
		for _, message := range stream.Messages {

			// Log Stream Message
			logger.Info().Msgf("Consumer %s: Message ID: %s, Data: %v\n", consumerName, message.ID, message.Values)

			// Acknowledge Msg
			if err := client.Client.XAck(ctx, streamName, groupName, message.ID).Err(); err != nil {
				logger.Error().Msgf("Failed to acknowledge message: %v", err)
			}
		}
	}
}

// writeHeapProfile writes the current heap profile to a file.
func writeHeapProfile(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Could not create heap profile: %v\n", err)
		return
	}
	defer f.Close()

	runtime.GC() // Force garbage collection
	if err := pprof.WriteHeapProfile(f); err != nil {
		fmt.Printf("Could not write heap profile: %v\n", err)
	}
}
