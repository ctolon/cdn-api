package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// NatsClient is a wrapper for the NATS client
type NatsClient struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

// NewNatsClient creates a new NATS client
func NewNatsClient(jetstream bool) (*NatsClient, error) {

	nc, err := nats.Connect("nats://nats:4222")
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	var jsCtx nats.JetStreamContext
	if jetstream {
		jsCtx, err = nc.JetStream()
		if err != nil {
			return nil, fmt.Errorf("jetstream: %w", err)
		}
	}

	return &NatsClient{nc: nc, js: jsCtx}, nil
}

// CreateJetStream creates a new stream
func (n *NatsClient) CreateJetStream(ctx context.Context) (*nats.StreamInfo, error) {

	if n.js == nil {
		return nil, errors.New("jetstream is not enabled")
	}

	stream, err := n.js.AddStream(&nats.StreamConfig{
		Name:              "test_stream",
		Subjects:          []string{"subject.1", "subject.2", "subject.N"},
		Retention:         nats.InterestPolicy, // remove acked messages
		Discard:           nats.DiscardOld,     // when the stream is full, discard old messages
		MaxAge:            7 * 24 * time.Hour,  // max age of stored messages is 7 days
		Storage:           nats.FileStorage,    // type of message storage
		MaxMsgsPerSubject: 100_000_000,         // max stored messages per subject
		MaxMsgSize:        4 << 20,             // max single message size is 4 MB
		NoAck:             false,               // we need the "ack" system for the message queue system
	}, nats.Context(ctx))
	if err != nil {
		return nil, fmt.Errorf("add stream: %w", err)
	}

	return stream, nil
}

// PublishMsgWithCtx publishes a message to a subject with context (for long running processes in goroutines)
func (n *NatsClient) PublishMsgWithCtx(ctx context.Context, subject string, payload []byte) error {

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			err := n.nc.Publish(subject, payload)
			if err != nil {
				return fmt.Errorf("publish: %w", err)
			}
		}
	}
}

// PublishMsg publishes a message to a subject
func (n *NatsClient) PublishMsg(subject string, payload []byte) error {
	err := n.nc.Publish(subject, payload)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return nil
}

// CreateConsumer creates a new consumer
func (n *NatsClient) CreateConsumer(ctx context.Context, consumerGroupName, streamName string) (*nats.ConsumerInfo, error) {
	if n.js == nil {
		return nil, errors.New("jetstream is not enabled")
	}

	consumer, err := n.js.AddConsumer(streamName, &nats.ConsumerConfig{
		Durable:       consumerGroupName,      // durable name is the same as consumer group name
		DeliverPolicy: nats.DeliverAllPolicy,  // deliver all messages, even if they were sent before the consumer was created
		AckPolicy:     nats.AckExplicitPolicy, // ack messages manually
		AckWait:       5 * time.Second,        // wait for ack for 5 seconds
		MaxAckPending: -1,                     // unlimited number of pending acks
	}, nats.Context(ctx))
	if err != nil {
		return nil, fmt.Errorf("add consumer: %w", err)
	}

	return consumer, nil
}

// PullSubscribe creates a new subscription on jetstream
func (n *NatsClient) PullSubscribe(ctx context.Context, subject, consumerGroupName, streamName string) (*nats.Subscription, error) {
	if n.js == nil {
		return nil, errors.New("jetstream is not enabled")
	}

	pullSub, err := n.js.PullSubscribe(
		subject,
		consumerGroupName,
		nats.ManualAck(),                         // ack messages manually
		nats.Bind(streamName, consumerGroupName), // bind consumer to the stream
		nats.Context(ctx),                        // use context to cancel the subscription
	)
	if err != nil {
		return nil, fmt.Errorf("pull subscribe: %w", err)
	}

	return pullSub, nil
}

// Subscribe creates a new subscription
func (n *NatsClient) Subscribe(subject string, cb nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := n.nc.Subscribe(subject, cb)
	if err != nil {
		return nil, fmt.Errorf("subscribe: %w", err)
	}
	return sub, nil
}

// ChanSubscribe creates a new subscription with a channel
func (n *NatsClient) ChanSubscribe(subject string) (<-chan *nats.Msg, *nats.Subscription, error) {
	ch := make(chan *nats.Msg)
	sub, err := n.nc.ChanSubscribe(subject, ch)
	if err != nil {
		return nil, nil, fmt.Errorf("chan subscribe: %w", err)
	}
	return ch, sub, nil
}

// AckMsg acknowledges a message
func (n *NatsClient) AckMsg(msg *nats.Msg) error {
	err := msg.Ack()
	if err != nil {
		return fmt.Errorf("ack: %w", err)
	}
	return nil
}

func (n *NatsClient) NackMsg(msg *nats.Msg) error {
	err := msg.Nak()
	if err != nil {
		return fmt.Errorf("nack: %w", err)
	}
	return nil
}

// FetchOne fetches one message from the subscription
func (n *NatsClient) FetchOne(ctx context.Context, pullSub *nats.Subscription) (*nats.Msg, error) {
	return fetchOne(ctx, pullSub)
}

// Close closes the NATS client
func (n *NatsClient) Close() {
	n.nc.Close()
}

// fetchOne fetches one message from the subscription
func fetchOne(ctx context.Context, pullSub *nats.Subscription) (*nats.Msg, error) {
	msgs, err := pullSub.Fetch(1, nats.Context(ctx))
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	if len(msgs) == 0 {
		return nil, errors.New("no messages")
	}

	return msgs[0], nil
}

func SetReconnectionHandler(nc *nats.Conn) {
	nc.SetReconnectHandler(func(nc *nats.Conn) {
		// Update the connection in the publisher and consumers
	})
}
