package queue

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// KafkaProducer represents a producer
type KafkaProducer struct {
	Logger   *zerolog.Logger
	Config   *sarama.Config
	Producer sarama.SyncProducer
}

// KafkaAsyncProducer represents a async producer
type KafkaAsyncProducer struct {
	Logger   *zerolog.Logger
	Config   *sarama.Config
	Producer sarama.AsyncProducer
}

// NewProducerConfig creates a new Sarama configuration for a producer
func NewProducerConfig(
	requiredAcks sarama.RequiredAcks,
	clientId string,
	returnSuccess bool,
	// partitioner string,
) *sarama.Config {

	// Parse Kafka version
	version, err := sarama.ParseKafkaVersion(KAFKA_VERSION)
	if err != nil {
		panic(err)
	}

	config := sarama.NewConfig()
	config.Version = version

	if clientId == "" {
		clientId = uuid.New().String()
		config.ClientID = clientId
	}

	config.Producer.RequiredAcks = requiredAcks
	config.Producer.Return.Successes = returnSuccess
	config.Producer.Partitioner = sarama.NewHashPartitioner

	/*
		switch partitioner {
		case "":
			if partition >= 0 {
				config.Producer.Partitioner = sarama.NewManualPartitioner
			} else {
				config.Producer.Partitioner = sarama.NewHashPartitioner
			}
		case "hash":
			config.Producer.Partitioner = sarama.NewHashPartitioner
		case "random":
			config.Producer.Partitioner = sarama.NewRandomPartitioner
		case "manual":
			config.Producer.Partitioner = sarama.NewManualPartitioner
			if *partition == -1 {
				printUsageErrorAndExit("-partition is required when partitioning manually")
			}
		default:
			config.Producer.Partitioner = sarama.NewHashPartitioner
		}
	*/

	config.Net.SASL.User = "admin"
	config.Net.SASL.Password = "admin"
	config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	config.Net.SASL.Enable = true

	return config
}

// NewAsyncProducerConfig creates a new Sarama configuration for a async producer
func NewAsyncProducerConfig(
	requiredAcks sarama.RequiredAcks,
	clientId string,
	returnSuccess bool,
	// partitioner string,
) *sarama.Config {

	// Parse Kafka version
	version, err := sarama.ParseKafkaVersion("7.3.8")
	if err != nil {
		panic(err)
	}

	config := sarama.NewConfig()
	config.Version = version

	if clientId == "" {
		clientId = uuid.New().String()
		config.ClientID = clientId
	}

	config.Producer.RequiredAcks = requiredAcks
	config.Producer.Return.Successes = returnSuccess
	config.Producer.Partitioner = sarama.NewHashPartitioner

	config.Net.SASL.User = "admin"
	config.Net.SASL.Password = "admin"
	config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	config.Net.SASL.Enable = true

	return config
}

// NewKafkaProducer creates a producer
func NewKafkaProducer(
	brokerList string,
	config *sarama.Config,
	logger *zerolog.Logger,
) (*KafkaProducer, error) {

	// Set default config if not provided
	if config == nil {
		config = NewProducerConfig(sarama.WaitForAll, "", true)
	}

	// Init producer
	producer, err := sarama.NewSyncProducer(strings.Split(brokerList, ","), config)
	if err != nil {
		return nil, err
	}

	return &KafkaProducer{
		Config:   config,
		Producer: producer,
		Logger:   logger,
	}, nil
}

// NewKafkaAsyncProducer creates a async producer for streaming messages
func NewKafkaAsyncProducer(
	brokerList string,
	config *sarama.Config,
	logger *zerolog.Logger,
) (*KafkaAsyncProducer, error) {

	// Set default config if not provided
	if config == nil {
		config = NewProducerConfig(sarama.WaitForAll, "", true)
	}

	// Init producer
	producer, err := sarama.NewAsyncProducer(strings.Split(brokerList, ","), config)
	if err != nil {
		return nil, err
	}

	return &KafkaAsyncProducer{
		Config:   config,
		Producer: producer,
		Logger:   logger,
	}, nil
}

// BuildMessage creates a new message with key, value, topic and headers and uses stdin if value is empty
// If headers is provided, it will be parsed and added to the message
// Build Messages encodes the key and value as strings
func (p *KafkaProducer) BuildMessage(key string, value string, topic string, headers string) (*sarama.ProducerMessage, error) {
	message := &sarama.ProducerMessage{Topic: topic}

	if key != "" {
		message.Key = sarama.StringEncoder(key)
	}

	if value != "" {
		message.Value = sarama.StringEncoder(value)
	} else if stdinAvailable() {
		bytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}
		message.Value = sarama.ByteEncoder(bytes)
	} else {
		return nil, fmt.Errorf("-value is required, or you have to provide the value on stdin")
	}

	if headers != "" {
		var hdrs []sarama.RecordHeader
		arrHdrs := strings.Split(headers, ",")
		for _, h := range arrHdrs {
			if header := strings.Split(h, ":"); len(header) != 2 {
				return nil, fmt.Errorf("-value is required, or you have to provide the value on stdin")
			} else {
				hdrs = append(hdrs, sarama.RecordHeader{
					Key:   []byte(header[0]),
					Value: []byte(header[1]),
				})
			}
		}

		if len(hdrs) != 0 {
			message.Headers = hdrs
		}
	}

	return message, nil
}

// SendMessage sends a message to a Kafka topic
func (p *KafkaProducer) SendMessage(message *sarama.ProducerMessage, silent bool) error {

	partition, offset, err := p.Producer.SendMessage(message)
	if err != nil {
		return err
	} else if !silent {
		p.Logger.Info().Msgf("topic=%s\tpartition=%d\toffset=%d\n", message.Topic, partition, offset)
	}
	return nil
}

// Close closes the producer
func (p *KafkaProducer) Close() {
	if err := p.Producer.Close(); err != nil {
		p.Logger.Error().Msgf("Failed to close producer: %s", err)
	}
}

// BuildMessage creates a new message with key, value, topic and headers and uses stdin if value is empty
// If headers is provided, it will be parsed and added to the message
// Build Messages encodes the key and value as strings
func (p *KafkaAsyncProducer) BuildMessage(key string, value string, topic string, headers string) (*sarama.ProducerMessage, error) {
	message := &sarama.ProducerMessage{Topic: topic}

	if key != "" {
		message.Key = sarama.StringEncoder(key)
	}

	if value != "" {
		message.Value = sarama.StringEncoder(value)
	} else if stdinAvailable() {
		bytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}
		message.Value = sarama.ByteEncoder(bytes)
	} else {
		return nil, fmt.Errorf("-value is required, or you have to provide the value on stdin")
	}

	if headers != "" {
		var hdrs []sarama.RecordHeader
		arrHdrs := strings.Split(headers, ",")
		for _, h := range arrHdrs {
			if header := strings.Split(h, ":"); len(header) != 2 {
				return nil, fmt.Errorf("-value is required, or you have to provide the value on stdin")
			} else {
				hdrs = append(hdrs, sarama.RecordHeader{
					Key:   []byte(header[0]),
					Value: []byte(header[1]),
				})
			}
		}

		if len(hdrs) != 0 {
			message.Headers = hdrs
		}
	}

	return message, nil
}

// SendMessage sends a message to a Kafka topic
func (p *KafkaAsyncProducer) SendMessage(message *sarama.ProducerMessage, silent bool) {

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	for {
		select {
		case p.Producer.Input() <- message:
			p.Logger.Info().Msgf("topic=%s\tpartition=%d\toffset=%d\n", message.Topic, message.Partition, message.Offset)
		case <-signals:
			p.Producer.AsyncClose() // Trigger a shutdown of the producer.
			return
		}
	}
}

// Close closes the producer
func (p *KafkaAsyncProducer) Close() {
	if err := p.Producer.Close(); err != nil {
		p.Logger.Error().Msgf("Failed to close producer: %s", err)
	}
}

// check if stdin is available
func stdinAvailable() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}
