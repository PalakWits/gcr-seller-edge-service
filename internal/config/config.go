package config

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	DatabaseURL        string `envconfig:"DATABASE_URL" required:"true"`
	Port               string `envconfig:"PORT" default:"8080"`
	LogLevel           string `envconfig:"LOG_LEVEL" default:"info"`
	MinIOEndpoint      string `envconfig:"MINIO_ENDPOINT" required:"true"`
	MinIOAccessKey     string `envconfig:"MINIO_ACCESS_KEY" required:"true"`
	MinIOSecretKey     string `envconfig:"MINIO_SECRET_KEY" required:"true"`
	MinIOUseSSL        bool   `envconfig:"MINIO_USE_SSL" default:"false"`
	MinIOBucket        string `envconfig:"MINIO_BUCKET" default:"ondc-payloads"`
	KafkaBrokers       string `envconfig:"KAFKA_BROKERS" required:"true"`
	KafkaOnSearchTopic string `envconfig:"KAFKA_ON_SEARCH_TOPIC" default:"ondc.on_search.pointer"`
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("Warning: error loading .env file: %v\n", err)
	}

	config := &Config{}

	err = envconfig.Process("", config)
	if err != nil {
		return nil, fmt.Errorf("error processing envconfig: %w", err)
	}

	return config, nil
}
