package queue

import (
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// KafkaPartitionedConsumer represents a consumer with workers
type KafkaPartitionedConsumer struct {
	Logger     *zerolog.Logger
	Config     *sarama.Config
	Topics     []string
	Partitions []int32
	Consumer   sarama.Consumer
	Messages   chan *sarama.ConsumerMessage
	Closing    chan struct{}
}

// NewKafkaPartitionedConsumerConfig creates a new Sarama configuration for a consumer
func NewKafkaPartitionedConsumerConfig(
	autoCommit bool,
	clientId string,
	user string,
	password string,
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

	// config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Offsets.AutoCommit.Enable = autoCommit
	config.ChannelBufferSize = 1000000

	if user == "" {
		user = "admin"
	}
	if password == "" {
		password = "admin"
	}
	config.Net.SASL.User = user
	config.Net.SASL.Password = password
	config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	config.Net.SASL.Enable = true

	return config
}

// NewKafkaPartitionedConsumer creates a new consumer with workers
func NewKafkaPartitionedConsumer(
	brokerList string,
	config *sarama.Config,
	topics []string,
	partitions string,
	logger *zerolog.Logger,
) (*KafkaPartitionedConsumer, error) {

	// Create channels
	var (
		messages = make(chan *sarama.ConsumerMessage, 1024)
		closing  = make(chan struct{})
	)

	// Set default config if not provided
	if config == nil {
		config = NewKafkaPartitionedConsumerConfig(false, "", "", "")
	}

	// Init consumer
	c, err := sarama.NewConsumer(strings.Split(brokerList, ","), config)
	if err != nil {
		return nil, err
	}

	// Get partitions of topics
	var pList []int32
	if partitions == "all" {
		pList, err = getAllPartitions(c, topics)
		if err != nil {
			return nil, err
		}
	} else {
		pList, err = getPartitions(partitions)
		if err != nil {
			return nil, err
		}
	}

	config.ChannelBufferSize = 1000000
	return &KafkaPartitionedConsumer{
		Config:     config,
		Topics:     topics,
		Partitions: pList,
		Consumer:   c,
		Messages:   messages,
		Closing:    closing,
		Logger:     logger,
	}, nil
}

// SetupSignalCh sets up signal channel for graceful shutdown
func (c *KafkaPartitionedConsumer) SetupSignalCh() {
	c.Logger.Info().Msg("Setting up signal channel for graceful shutdown...")
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, os.Interrupt)
	<-signals
	c.Logger.Println("Initiating shutdown of consumer...")
	close(c.Closing)
}

// StartWithWorkers starts the consumer with workers
// and returns a channel to receive messages
func (c *KafkaPartitionedConsumer) StartConsumeCh(wg *sync.WaitGroup) error {
	topics := c.Topics[0]
	c.Logger.Info().Msg("Starting consumer with workers...")
	for _, partition := range c.Partitions {
		pc, err := c.Consumer.ConsumePartition(topics, partition, c.Config.Consumer.Offsets.Initial)
		if err != nil {
			return err
		}

		go func(pc sarama.PartitionConsumer) {
			<-c.Closing
			pc.AsyncClose()
		}(pc)

		wg.Add(1)
		go func(pc sarama.PartitionConsumer) {
			defer wg.Done()
			for message := range pc.Messages() {
				c.Messages <- message
			}
		}(pc)
	}

	return nil

	/*
		// Main goroutine to consume messages
		// Consume messages
		go func() {
			for msg := range c.Messages {
				fmt.Printf("Partition:\t%d\n", msg.Partition)
				fmt.Printf("Offset:\t%d\n", msg.Offset)
				fmt.Printf("Key:\t%s\n", string(msg.Key))
				fmt.Printf("Value:\t%s\n", string(msg.Value))
			}
		}()

		wg.Wait()
		logger.Println("Done consuming topic", "python.kafka.benchmarkkk.sync")
		close(c.Messages)

		if err := c.Consumer.Close(); err != nil {
			logger.Println("Failed to close consumer: ", err)
		}
	*/
}

// getAllPartitions returns a list of all partitions for topics
func getAllPartitions(c sarama.Consumer, topics []string) ([]int32, error) {

	var partitions []int32
	for _, topic := range topics {
		plist, err := c.Partitions(topic)
		if err != nil {
			return nil, err
		}
		partitions = append(partitions, plist...)
	}
	return partitions, nil
}

// getPartitions returns a list of partitions as provided
func getPartitions(partitions string) ([]int32, error) {

	tmp := strings.Split(partitions, ",")
	var pList []int32
	for i := range tmp {
		val, err := strconv.ParseInt(tmp[i], 10, 32)
		if err != nil {
			return nil, err
		}
		pList = append(pList, int32(val))
	}

	return pList, nil
}

// **** //

// Consumer represents a Sarama consumer group consumer
type KafkaConsumer struct {
	Ready  chan bool
	Config *sarama.Config
	Logger *zerolog.Logger
	Client sarama.ConsumerGroup
	Topics []string
}

// NewConsumerConfig creates a new Sarama configuration for a consumer
func NewKafkaConsumerConfig(
	autoCommit bool,
	clientId string,
) (*sarama.Config, error) {

	// Parse Kafka version
	version, err := sarama.ParseKafkaVersion("7.3.8")
	if err != nil {
		return nil, err
	}

	config := sarama.NewConfig()
	config.Version = version

	if clientId == "" {
		clientId = uuid.New().String()
		config.ClientID = clientId
	}

	// config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Offsets.AutoCommit.Enable = autoCommit

	config.Consumer.Fetch.Default = 1024 * 1024 * 20
	config.Consumer.MaxWaitTime = 500 * time.Millisecond
	config.ChannelBufferSize = 256 * 4

	config.Net.SASL.User = "admin"
	config.Net.SASL.Password = "admin"
	config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	config.Net.SASL.Enable = true

	return config, nil
}

// NewConsumer creates a new Sarama consumer group consumer
func NewKafkaConsumer(
	config *sarama.Config,
	logger *zerolog.Logger,
	brokerList string,
	groupId string,
	topics []string,
) (*KafkaConsumer, error) {

	if config == nil {
		var err error
		config, err = NewKafkaConsumerConfig(false, "")
		if err != nil {
			return nil, err
		}
	}

	client, err := sarama.NewConsumerGroup(strings.Split(brokerList, ","), groupId, config)
	if err != nil {
		return nil, err
	}

	return &KafkaConsumer{
		Ready:  make(chan bool),
		Logger: logger,
		Config: config,
		Client: client,
		Topics: topics,
	}, nil
}

// SetChannelBufferSize sets the buffer size of the consumer channel
func (consumer *KafkaConsumer) SetChannelBufferSize(size int) {
	consumer.Config.ChannelBufferSize = size
}

// GetKafkaPartitions returns a list of partitions for a topic
func GetKafkaPartitions(c sarama.Consumer, topic string) ([]int32, error) {
	return c.Partitions(topic)
}
