package config

import (
	// "sync"

	"github.com/spf13/viper"

	"github.com/ctolon/cdn-api/internal/app/providers"
	"github.com/ctolon/cdn-api/internal/utils"
)

// var configOnce sync.Once
// var configLoadError error

// AppConfig represents the application configuration
type AppConfig struct {
	loggerConfig   `mapstructure:",squash"`
	minioConfig    `mapstructure:",squash"`
	serverConfig   `mapstructure:",squash"`
	redisConfig    `mapstructure:",squash"`
	rabbitmqConfig `mapstructure:",squash"`
	kafkaConfig    `mapstructure:",squash"`
}

// GlobalAppConfig represents the application configuration
var GlobalAppConfig AppConfig

// LoadConfig loads configuration from the specified provider
func LoadConfig(cfgProvider providers.ConfigProvider) {

	var configuration any
	cfgProvider.LoadConfig()

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	if err := viper.Unmarshal(&configuration); err != nil {
		panic(err)
	}

	if err := utils.ValidateMultipleStructs(configuration); err != nil {
		panic(err)
	}
}

// LoadConfigIntoApp loads configuration from the specified provider into the application configuration
func LoadConfigIntoApp(cfgProvider providers.ConfigProvider) {

	cfgProvider.LoadConfig()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			panic(err)
		} else {
			// Config file was found but another error was produced
			panic(err)
		}
	}

	if err := viper.Unmarshal(&GlobalAppConfig); err != nil {
		panic(err)
	}

	if err := utils.ValidateMultipleStructs(GlobalAppConfig); err != nil {
		panic(err)
	}
}

func LoadConfigIntoStruct(cfgProvider providers.ConfigProvider) *AppConfig {

	cfgProvider.LoadConfig()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			panic(err)
		} else {
			// Config file was found but another error was produced
			panic(err)
		}
	}

	var configuration *AppConfig
	if err := viper.Unmarshal(&configuration); err != nil {
		panic(err)
	}

	if err := utils.ValidateMultipleStructs(configuration); err != nil {
		panic(err)
	}

	return configuration
}

// LoadConfigFromViper loads configuration from the specified provider
func LoadConfigFromViper(cfgProvider providers.ConfigProvider, v *viper.Viper) *viper.Viper {

	v2, err := cfgProvider.LoadConfigWithNewViper(v)
	if err != nil {
		panic(err)
	}

	if err := v2.ReadInConfig(); err != nil {
		panic(err)
	}

	return v2
}

// ReadConfig reads configuration from the specified provider
func ReadConfig(cfgProvider providers.ConfigProvider) {

	cfgProvider.LoadConfig()

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
}

// ReadConfigFromViper reads configuration from the specified provider
func ReadConfigFromViper(cfgProvider providers.ConfigProvider, v *viper.Viper) *viper.Viper {

	v2, err := cfgProvider.LoadConfigWithNewViper(v)
	if err != nil {
		panic(err)
	}

	if err := v2.ReadInConfig(); err != nil {
		panic(err)
	}

	return v2

}
