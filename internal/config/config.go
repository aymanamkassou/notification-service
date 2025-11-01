package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Port               string   `envconfig:"PORT" default:"8080"`
	DatabaseURL        string   `envconfig:"DATABASE_URL" required:"true"`
	RedisAddr          string   `envconfig:"REDIS_ADDR" default:"localhost:6379"`
	VAPIDPublicKey     string   `envconfig:"VAPID_PUBLIC_KEY" required:"true"`
	VAPIDPrivateKey    string   `envconfig:"VAPID_PRIVATE_KEY" required:"true"`
	HMACSecret         string   `envconfig:"HMAC_SECRET" required:"true"`
	LogLevel           string   `envconfig:"LOG_LEVEL" default:"info"`
	CORSAllowedOrigins []string `envconfig:"CORS_ALLOWED_ORIGINS"`
}

// Load reads config from environment variables with validation.
func Load() (*Config, error) {
	var cfg Config
	// Use empty prefix so envconfig reads exact environment variable names
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &cfg, nil
}
