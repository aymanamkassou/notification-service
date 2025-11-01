package logger

import (
	"fmt"

	"go.uber.org/zap"
)

// New creates a zap logger based on the log level (debug, info, warn, error).
func New(level string) (*zap.Logger, error) {
	var cfg zap.Config
	switch level {
	case "debug":
		cfg = zap.NewDevelopmentConfig()
	case "info", "warn", "error":
		cfg = zap.NewProductionConfig()
		cfg.Level = parseLevel(level)
	default:
		return nil, fmt.Errorf("invalid log level: %s", level)
	}

	cfg.DisableStacktrace = true // Disable stacktraces for cleaner logs
	cfg.Encoding = "json"

	logger, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return logger, nil
}

func parseLevel(level string) zap.AtomicLevel {
	switch level {
	case "debug":
		return zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		return zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		return zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		return zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		return zap.NewAtomicLevelAt(zap.InfoLevel)
	}
}
