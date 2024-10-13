package config

// kafkaConfig holds the configuration for kafka.
type kafkaConfig struct {
	KafkaURL string `mapstructure:"KAFKA_BOOTSTRAP_SERVERS"`
}
