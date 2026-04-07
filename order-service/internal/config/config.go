package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Port         string
	DB           DBConfig
	Kafka        KafkaConfig
	Redis        RedisConfig
	Log          LogConfig
	Services     ServicesConfig
}

type DBConfig struct {
	URL          string
	MaxOpenConns int
}

type KafkaConfig struct {
	Brokers []string
}

type RedisConfig struct {
	URL string
}

type LogConfig struct {
	Level string
}

type ServicesConfig struct {
	InventoryURL string
	PaymentURL   string
}

func Load() (*Config, error) {
	viper.AutomaticEnv()

	viper.SetDefault("PORT", "8084")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("DB_MAX_OPEN_CONNS", 25)

	kafkaBrokers := viper.GetString("KAFKA_BROKERS")
	brokers := strings.Split(kafkaBrokers, ",")

	return &Config{
		Port: viper.GetString("PORT"),
		DB: DBConfig{
			URL:          viper.GetString("DATABASE_URL"),
			MaxOpenConns: viper.GetInt("DB_MAX_OPEN_CONNS"),
		},
		Kafka: KafkaConfig{
			Brokers: brokers,
		},
		Redis: RedisConfig{
			URL: viper.GetString("REDIS_URL"),
		},
		Log: LogConfig{
			Level: viper.GetString("LOG_LEVEL"),
		},
		Services: ServicesConfig{
			InventoryURL: viper.GetString("INVENTORY_SERVICE_URL"),
			PaymentURL:   viper.GetString("PAYMENT_SERVICE_URL"),
		},
	}, nil
}
