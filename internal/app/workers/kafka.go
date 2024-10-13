package workers

import (
	"context"
	//"encoding/json"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/ctolon/cdn-api/internal/core/queue"
	"github.com/rs/zerolog"
)

// KafkaHandler represents a consumer with workers
type KafkaHandler struct {
	Consumer *queue.KafkaConsumer
	Logger   *zerolog.Logger
}

// NewKafkaHandler creates a new Sarama configuration for a consumer
func NewKafkaHandler(consumer *queue.KafkaConsumer, logger *zerolog.Logger) *KafkaHandler {
	return &KafkaHandler{
		Consumer: consumer,
		Logger:   logger,
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *KafkaHandler) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(h.Consumer.Ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *KafkaHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (h *KafkaHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// NOTE:
	// Do not move the code below to a goroutine.
	// The ConsumeClaim itself is called within a goroutine, see:
	// https://github.com/IBM/sarama/blob/main/consumer_group.go#L27-L29

	for {

		select {

		// onMessage
		case message, ok := <-claim.Messages():
			if !ok {
				h.Logger.Info().Msg("message channel was closed")
				return nil
			}
			// call onMessage function for consuming the message
			h.onMessage(session, message)

		// onSessionDone
		case <-session.Context().Done():
			return nil
		}
	}
}

// StartConsumer starts the consumer
func (h *KafkaHandler) StartConsumer(ctx context.Context, cancel context.CancelFunc, topics []string) {
	keepRunning := true
	consumptionIsPaused := false
	wg := &sync.WaitGroup{}

	h.Logger.Info().Msgf("Starting consumer with topics: %v", topics)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {

			// check if context was cancelled, signaling that the consumer should stop
			select {
			case <-ctx.Done():
				return
			default:
			}

			if err := h.Consumer.Client.Consume(ctx, h.Consumer.Topics, h); err != nil {
				if errors.Is(err, sarama.ErrClosedConsumerGroup) {
					return
				}
			}

			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}
			h.Consumer.Ready = make(chan bool)
		}
	}()

	// Await till the consumer has been set up
	<-h.Consumer.Ready
	h.Logger.Info().Msgf("Sarama consumer up and running with topics: %v", topics)

	sigusr1 := make(chan os.Signal, 1)
	signal.Notify(sigusr1, syscall.SIGUSR1)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	for keepRunning {
		select {
		case <-ctx.Done():
			h.Logger.Info().Msg("terminating: context cancelled")
			keepRunning = false
		case <-sigterm:
			h.Logger.Info().Msg("terminating: via signal")
			keepRunning = false
		case <-sigusr1:
			toggleConsumptionFlow(h.Consumer.Client, &consumptionIsPaused)
		}
	}
	cancel()
	wg.Wait()
	if err := h.Consumer.Client.Close(); err != nil {
		h.Logger.Error().Msgf("Error closing client: %v", err)
	}
}

// onMessage is a helper function to handle messages on the channel (callback)
func (h *KafkaHandler) onMessage(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) {
	h.Logger.Info().Msgf("Message claimed: value = %s, timestamp = %v, topic = %s", string(message.Value), message.Timestamp, message.Topic)

	//var entity any
	//if err := json.Unmarshal(message.Value, &entity); err != nil {
	//	h.Logger.Error().Msgf("Error unmarshalling message: %v", err)
	//	return
	//}

	session.MarkMessage(message, "")
	session.Commit()
	h.Logger.Info().Msgf("Message marked: value = %s, timestamp = %v, topic = %s", string(message.Value), message.Timestamp, message.Topic)
}

func (h *KafkaHandler) StartWithWorkers(ctx context.Context, cancel context.CancelFunc, consumerName string, maxWorkers int, wg *sync.WaitGroup, delayToSchedule int) {

	for i := 0; i < maxWorkers; i++ {
		//defer wg.Done()
		h.Logger.Printf("Starting worker %d", i)
		wg.Add(1)
		go h.StartConsumer(ctx, cancel, h.Consumer.Topics)
		<-time.After(time.Millisecond * time.Duration(delayToSchedule))
	}

	h.Logger.Printf("Started workers successfully")

	wg.Wait()
}

// toggleConsumptionFlow toggles the consumption flow
func toggleConsumptionFlow(client sarama.ConsumerGroup, isPaused *bool) {
	if *isPaused {
		client.ResumeAll()
		//log.Println("Resuming consumption")
	} else {
		client.PauseAll()
		//log.Println("Pausing consumption")
	}

	*isPaused = !*isPaused
}

// PartitionedKafkaWorker represents a consumer with workers
type PartitionedKafkaWorker struct {
	Consumer *queue.KafkaPartitionedConsumer
	Logger   *zerolog.Logger
}

// NewPartitionedKafkaWorker creates a new Sarama worker for partitioned topics
func NewPartitionedKafkaWorker(consumer *queue.KafkaPartitionedConsumer, logger *zerolog.Logger) *PartitionedKafkaWorker {
	return &PartitionedKafkaWorker{
		Consumer: consumer,
		Logger:   logger,
	}
}

// StartPartitionedKafkaWorker starts the consumer
func (h *PartitionedKafkaWorker) StartPartitionedKafkaWorker(wg *sync.WaitGroup) error {
	topics := h.Consumer.Topics[0]
	h.Consumer.Logger.Info().Msg("starting consumer with workers...")
	for _, partition := range h.Consumer.Partitions {
		pc, err := h.Consumer.Consumer.ConsumePartition(topics, partition, h.Consumer.Config.Consumer.Offsets.Initial)
		if err != nil {
			return err
		}

		go func(pc sarama.PartitionConsumer) {
			<-h.Consumer.Closing
			pc.AsyncClose()
		}(pc)

		wg.Add(1)
		go func(pc sarama.PartitionConsumer) {
			defer wg.Done()
			for message := range pc.Messages() {
				h.Consumer.Messages <- message
			}
		}(pc)

	}

	// Message Handler for consuming messages
	go func() {
		for msg := range h.Consumer.Messages {
			h.Logger.Printf("Partition:\t%d\n", msg.Partition)
			h.Logger.Printf("Offset:\t%d\n", msg.Offset)
			h.Logger.Printf("Key:\t%s\n", string(msg.Key))
			h.Logger.Printf("Value:\t%s\n", string(msg.Value))
		}
	}()

	h.Logger.Println("workers started. waiting for workers to finish...")
	wg.Wait()
	h.Logger.Println("done consuming topic.")
	if h.Consumer.Messages != nil {
		close(h.Consumer.Messages)
	}

	if err := h.Consumer.Consumer.Close(); err != nil {
		h.Logger.Println("failed to close consumer: ", err)
	}
	return nil
}
