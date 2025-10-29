package config

type Config struct {
	Port             string   `env:"PORT" default:"8080"`
	DatabaseURL      string   `env:"DATABASE_URL" required:"true"`
	RedisAddr        string   `env:"REDIS_ADDR" default:"localhost:6379"`
	VAPIDPublicKey   string   `env:"VAPID_PUBLIC_KEY" required:"true"`
	VAPIDPrivateKey  string   `env:"VAPID_PRIVATE_KEY" required:"true"`
	HMACSecret       string   `env:"HMAC_SECRET" required:"true"`
	LogLevel         string   `env:"LOG_LEVEL" default:"info"`
	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS"`
}
