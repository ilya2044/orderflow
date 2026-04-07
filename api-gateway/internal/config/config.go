package config

import "github.com/spf13/viper"

type Config struct {
	Port     string
	Services ServicesConfig
	Redis    RedisConfig
	Log      LogConfig
	Tracing  TracingConfig
	RateLimit RateLimitConfig
}

type ServicesConfig struct {
	AuthURL         string
	UserURL         string
	ProductURL      string
	OrderURL        string
	InventoryURL    string
	PaymentURL      string
	NotificationURL string
}

type RedisConfig struct {
	URL string
}

type LogConfig struct {
	Level string
}

type TracingConfig struct {
	JaegerEndpoint string
	ServiceName    string
}

type RateLimitConfig struct {
	RequestsPerSecond int
	BurstSize         int
}

func Load() (*Config, error) {
	viper.AutomaticEnv()

	viper.SetDefault("PORT", "8080")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("RATE_LIMIT_RPS", 100)
	viper.SetDefault("RATE_LIMIT_BURST", 200)

	return &Config{
		Port: viper.GetString("PORT"),
		Services: ServicesConfig{
			AuthURL:         viper.GetString("AUTH_SERVICE_URL"),
			UserURL:         viper.GetString("USER_SERVICE_URL"),
			ProductURL:      viper.GetString("PRODUCT_SERVICE_URL"),
			OrderURL:        viper.GetString("ORDER_SERVICE_URL"),
			InventoryURL:    viper.GetString("INVENTORY_SERVICE_URL"),
			PaymentURL:      viper.GetString("PAYMENT_SERVICE_URL"),
			NotificationURL: viper.GetString("NOTIFICATION_SERVICE_URL"),
		},
		Redis: RedisConfig{
			URL: viper.GetString("REDIS_URL"),
		},
		Log: LogConfig{
			Level: viper.GetString("LOG_LEVEL"),
		},
		Tracing: TracingConfig{
			JaegerEndpoint: viper.GetString("JAEGER_ENDPOINT"),
			ServiceName:    "api-gateway",
		},
		RateLimit: RateLimitConfig{
			RequestsPerSecond: viper.GetInt("RATE_LIMIT_RPS"),
			BurstSize:         viper.GetInt("RATE_LIMIT_BURST"),
		},
	}, nil
}
