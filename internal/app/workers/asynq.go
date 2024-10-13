package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"

	"github.com/ctolon/cdn-api/internal/config"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

// ASYNQ_TASK
const ASYNQ_TASK = "asynq-task"

// AsynqProcessor implements asynq.Handler interface.
type AsynqProcessor struct {
	logger *zerolog.Logger
}

// ProcessTask processes the task.
func (processor *AsynqProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p any
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	processor.logger.Info().Msgf("%v", p)
	return nil
}

// NewAsynqProcessor creates a new AsynqProcessor
func NewAsynqProcessor(logger zerolog.Logger) *AsynqProcessor {
	return &AsynqProcessor{
		logger: &logger,
	}
}

// RunAsynqWorker runs Asynq Worker.
func RunAsynqWorker(config *config.AppConfig, logger zerolog.Logger) {

	url, err := url.Parse(config.RedisURL)
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

	srv := asynq.NewServer(
		redisOpts,
		asynq.Config{
			// Concurrency: 10,
			Queues: map[string]int{
				ASYNQ_TASK: 10,
				//"critical": 6,
				//"default":  3,
				//"low":      1,
			},
		},
	)

	// Register the task handlers with the server.
	mux := asynq.NewServeMux()
	mux.Handle(ASYNQ_TASK, NewAsynqProcessor(logger))
	if err := srv.Run(mux); err != nil {
		panic(err)
	}
}
