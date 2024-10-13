package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

// RabbitClient is the base struct for handling connection recovery, consumption and
// publishing. Note that this struct has an internal mutex to safeguard against
// data races. As you develop and iterate over this example, you may need to add
// further locks, or safeguards, to keep your application safe from data races
type RabbitClient struct {
	m               *sync.Mutex
	queueName       string
	exchange        string
	kind            string
	key             string
	connection      *amqp.Connection
	channel         *amqp.Channel
	done            chan bool
	notifyConnClose chan *amqp.Error
	notifyChanClose chan *amqp.Error
	notifyConfirm   chan amqp.Confirmation
	isReady         bool
	autoack         bool
	initQueue       bool
	logger          *zerolog.Logger
}

const (
	// When reconnecting to the server after connection failure
	RECONNECT_DELAY = 5 * time.Second

	// When setting up the channel after a channel exception
	REINIT_DELAY = 2 * time.Second

	// When resending messages the server didn't confirm
	RESEND_DELAY = 5 * time.Second
)

var (
	errNotConnected  = errors.New("not connected to a server")
	errAlreadyClosed = errors.New("already closed: not connected to the server")
	errShutdown      = errors.New("client is shutting down")
)

// New creates a new consumer state instance, and automatically
// attempts to connect to the server.
// queueName is the name of the queue to connect to. It can be empty as "" on init.
// and set later with CreateQueueWithBinding or SetQueueName
func NewRabbitClient(queueName, addr string, l *zerolog.Logger, failover bool, exchange, kind, key string, initQueue bool) *RabbitClient {

	client := RabbitClient{
		m:         &sync.Mutex{},
		logger:    l,
		queueName: queueName,
		done:      make(chan bool),
		autoack:   false,
		exchange:  exchange,
		kind:      kind,
		key:       key,
		initQueue: initQueue,
	}
	if failover {
		go client.handleReconnect(addr)
	}
	return &client
}

// SetAutoAck sets the autoack flag to true
func (client *RabbitClient) SetAutoAck(autoack bool) {
	client.autoack = true
}

// UnsetAutoAck sets the autoack flag to false
func (client *RabbitClient) UnsetAutoAck(autoack bool) {
	client.autoack = true
}

// GetAutoAck returns the autoack flag
func (client *RabbitClient) GetAutoAck() bool {
	return client.autoack
}

// GetChannel returns the channel
func (client *RabbitClient) GetChannel() *amqp.Channel {
	return client.channel
}

// GetnotifyChanClose chan *amqp.Error
func (client *RabbitClient) GetNotifyChanClose() chan *amqp.Error {
	return client.notifyChanClose
}

// SetQueueName sets the queue name
func (client *RabbitClient) SetQueueName(queueName string) {
	client.queueName = queueName
}

// handleReconnect will wait for a connection error on
// notifyConnClose, and then continuously attempt to reconnect.
func (client *RabbitClient) handleReconnect(addr string) {
	for {
		client.m.Lock()
		client.isReady = false
		client.m.Unlock()

		client.logger.Info().Msg("Attempting to connect")

		conn, err := client.connect(addr)
		if err != nil {
			client.logger.Error().Msg("Failed to connect. Retrying...")

			select {
			case <-client.done:
				return
			case <-time.After(RECONNECT_DELAY):
			}
			continue
		}

		if done := client.handleReInit(conn); done {
			break
		}
	}
}

// connect will create a new AMQP connection
func (client *RabbitClient) connect(addr string) (*amqp.Connection, error) {
	conn, err := amqp.Dial(addr)
	if err != nil {
		return nil, err
	}

	client.changeConnection(conn)
	client.logger.Info().Msg("Connected!")
	return conn, nil
}

// handleReInit will wait for a channel error
// and then continuously attempt to re-initialize both channels
func (client *RabbitClient) handleReInit(conn *amqp.Connection) bool {
	for {
		client.m.Lock()
		client.isReady = false
		client.m.Unlock()

		err := client.init(conn)
		if err != nil {
			client.logger.Error().Msg("Failed to initialize channel. Retrying...")

			select {
			case <-client.done:
				return true
			case <-client.notifyConnClose:
				client.logger.Warn().Msg("Connection closed. Reconnecting...")
				return false
			case <-time.After(REINIT_DELAY):
			}
			continue
		}

		select {
		case <-client.done:
			return true
		case <-client.notifyConnClose:
			client.logger.Warn().Msg("Connection closed. Reconnecting...")
			return false
		case <-client.notifyChanClose:
			client.logger.Warn().Msg("Channel closed. Re-running init...")
		}
	}
}

// init will initialize channel & declare queue
func (client *RabbitClient) init(conn *amqp.Connection) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}

	err = ch.Confirm(false)
	if err != nil {
		return err
	}

	if client.initQueue {
		err = client.CreateQueueWithBinding(ch, client.queueName, client.exchange, client.kind, client.key)
		/*
			_, err = ch.QueueDeclare(
				client.queueName,
				true,  // Durable
				false, // Delete when unused
				false, // Exclusive
				false, // No-wait
				nil,   // Arguments
			)
		*/
		if err != nil {
			return err
		}
	}

	client.changeChannel(ch)
	client.m.Lock()
	client.isReady = true
	client.m.Unlock()
	client.logger.Info().Msg("Setup!")

	return nil
}

// changeConnection takes a new connection to the queue,
// and updates the close listener to reflect this.
func (client *RabbitClient) changeConnection(connection *amqp.Connection) {
	client.connection = connection
	client.notifyConnClose = make(chan *amqp.Error, 1)
	client.connection.NotifyClose(client.notifyConnClose)
}

// changeChannel takes a new channel to the queue,
// and updates the channel listeners to reflect this.
func (client *RabbitClient) changeChannel(channel *amqp.Channel) {
	client.channel = channel
	client.notifyChanClose = make(chan *amqp.Error, 1)
	client.notifyConfirm = make(chan amqp.Confirmation, 1)
	client.channel.NotifyClose(client.notifyChanClose)
	client.channel.NotifyPublish(client.notifyConfirm)
}

// Push will push data onto the queue, and wait for a confirmation.
// This will block until the server sends a confirmation. Errors are
// only returned if the push action itself fails, see UnsafePush.
func (client *RabbitClient) Push(data []byte) error {
	client.m.Lock()
	if !client.isReady {
		client.m.Unlock()
		return errors.New("failed to push: not connected")
	}
	client.m.Unlock()
	for {
		err := client.UnsafePush(data)
		if err != nil {
			client.logger.Error().Msg("Push failed. Retrying...")
			select {
			case <-client.done:
				return errShutdown
			case <-time.After(RESEND_DELAY):
			}
			continue
		}
		confirm := <-client.notifyConfirm
		if confirm.Ack {
			client.logger.Printf("Push confirmed [%d]!", confirm.DeliveryTag)
			return nil
		}
	}
}

// UnsafePush will push to the queue without checking for
// confirmation. It returns an error if it fails to connect.
// No guarantees are provided for whether the server will
// receive the message.
func (client *RabbitClient) UnsafePush(data []byte) error {
	client.m.Lock()
	if !client.isReady {
		client.m.Unlock()
		return errNotConnected
	}
	client.m.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return client.channel.PublishWithContext(
		ctx,
		client.exchange, // Exchange
		client.key,      // Routing key
		false,           // Mandatory
		false,           // Immediate
		amqp.Publishing{
			ContentType:  "text/plain",
			Body:         data,
			DeliveryMode: amqp.Persistent,
		},
	)
}

// Consume will continuously put queue items on the channel.
// It is required to call delivery.Ack when it has been
// successfully processed, or delivery.Nack when it fails.
// Ignoring this will cause data to build up on the server.
func (client *RabbitClient) Consume(consumerName string) (<-chan amqp.Delivery, error) {
	client.m.Lock()
	if !client.isReady {
		client.m.Unlock()
		return nil, errNotConnected
	}
	client.m.Unlock()

	if err := client.channel.Qos(
		1,     // prefetchCount
		0,     // prefetchSize
		false, // global
	); err != nil {
		return nil, err
	}

	return client.channel.Consume(
		client.queueName,
		consumerName,   // Consumer
		client.autoack, // Auto-Ack
		false,          // Exclusive
		false,          // No-local
		false,          // No-Wait
		nil,            // Args
	)
}

// Consume will continuously put queue items on the channel.
// It is required to call delivery.Ack when it has been
// successfully processed, or delivery.Nack when it fails.
// Ignoring this will cause data to build up on the server.
func (client *RabbitClient) ConsumeWithCtx(ctx context.Context, consumerName string) (<-chan amqp.Delivery, error) {
	client.m.Lock()
	if !client.isReady {
		client.m.Unlock()
		return nil, errNotConnected
	}
	client.m.Unlock()

	if err := client.channel.Qos(
		1,     // prefetchCount
		0,     // prefetchSize
		false, // global
	); err != nil {
		return nil, err
	}

	return client.channel.ConsumeWithContext(
		ctx, // Context
		client.queueName,
		consumerName,   // Consumer
		client.autoack, // Auto-Ack
		false,          // Exclusive
		false,          // No-local
		false,          // No-Wait
		nil,            // Args
	)
}

// Close will cleanly shut down the channel and connection.
func (client *RabbitClient) Close() error {
	client.m.Lock()
	// we read and write isReady in two locations, so we grab the lock and hold onto
	// it until we are finished
	defer client.m.Unlock()

	if !client.isReady {
		return errAlreadyClosed
	}
	close(client.done)
	err := client.channel.Close()
	if err != nil {
		return err
	}
	err = client.connection.Close()
	if err != nil {
		return err
	}

	client.isReady = false
	return nil
}

// QueueDeclare declares a new queue
func (client *RabbitClient) QueueDeclare(ch *amqp.Channel, queue string) error {

	client.logger.Info().Msgf("Declaring queue: %s\n", queue)
	_, err := ch.QueueDeclare(
		queue, // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return err
	}
	return nil
}

func (client *RabbitClient) ExchangeDeclare(ch *amqp.Channel, exchange, kind string) error {

	client.logger.Info().Msgf("Declaring exchange: %s\n", exchange)
	err := ch.ExchangeDeclare(
		exchange, // name
		kind,     // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		return err
	}
	return nil
}

func (client *RabbitClient) BindQueue(ch *amqp.Channel, queue, exchange, routingKey string) error {

	client.logger.Info().Msgf("Binding queue: %s\n", queue)
	err := ch.QueueBind(
		queue,      // queue name
		routingKey, // routing key
		exchange,   // exchange
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		return err
	}
	return nil
}

// UnbindQueue unbinds a queue from an exchange
func (client *RabbitClient) UnbindQueue(ch *amqp.Channel, queue, exchange, routingKey string) error {
	err := ch.QueueUnbind(
		queue,      // queue name
		routingKey, // routing key
		exchange,   // exchange
		nil,        // arguments
	)
	if err != nil {
		return err
	}
	return nil
}

// DeleteQueue deletes a queue
func (client *RabbitClient) DeleteQueue(ch *amqp.Channel, queue string) error {
	_, err := ch.QueueDelete(
		queue, // name
		false, // ifUnused
		false, // ifEmpty
		false, // noWait
	)
	if err != nil {
		return err
	}
	return nil
}

// DeleteExchange deletes an exchange
func (client *RabbitClient) DeleteExchange(ch *amqp.Channel, exchange string) error {
	err := ch.ExchangeDelete(
		exchange, // name
		false,    // ifUnused
		false,    // noWait
	)
	if err != nil {
		return err
	}
	return nil
}

// PurgeQueue purges a queue
func (client *RabbitClient) PurgeQueue(ch *amqp.Channel, queue string) error {
	_, err := ch.QueuePurge(
		queue, // name
		false, // noWait
	)
	if err != nil {
		return err
	}
	return nil
}

// CreateQueueWithBinding creates a queue with a binding (if exists not)
func (client *RabbitClient) CreateQueueWithBinding(ch *amqp.Channel, queue string, exchange string, kind string, key string) error {

	if queue != "" {
		err := client.QueueDeclare(ch, queue)
		if err != nil {
			customErr := fmt.Errorf("error declaring queue: %s", err)
			return customErr
		}
	}

	if exchange != "" {
		err := client.ExchangeDeclare(ch, exchange, kind)
		if err != nil {
			customErr := fmt.Errorf("error declaring exchange: %s", err)
			return customErr
		}
	}

	if key != "" {
		err := client.BindQueue(ch, queue, exchange, key)
		if err != nil {
			customErr := fmt.Errorf("error binding queue: %s", err)
			return customErr
		}
	}
	return nil

}

// CheckQueue checks if a queue exists
func (client *RabbitClient) CheckQueue(ch *amqp.Channel, queue string) (*amqp.Queue, error) {
	q, err := ch.QueueDeclarePassive(
		queue, // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return nil, err
	}
	return &q, nil
}

// CheckExchange checks if an exchange exists
func (client *RabbitClient) CheckExchange(ch *amqp.Channel, exchange string, kind string) error {
	err := ch.ExchangeDeclarePassive(
		exchange, // name
		kind,     // type
		false,    // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		return err
	}
	return nil
}

// GenericMessage is a type alias for a map of strings to interfaces
type GenericMessage map[string]any

// GenericSerialize will convert a message to a byte slice
func GenericSerialize(msg GenericMessage) ([]byte, error) {
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	err := encoder.Encode(msg)
	return b.Bytes(), err
}

// GenericDeserialize will convert a byte slice to a message
func GenericDeserialize(b []byte) (GenericMessage, error) {
	var msg GenericMessage
	buf := bytes.NewBuffer(b)
	decoder := json.NewDecoder(buf)
	err := decoder.Decode(&msg)
	return msg, err
}

// Serialize will convert a message to a byte slice
func Serialize(msg any) ([]byte, error) {
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	err := encoder.Encode(msg)
	return b.Bytes(), err
}

// Deserialize will convert a byte slice to a message
func Deserialize(b []byte) (any, error) {
	var msg any
	buf := bytes.NewBuffer(b)
	decoder := json.NewDecoder(buf)
	err := decoder.Decode(&msg)
	return msg, err
}
