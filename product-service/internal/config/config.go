package config

import "github.com/spf13/viper"

type Config struct {
	Port          string
	Mongo         MongoConfig
	Elasticsearch ElasticsearchConfig
	Redis         RedisConfig
	Minio         MinioConfig
	Log           LogConfig
}

type MongoConfig struct {
	URL      string
	Database string
}

type ElasticsearchConfig struct {
	URL string
}

type RedisConfig struct {
	URL string
}

type MinioConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type LogConfig struct {
	Level string
}

func Load() (*Config, error) {
	viper.AutomaticEnv()

	viper.SetDefault("PORT", "8083")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("MONGO_DATABASE", "product_db")
	viper.SetDefault("MINIO_BUCKET", "products")
	viper.SetDefault("MINIO_USE_SSL", false)

	return &Config{
		Port: viper.GetString("PORT"),
		Mongo: MongoConfig{
			URL:      viper.GetString("MONGO_URL"),
			Database: viper.GetString("MONGO_DATABASE"),
		},
		Elasticsearch: ElasticsearchConfig{
			URL: viper.GetString("ELASTICSEARCH_URL"),
		},
		Redis: RedisConfig{
			URL: viper.GetString("REDIS_URL"),
		},
		Minio: MinioConfig{
			Endpoint:  viper.GetString("MINIO_ENDPOINT"),
			AccessKey: viper.GetString("MINIO_ACCESS_KEY"),
			SecretKey: viper.GetString("MINIO_SECRET_KEY"),
			Bucket:    viper.GetString("MINIO_BUCKET"),
			UseSSL:    viper.GetBool("MINIO_USE_SSL"),
		},
		Log: LogConfig{
			Level: viper.GetString("LOG_LEVEL"),
		},
	}, nil
}
