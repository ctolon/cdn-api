package main

import (
	"encoding/json"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/ctolon/cdn-api/internal/app/providers"
	"github.com/ctolon/cdn-api/internal/config"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

func main() {

	cfgProvider := providers.NewConfigProvider(providers.LOCAL, "", providers.DOTENV, "", "")
	cfg := config.LoadConfigIntoStruct(cfgProvider)

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Parse Redis URL
	url, err := url.Parse(cfg.RedisURL)
	if err != nil {
		panic(err)
	}
	host, port, err := net.SplitHostPort(url.Host)
	if err != nil {
		panic(err)
	}
	password, exist := url.User.Password()
	if !exist {
		password = ""
	}

	redisOpts := asynq.RedisClientOpt{Addr: host + ":" + port}
	if password != "" {
		redisOpts.Password = password
	}

	// Create a new client
	client := asynq.NewClient(redisOpts)

	// Create a payload
	payload, err := json.Marshal([]byte(`{"name": "John Doe"}`))
	if err != nil {
		logger.Fatal().Msgf("json error: %v", err)
	}

	// Enqueue a task
	task := asynq.NewTask("asynq-task", payload, asynq.MaxRetry(5), asynq.Timeout(20*time.Minute), asynq.Queue("asynq-task"))
	info, err := client.Enqueue(task)
	if err != nil {
		logger.Fatal().Msgf("could not enqueue task: %v", err)
	}
	logger.Info().Msgf("enqueued task: id=%s queue=%s", info.ID, info.Queue)

}
