package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Port     string
	GRPCPort string
	DB       DBConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Log      LogConfig
	Tracing  TracingConfig
}

type DBConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	URL string
	DB  int
}

type JWTConfig struct {
	Secret              string
	AccessExpireMinutes int
	RefreshExpireDays   int
}

type LogConfig struct {
	Level string
}

type TracingConfig struct {
	JaegerEndpoint string
	ServiceName    string
}

func Load() (*Config, error) {
	viper.AutomaticEnv()

	viper.SetDefault("PORT", "8081")
	viper.SetDefault("GRPC_PORT", "9081")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("JWT_ACCESS_EXPIRE_MINUTES", 60)
	viper.SetDefault("JWT_REFRESH_EXPIRE_DAYS", 7)
	viper.SetDefault("DB_MAX_OPEN_CONNS", 25)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 5)
	viper.SetDefault("DB_CONN_MAX_LIFETIME_SECONDS", 300)

	return &Config{
		Port:     viper.GetString("PORT"),
		GRPCPort: viper.GetString("GRPC_PORT"),
		DB: DBConfig{
			URL:             viper.GetString("DATABASE_URL"),
			MaxOpenConns:    viper.GetInt("DB_MAX_OPEN_CONNS"),
			MaxIdleConns:    viper.GetInt("DB_MAX_IDLE_CONNS"),
			ConnMaxLifetime: time.Duration(viper.GetInt("DB_CONN_MAX_LIFETIME_SECONDS")) * time.Second,
		},
		Redis: RedisConfig{
			URL: viper.GetString("REDIS_URL"),
			DB:  viper.GetInt("REDIS_DB"),
		},
		JWT: JWTConfig{
			Secret:              viper.GetString("JWT_SECRET"),
			AccessExpireMinutes: viper.GetInt("JWT_ACCESS_EXPIRE_MINUTES"),
			RefreshExpireDays:   viper.GetInt("JWT_REFRESH_EXPIRE_DAYS"),
		},
		Log: LogConfig{
			Level: viper.GetString("LOG_LEVEL"),
		},
		Tracing: TracingConfig{
			JaegerEndpoint: viper.GetString("JAEGER_ENDPOINT"),
			ServiceName:    "auth-service",
		},
	}, nil
}
