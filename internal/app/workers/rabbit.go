package workers

// https://github.com/rabbitmq/amqp091-go/blob/main/example_client_test.go
// https://stackoverflow.com/questions/41991926/how-to-detect-dead-rabbitmq-connection
// golang-struct-to-elastic-mapping

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ctolon/cdn-api/internal/core/queue"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

// RabbitListener is a Generic listener
type RabbitListener[T any] struct {
	client *queue.RabbitClient
	logger *zerolog.Logger
}

// QueueProps is Struct for represent Queue Structure in RabbitMQ
type QueueProps struct {
	QueueName    string
	ExchangeName string
	ExchangeType string
	RoutingKey   string
}

// NewRabbitListener creates a new RabbitListener
func NewRabbitListener[T any](
	addr string,
	logger *zerolog.Logger,
	props *QueueProps,
) *RabbitListener[T] {

	rc := queue.NewRabbitClient(props.QueueName, addr, logger, true, props.ExchangeName, props.ExchangeType, props.RoutingKey, true)
	l := &RabbitListener[T]{
		client: rc,
		logger: logger,
	}
	return l
}

// Start starts the listener without workers
func (l *RabbitListener[T]) Start(consumerName string) {

	l.logger.Info().Msg("Starting listener...")

	// Give the connection sometime to set up
	<-time.After(time.Second)

	// Create a context to cancel the execution
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc() // cancel execution context when the main function returns

	// Create a delivery channel to receive messages
	deliveries, err := l.client.Consume(consumerName)
	if err != nil {
		l.logger.Info().Msgf("Could not start consuming: %s\n", err)
		return
	}

	// Notify when the channel is closed
	chClosedCh := make(chan *amqp.Error, 1)
	l.client.GetChannel().NotifyClose(chClosedCh)

	// Start consuming messages
	l.logger.Info().Msgf("%s Started consuming messages: ", consumerName)
	l.onMessage(ctx, consumerName, deliveries, chClosedCh)
}

// StartWithWorkers starts the listener with workers
func (l *RabbitListener[T]) StartWithWorkers(consumerName string, maxWorkers int, wg *sync.WaitGroup, delayToSchedule int) {

	for i := 0; i < maxWorkers; i++ {
		l.logger.Info().Msgf("Starting worker %d", i)
		i := i
		wg.Add(1)
		go l.Start(fmt.Sprintf("%s-%d", consumerName, i))
		<-time.After(time.Millisecond * time.Duration(delayToSchedule))
	}

	l.logger.Info().Msg("Started workers successfully")

	wg.Wait()
}

// onMessage is a helper function to handle messages on the channel (callback)
func (l *RabbitListener[T]) onMessage(
	ctx context.Context,
	consumerName string,
	deliveries <-chan amqp.Delivery,
	chClosedCh chan *amqp.Error) {

	var err error

	for {

		select {

		// This case handles the event of closed context e.g. normal shutdown
		case <-ctx.Done():
			err = l.client.Close()
			if err != nil {
				l.logger.Error().Msgf("Close failed: %s\n", err)
				// sentry.CaptureException(err)
			}
			return

		// This case handles the event of closed channel e.g. abnormal shutdown
		case amqErr := <-chClosedCh:
			// This case handles the event of closed channel e.g. abnormal shutdown
			l.logger.Error().Msgf("AMQP Channel closed due to: %s\n", amqErr)
			<-time.After(queue.RECONNECT_DELAY)

			deliveries, err = l.client.Consume(consumerName)
			if err != nil {
				// If the AMQP channel is not ready, it will continue the loop. Next
				// iteration will enter this case because chClosedCh is closed by the
				// library
				l.logger.Error().Msg("Error trying to consume, will try again")
				// sentry.CaptureException(err)
				continue
			}

			// Re-set channel to receive notifications
			// The library closes this channel after abnormal shutdown
			chClosedCh = make(chan *amqp.Error, 1)
			l.client.GetChannel().NotifyClose(chClosedCh)

		// This case handles the event of a message received
		case delivery := <-deliveries:
			l.logger.Info().Msgf("Received message: %d\n", delivery.DeliveryTag)
			// defer sentry.RecoverWithContext(ctx)
			/*
				defer func() {
					defer sentry.Recover()
					if r := recover(); r != nil {
						l.logger.Error().Msgf("Recovered from panic: %v", r)
						// msg := kp.BuildMessage("", )
						// l.client.GetNotifyChanClose() <- amqp.ErrClosed
						// time.After(time.Second)
					}
				}()
			*/

			// Unmarshal the message
			_, sErr := l.DecodeMsg(delivery)
			if sErr != nil {
				continue
			}

			// Acknowledge the message
			l.Ack(delivery)

		} // end select
	} // end for
}

// Ack acknowledges the message
func (l *RabbitListener[T]) Ack(delivery amqp.Delivery) error {
	if err := delivery.Ack(false); err != nil {
		l.logger.Error().Msgf("Error commiting message: %s\n", err)
		return err
	}
	return nil
}

// Nack rejects the message
func (l *RabbitListener[T]) Nack(delivery amqp.Delivery) error {
	if err := delivery.Nack(false, true); err != nil {
		l.logger.Error().Msgf("Error rejecting message: %s\n", err)
		return err
	}
	return nil
}

// DecodeMsg decodes the rabbit message into the entity
func (l *RabbitListener[T]) DecodeMsg(delivery amqp.Delivery) (*T, error) {
	var entity T
	err := json.Unmarshal(delivery.Body, &entity)
	if err != nil {
		l.logger.Error().Msgf("Error unmarshalling message: %s\n", err)
		if err := delivery.Ack(false); err != nil {
			l.logger.Error().Msgf("Error acknowledging message: %s\n", err)
		}
		return nil, err
	}
	// entity.PreProcess()
	return &entity, nil
}
