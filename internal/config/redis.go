package config

// redisConfig holds the configuration for redis.
type redisConfig struct {
	RedisURL string `mapstructure:"REDIS_URL"`
}
