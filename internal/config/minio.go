package config

// minioConfig holds the configuration for minio.
type minioConfig struct {
	MinioAccessKey string `mapstructure:"MINIO_ACCESS_KEY"`
	MinioSecretKey string `mapstructure:"MINIO_SECRET_KEY"`
	MinioEndpoint  string `mapstructure:"MINIO_ENDPOINT"`
}
