package config

// serverConfig holds the configuration for the server.
type serverConfig struct {
	Host string `mapstructure:"APP_HOST"`
	Port int    `mapstructure:"APP_PORT"`
}
