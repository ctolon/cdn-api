package config

// rabbitmqConfig holds the configuration for rabbitmq.
type rabbitmqConfig struct {
	RabbitURL string `mapstructure:"RABBIT_URL"`
}
